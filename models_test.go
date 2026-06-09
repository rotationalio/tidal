package tidal_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/tidal"
	"go.rtnl.ai/ulid"
)

//============================================================================
// Test Models
//============================================================================

type User struct {
	tidal.BaseModel
	Name     string
	DoB      sql.NullTime
	Email    string
	Password string
	Verified bool
	LastSeen sql.NullTime
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
			sql.Named("dob", u.DoB),
			sql.Named("email", u.Email),
			sql.Named("modified", u.Modified),
		}
	default:
		return []sql.NamedArg{
			sql.Named("id", u.ID),
			sql.Named("name", u.Name),
			sql.Named("dob", u.DoB),
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
		return s.Scan(&u.ID, &u.Name, &u.DoB, &u.Email, &u.Password, &u.Verified, &u.LastSeen, &u.Created, &u.Modified)
	}
}

//============================================================================
// Base Model Tests
//============================================================================

func TestBaseModel_Prepare(t *testing.T) {
	t.Run("Create", func(t *testing.T) {
		model := tidal.BaseModel{}
		model.Prepare(tidal.Create)
		require.False(t, model.ID.IsZero())
		require.False(t, model.Created.IsZero())
		require.False(t, model.Modified.IsZero())
	})

	t.Run("Update", func(t *testing.T) {
		model := tidal.BaseModel{}
		model.Prepare(tidal.Update)
		require.True(t, model.ID.IsZero())
		require.True(t, model.Created.IsZero())
		require.False(t, model.Modified.IsZero())

		prev := time.Now().Add(-38292 * time.Second)
		model = tidal.BaseModel{
			ID:       ulid.MustParse("01KTESYNDPVTRWK05N2TXFKGQZ"),
			Created:  prev,
			Modified: prev,
		}

		model.Prepare(tidal.Update)
		require.Equal(t, ulid.MustParse("01KTESYNDPVTRWK05N2TXFKGQZ"), model.ID)
		require.Equal(t, prev, model.Created)
		require.NotEqual(t, prev, model.Modified)
	})
}

func TestBaseModel_Validate(t *testing.T) {
	t.Run("Update", func(t *testing.T) {
		model := tidal.BaseModel{}
		require.ErrorIs(t, model.Validate(tidal.Update), tidal.ErrMissingID)
	})
}

func TestBaseModel_IsZero(t *testing.T) {
	t.Run("Zero", func(t *testing.T) {
		model := tidal.BaseModel{}
		require.True(t, model.IsZero())
	})

	t.Run("Not Zero", func(t *testing.T) {
		model := tidal.BaseModel{
			ID: ulid.MustParse("01KTESYNDPVTRWK05N2TXFKGQZ"),
		}
		require.False(t, model.IsZero())
	})
}
