package server

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-plugin-opencost/pkg/version"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// Info builds the plugin record reported by GetPluginInfo for the configured profile.
// WithCapabilities replaces the SDK's inferred set, so every capability the profile serves is listed here.
// Profile kubecost adds BUDGETS because only that profile can answer GetBudgets.
// RECOMMENDATIONS appears only when the recommendations flag is on.
func Info(cfg allocation.Config) *pluginsdk.PluginInfo {
	capabilities := []pbc.PluginCapability{
		pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS,
		pbc.PluginCapability_PLUGIN_CAPABILITY_PRICING_SPEC,
		pbc.PluginCapability_PLUGIN_CAPABILITY_ESTIMATE_COST,
		pbc.PluginCapability_PLUGIN_CAPABILITY_BATCH_COST,
		pbc.PluginCapability_PLUGIN_CAPABILITY_DRY_RUN,
	}
	if profile, err := cfg.ProfileName(); err == nil && profile == allocation.ProfileKubecost {
		capabilities = append(capabilities, pbc.PluginCapability_PLUGIN_CAPABILITY_BUDGETS)
	}
	if cfg.Recommendations {
		capabilities = append(capabilities, pbc.PluginCapability_PLUGIN_CAPABILITY_RECOMMENDATIONS)
	}
	return pluginsdk.NewPluginInfo(
		pluginName,
		version.GetVersionInfo().Version,
		pluginsdk.WithSpecVersion(pluginsdk.SpecVersion),
		pluginsdk.WithProviders("kubernetes"),
		pluginsdk.WithCapabilities(capabilities...),
	)
}

// GetPluginInfo returns the name, the pkg/version value, and a v-prefixed spec version.
func (s *Server) GetPluginInfo(
	ctx context.Context,
	_ *pbc.GetPluginInfoRequest,
) (*pbc.GetPluginInfoResponse, error) {
	return observeResult(ctx, s, "GetPluginInfo", func() (*pbc.GetPluginInfoResponse, error) {
		log := s.requestLogger(ctx)
		log.Info().Msg("plugin info")
		info := Info(s.cli.GetConfig())
		if validateErr := info.Validate(); validateErr != nil {
			return nil, status.Errorf(codes.Internal, "plugin info: %v", validateErr)
		}
		return &pbc.GetPluginInfoResponse{
			Name:         info.Name,
			Version:      info.Version,
			SpecVersion:  info.SpecVersion,
			Providers:    append([]string{}, info.Providers...),
			Metadata:     info.Metadata,
			Capabilities: append([]pbc.PluginCapability{}, info.Capabilities...),
		}, nil
	})
}
