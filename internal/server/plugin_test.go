package server_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-plugin-opencost/internal/server"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func TestServerNameAndSDKRegistration(t *testing.T) {
	t.Parallel()

	require.Equal(t, "opencost", server.PluginName, "PluginName must equal the install name")

	cli, err := allocation.NewClient(t.Context(), allocation.Config{BaseURL: "http://127.0.0.1:9"})
	require.NoError(t, err)

	srv := server.New(cli)
	require.Equal(t, server.PluginName, srv.Name())

	var plugin pluginsdk.Plugin = srv
	require.Equal(t, server.PluginName, plugin.Name())

	info, err := srv.GetPluginInfo(t.Context(), &pbc.GetPluginInfoRequest{})
	require.NoError(t, err)
	require.Equal(t, server.PluginName, info.GetName())
}
