package server_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-plugin-opencost/internal/server"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
)

func TestServerNameAndSDKRegistration(t *testing.T) {
	t.Parallel()

	cli, err := allocation.NewClient(t.Context(), allocation.Config{BaseURL: "http://127.0.0.1:9"})
	require.NoError(t, err)

	srv := server.New(cli)
	require.Equal(t, "opencost", srv.Name())

	var plugin pluginsdk.Plugin = srv
	require.Equal(t, "opencost", plugin.Name())
}
