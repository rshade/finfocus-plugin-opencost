package main

import (
	"net/http"
	"sync"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-plugin-opencost/internal/kubeproxy/kubetest"
	"github.com/rshade/finfocus-plugin-opencost/internal/server"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func TestKubeClientKeepsOperatorConfig(t *testing.T) {
	t.Parallel()

	var (
		mu    sync.Mutex
		paths []string
	)
	api := kubetest.Serve(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		_, _ = w.Write([]byte(`{"code":200,"data":[{}]}`))
	}))
	kubeconfig := kubetest.WriteKubeconfig(t, kubetest.Entry{Name: "c", Server: api.URL, Token: "t"})
	cfg := allocation.Config{
		Currency: "EUR",
		CacheTTL: -1,
		Kube:     allocation.Kube{Service: "opencost", Kubeconfig: kubeconfig},
	}
	require.NoError(t, cfg.Validate())

	cli, err := newClient(t.Context(), cfg, zerolog.Nop())
	require.NoError(t, err)
	require.Empty(t, cli.GetConfig().BaseURL)
	require.NoError(t, cli.GetConfig().Validate())

	resp, err := server.New(cli).HandleDryRun(t.Context(), &pbc.DryRunRequest{
		Resource: &pbc.ResourceDescriptor{ResourceType: "kubernetes:apps/v1:Deployment"},
	})
	require.NoError(t, err)
	require.True(t, resp.GetConfigurationValid(), resp.GetConfigurationErrors())
	require.Empty(t, resp.GetConfigurationErrors())

	_, err = cli.GetDetailedAllocation(t.Context(), allocation.Query{Window: "30d"})
	require.NoError(t, err)
	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, []string{"/api/v1/namespaces/opencost/services/opencost:9003/proxy/allocation"}, paths)
}
