package conn_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/tidal/conn"
	"go.rtnl.ai/tidal/errors"
	"go.rtnl.ai/x/dsn"
)

// Tests that [conn.Open] returns an error when given an unsupported provider.
func TestOpenUnsupportedProvider(t *testing.T) {
	_, err := conn.Open(context.Background(), &dsn.DSN{Provider: "mysql"})
	require.Error(t, err)

	var unsupported errors.UnsupportedProvider
	require.ErrorAs(t, err, &unsupported)
	require.Equal(t, "mysql", string(unsupported))
}
