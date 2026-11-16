package server_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-plugin-opencost/internal/server"
	"github.com/rshade/finfocus-plugin-opencost/pkg/version"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func TestGetPluginInfoReportsSpecAndCapabilities(t *testing.T) {
	t.Parallel()

	cli, err := allocation.NewClient(t.Context(), allocation.Config{BaseURL: "http://127.0.0.1:9"})
	require.NoError(t, err)

	resp, err := server.New(cli).GetPluginInfo(t.Context(), &pbc.GetPluginInfoRequest{})
	require.NoError(t, err)
	require.Equal(t, "opencost", resp.GetName())
	require.Equal(t, version.GetVersionInfo().Version, resp.GetVersion())
	require.True(t, strings.HasPrefix(resp.GetSpecVersion(), "v"), resp.GetSpecVersion())
	require.Equal(t, pluginsdk.SpecVersion, resp.GetSpecVersion())
	require.Contains(t, resp.GetCapabilities(), pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS)
	require.Contains(t, resp.GetCapabilities(), pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS)
	require.Contains(t, resp.GetCapabilities(), pbc.PluginCapability_PLUGIN_CAPABILITY_PRICING_SPEC)
	require.Contains(t, resp.GetCapabilities(), pbc.PluginCapability_PLUGIN_CAPABILITY_ESTIMATE_COST)
	require.Contains(t, resp.GetCapabilities(), pbc.PluginCapability_PLUGIN_CAPABILITY_BATCH_COST)
	require.Contains(t, resp.GetCapabilities(), pbc.PluginCapability_PLUGIN_CAPABILITY_DRY_RUN)
}

func TestGetPluginInfoCapabilitiesFollowProfile(t *testing.T) {
	t.Parallel()

	base := []pbc.PluginCapability{
		pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_PRICING_SPEC,
		pbc.PluginCapability_PLUGIN_CAPABILITY_ESTIMATE_COST,
		pbc.PluginCapability_PLUGIN_CAPABILITY_BATCH_COST,
		pbc.PluginCapability_PLUGIN_CAPABILITY_DRY_RUN,
	}
	cases := []struct {
		name    string
		profile string
		want    []pbc.PluginCapability
	}{
		{"default profile", "", base},
		{"opencost", allocation.ProfileOpenCost, base},
		{
			"kubecost",
			allocation.ProfileKubecost,
			append(append([]pbc.PluginCapability{}, base...), pbc.PluginCapability_PLUGIN_CAPABILITY_BUDGETS),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cli, err := allocation.NewClient(t.Context(), allocation.Config{
				BaseURL: "http://127.0.0.1:9",
				Profile: tc.profile,
			})
			require.NoError(t, err)

			resp, err := server.New(cli).GetPluginInfo(t.Context(), &pbc.GetPluginInfoRequest{})
			require.NoError(t, err)
			require.Equal(t, tc.want, resp.GetCapabilities())
		})
	}
}
