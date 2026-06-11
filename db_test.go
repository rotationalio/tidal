package tidal_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/tidal"
	"go.rtnl.ai/x/dsn"
)

// Tests that [tidal.Open] returns an error when given an unsupported provider.
func TestOpenUnsupportedProvider(t *testing.T) {
	_, err := tidal.Open(context.Background(), &dsn.DSN{Provider: "mysql"})
	require.Error(t, err)

	var unsupported tidal.UnsupportedProvider
	require.ErrorAs(t, err, &unsupported)
	require.Equal(t, "mysql", string(unsupported))
}
