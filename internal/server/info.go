package server

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rshade/finfocus-plugin-opencost/pkg/version"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// Info builds the plugin record reported by GetPluginInfo.
// WithCapabilities replaces the SDK's inferred set, so every implemented capability is listed here.
func Info() *pluginsdk.PluginInfo {
	return pluginsdk.NewPluginInfo(
		pluginName,
		version.GetVersionInfo().Version,
		pluginsdk.WithSpecVersion(pluginsdk.SpecVersion),
		pluginsdk.WithProviders("kubernetes"),
		pluginsdk.WithCapabilities(
			pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS,
			pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS,
			pbc.PluginCapability_PLUGIN_CAPABILITY_PRICING_SPEC,
			pbc.PluginCapability_PLUGIN_CAPABILITY_ESTIMATE_COST,
			pbc.PluginCapability_PLUGIN_CAPABILITY_BATCH_COST,
		),
	)
}

// GetPluginInfo returns the name, the pkg/version value, and a v-prefixed spec version.
func (s *Server) GetPluginInfo(ctx context.Context, _ *pbc.GetPluginInfoRequest) (*pbc.GetPluginInfoResponse, error) {
	log := s.requestLogger(ctx)
	log.Info().Msg("plugin info")
	info := Info()
	if err := info.Validate(); err != nil {
		return nil, status.Errorf(codes.Internal, "plugin info: %v", err)
	}
	return &pbc.GetPluginInfoResponse{
		Name:         info.Name,
		Version:      info.Version,
		SpecVersion:  info.SpecVersion,
		Providers:    append([]string{}, info.Providers...),
		Metadata:     info.Metadata,
		Capabilities: append([]pbc.PluginCapability{}, info.Capabilities...),
	}, nil
}
