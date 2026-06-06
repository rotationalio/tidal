package tidal_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/tidal"
	"go.rtnl.ai/ulid"
)

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
