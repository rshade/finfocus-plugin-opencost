package main

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-plugin-opencost/internal/kubeproxy"
)

// newClient builds the allocation client. In kube mode it resolves the
// kubeconfig first and hands the client the proxy transport, so the
// allocation package never learns that Kubernetes is involved.
func newClient(ctx context.Context, cfg allocation.Config, logger zerolog.Logger) (*allocation.Client, error) {
	if !cfg.Kube.Enabled() {
		return allocation.NewClient(ctx, cfg)
	}
	namespace, port := cfg.Kube.Target()
	conn, err := kubeproxy.Connect(kubeproxy.Target{
		Context:    cfg.Kube.Context,
		Namespace:  namespace,
		Service:    cfg.Kube.Service,
		Port:       port,
		Kubeconfig: cfg.Kube.Kubeconfig,
	})
	if err != nil {
		return nil, err
	}
	logger.Info().
		Str("kube_context", conn.Context).
		Str("kube_namespace", namespace).
		Str("kube_service", cfg.Kube.Service).
		Int("kube_port", port).
		Msg("reaching OpenCost through the Kubernetes API server service proxy")
	return allocation.NewClient(ctx, cfg,
		allocation.WithBaseURL(conn.BaseURL),
		allocation.WithTransport(conn.Transport))
}
