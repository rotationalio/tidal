// Package model defines the [Model] interface and shared base types for CRUD stores.
//
// Implement [Model] on your structs and embed [BaseModel] for ULID ids and timestamps.
//
// Example:
//
//	type User struct {
//		model.BaseModel
//		Name  string
//		Email string
//	}
//
//	func (u *User) Fields(op model.Operation) []string {
//		return []string{"id", "name", "email", "created", "modified"}
//	}
//
//	func (u *User) Params(op model.Operation) []sql.NamedArg {
//		return []sql.NamedArg{
//			sql.Named("id", u.ID),
//			sql.Named("name", u.Name),
//			sql.Named("email", u.Email),
//			sql.Named("created", u.Created),
//			sql.Named("modified", u.Modified),
//		}
//	}
//
//	func (u *User) Scan(op model.Operation, s model.Scanner) error {
//		return s.Scan(&u.ID, &u.Name, &u.Email, &u.Created, &u.Modified)
//	}
package model

import (
	"database/sql"
	"reflect"
	"time"

	"go.rtnl.ai/tidal/errors"
	"go.rtnl.ai/tidal/fields"
	"go.rtnl.ai/ulid"
)

//============================================================================
// Model Interface
//============================================================================

// All models that use the tidal package must implement this interface.
type Model interface {
	// Used during select operations to scan the model from a database row.
	Scan(Operation, Scanner) error

	// Used during select operations to identify the fields that should be returned and
	// the order in which they should be returned. Note that during create and update
	// operations, the fields are selected from the sql.NamedArgs returned by the
	// Params method.
	Fields(Operation) []string

	// Used during insert and update operations to supply fields and their values to
	// the database. Note that the names of the parameters must match the fields used
	// in the database schema.
	Params(Operation) []sql.NamedArg
}

// Scanner is an interface for [*Row], [*sql.Row], and [*sql.Rows] so that models can
// implement how they scan fields into their struct without having to specify every
// field every time.
type Scanner interface {
	Scan(dest ...any) error
}

// Prepare is an interface for models that need to prepare their fields before being
// created or updated in the database. This is called automatically by the store before
// creating or updating a model.
type Preparer interface {
	Prepare(Operation)
}

// Validator is an interface for models that need to have non-database constraints
// validated before being created or updated -- these business rules are not enforced
// by the database, but may be mirrored by database constratints depending on the store
// type.
type Validator interface {
	Validate(Operation) error
}

//============================================================================
// Model Factory
//============================================================================

// Make returns a new zero value of M, allocating a pointer when M is a pointer type.
func Make[M Model]() M {
	var instance M

	t := reflect.TypeOf(instance)
	if t != nil && t.Kind() == reflect.Ptr {
		return reflect.New(t.Elem()).Interface().(M)
	}

	return instance
}

//============================================================================
// Base Model
//============================================================================

// BaseModel is embedded into most models to provide ID management and timestamps.
type BaseModel struct {
	ID       ulid.ULID
	Created  fields.Timestamp
	Modified fields.Timestamp
}

var _ Preparer = (*BaseModel)(nil)

// Updates the modified timestamp for the model. If creating a new record, also creates
// a new ULID for the ID field and the created timestamp. Database stores may override
// the values set in this method, see the specific store type for details.
func (b *BaseModel) Prepare(op Operation) {
	switch op {
	case Create:
		b.ID = ulid.MakeSecure()
		b.Created = fields.Time(time.Now())
		b.Modified = b.Created

	case Update:
		b.Modified.Now()
	}
}

// Validates the base model before update or create. This is called after Prepare so
// there is no point checking for timstamp zero values as they will be set by Prepare.
func (b *BaseModel) Validate(op Operation) error {
	switch op {
	case Update:
		if b.ID.IsZero() {
			return errors.ErrMissingID
		}
	}
	return nil
}

// IsZero reports whether the receiver is nil or all fields are zero.
func (b *BaseModel) IsZero() bool {
	return b == nil || (b.ID.IsZero() && b.Created.IsZero() && b.Modified.IsZero())
}

// Compare returns the result of comparing the ID ULIDs together. Generally speaking
// this method sorts base models ascending by Created timestamp but without worrying
// about database driver timestamp precision.
//
// NOTE: this is not true equality because it doesn't compare all fields, just the ID.
func (b BaseModel) Compare(other BaseModel) int {
	return b.ID.Compare(other.ID)
}

// Equal returns true if all fields are equal. Timestamp fields are compared at the
// millisecond precision for database driver compatibility.
func (b BaseModel) Equal(other BaseModel) bool {
	if b.ID != other.ID {
		return false
	}

	if b.Created.Equal(other.Created) {
		return false
	}

	if b.Modified.Equal(other.Modified) {
		return false
	}

	return true
}
