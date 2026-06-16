package model_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/tidal/model"
	"go.rtnl.ai/ulid"
)

//============================================================================
// Base Model Tests
//============================================================================

func TestBaseModel_Prepare(t *testing.T) {
	t.Run("Create", func(t *testing.T) {
		m := model.BaseModel{}
		m.Prepare(model.Create)
		require.False(t, m.ID.IsZero())
		require.False(t, m.Created.IsZero())
		require.False(t, m.Modified.IsZero())
	})

	t.Run("Update", func(t *testing.T) {
		m := model.BaseModel{}
		m.Prepare(model.Update)
		require.True(t, m.ID.IsZero())
		require.True(t, m.Created.IsZero())
		require.False(t, m.Modified.IsZero())

		prev := time.Now().Add(-38292 * time.Second)
		m = model.BaseModel{
			ID:       ulid.MustParse("01KTESYNDPVTRWK05N2TXFKGQZ"),
			Created:  prev,
			Modified: prev,
		}

		m.Prepare(model.Update)
		require.Equal(t, ulid.MustParse("01KTESYNDPVTRWK05N2TXFKGQZ"), m.ID)
		require.Equal(t, prev, m.Created)
		require.NotEqual(t, prev, m.Modified)
	})
}

func TestBaseModel_Validate(t *testing.T) {
	t.Run("Update", func(t *testing.T) {
		m := model.BaseModel{}
		require.ErrorIs(t, m.Validate(model.Update), model.ErrMissingID)
	})
}

func TestBaseModel_IsZero(t *testing.T) {
	t.Run("Zero", func(t *testing.T) {
		m := model.BaseModel{}
		require.True(t, m.IsZero())
	})

	t.Run("Not Zero", func(t *testing.T) {
		m := model.BaseModel{
			ID: ulid.MustParse("01KTESYNDPVTRWK05N2TXFKGQZ"),
		}
		require.False(t, m.IsZero())
	})
}
