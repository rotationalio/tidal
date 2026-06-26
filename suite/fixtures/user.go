package fixtures

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"go.rtnl.ai/tidal"
	"go.rtnl.ai/tidal/fields"
	"go.rtnl.ai/ulid"
)

//============================================================================
// Test Models
//============================================================================

// User is a test [tidal.Model] for integration and conformance suites.
type User struct {
	tidal.BaseModel
	Name     string
	DOB      fields.Timestamp
	Email    string
	Password string
	Verified bool
	LastSeen fields.Timestamp
}

var _ tidal.Model = (*User)(nil)

func (u *User) Fields(op tidal.Operation) []string {
	switch op {
	case tidal.List:
		return []string{"id", "name", "email", "created", "modified"}
	default:
		return []string{"id", "name", "dob", "email", "password", "verified", "last_seen", "created", "modified"}
	}
}

func (u *User) Params(op tidal.Operation) []sql.NamedArg {
	switch op {
	case tidal.Update:
		return []sql.NamedArg{
			sql.Named("id", u.ID),
			sql.Named("name", u.Name),
			sql.Named("dob", u.DOB),
			sql.Named("email", u.Email),
			sql.Named("modified", u.Modified),
		}
	default:
		return []sql.NamedArg{
			sql.Named("id", u.ID),
			sql.Named("name", u.Name),
			sql.Named("dob", u.DOB),
			sql.Named("email", u.Email),
			sql.Named("password", u.Password),
			sql.Named("verified", u.Verified),
			sql.Named("last_seen", u.LastSeen),
			sql.Named("created", u.Created),
			sql.Named("modified", u.Modified),
		}
	}
}

func (u *User) Scan(op tidal.Operation, s tidal.Scanner) error {
	switch op {
	case tidal.List:
		return s.Scan(&u.ID, &u.Name, &u.Email, &u.Created, &u.Modified)
	default:
		return s.Scan(&u.ID, &u.Name, &u.DOB, &u.Email, &u.Password, &u.Verified, &u.LastSeen, &u.Created, &u.Modified)
	}
}

// Compare returns a deterministic lexical ordering across all User fields.
func (u *User) Compare(other *User) int {
	if u == nil || other == nil {
		switch {
		case u == nil && other == nil:
			return 0
		case u == nil:
			return -1
		default:
			return 1
		}
	}

	if diff := u.ID.Compare(other.ID); diff != 0 {
		return diff
	}
	if diff := u.Created.Compare(other.Created); diff != 0 {
		return diff
	}
	if diff := u.Modified.Compare(other.Modified); diff != 0 {
		return diff
	}
	if diff := strings.Compare(u.Name, other.Name); diff != 0 {
		return diff
	}
	if diff := u.DOB.Compare(other.DOB); diff != 0 {
		return diff
	}
	if diff := strings.Compare(u.Email, other.Email); diff != 0 {
		return diff
	}
	if diff := strings.Compare(u.Password, other.Password); diff != 0 {
		return diff
	}
	if u.Verified != other.Verified {
		if u.Verified {
			return 1
		}
		return -1
	}
	if diff := u.LastSeen.Compare(other.LastSeen); diff != 0 {
		return diff
	}
	return 0
}

// Equal reports semantic model equality with DB-friendly timestamp tolerance.
func (u *User) Equal(other *User) bool {
	return u.Compare(other) == 0
}

// NewConformanceUser returns a fresh User suitable for CRUD conformance runs.
func NewConformanceUser() *User {
	return &User{
		Name:     "Conformance User",
		DOB:      fields.Time(time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)),
		Email:    fmt.Sprintf("conformance-%s@example.com", ulid.MakeSecure().String()),
		Password: "test-password",
		Verified: true,
		LastSeen: fields.Time(time.Now().Add(-1 * time.Hour)),
	}
}
