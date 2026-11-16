package server_test

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-plugin-opencost/internal/kubeproxy"
	"github.com/rshade/finfocus-plugin-opencost/internal/kubeproxy/kubetest"
	"github.com/rshade/finfocus-plugin-opencost/internal/server"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const kubeToken = "kube-secret-token"

type proxyAnswer struct {
	status int
	body   string
}

// kubeServer starts a fake API server that answers every request the same way
// and returns a server whose client reaches it through the service proxy.
func kubeServer(t *testing.T, answer proxyAnswer) (*server.Server, func() string) {
	t.Helper()
	var mu sync.Mutex
	var lastPath string
	api := kubetest.Serve(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		lastPath = r.URL.Path
		mu.Unlock()
		w.WriteHeader(answer.status)
		_, _ = w.Write([]byte(answer.body))
	}))
	cfg := kubetest.WriteKubeconfig(t, kubetest.Entry{Name: "prod", Server: api.URL, Token: kubeToken})
	base, rt, err := kubeproxy.Resolve(kubeproxy.Target{
		Namespace: "cost", Service: "opencost", Port: 9003, Kubeconfig: cfg,
	})
	require.NoError(t, err)
	cli, err := allocation.NewClient(t.Context(),
		allocation.Config{BaseURL: base, Currency: "EUR"}, allocation.WithTransport(rt))
	require.NoError(t, err)
	return server.New(cli), func() string {
		mu.Lock()
		defer mu.Unlock()
		return lastPath
	}
}

func projectOne(t *testing.T, srv *server.Server) error {
	t.Helper()
	_, err := srv.GetProjectedCost(t.Context(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider: "kubernetes", ResourceType: "kubernetes:core/v1:Namespace", Id: "oc-test",
		},
	})
	return err
}

func TestKubeForbiddenIsPermissionDenied(t *testing.T) {
	srv, _ := kubeServer(t, proxyAnswer{http.StatusForbidden, `{"kind":"Status","code":403}`})

	err := projectOne(t, srv)
	require.Equal(t, codes.PermissionDenied, status.Code(err))
	for _, want := range []string{"services/proxy", "cost", "opencost"} {
		require.Contains(t, status.Convert(err).Message(), want)
	}
	require.NotContains(t, status.Convert(err).Message(), kubeToken)
	require.NotContains(t, status.Convert(err).Message(), "KUBECOST_API_TOKEN")
}

func TestKubeUnreachableServiceIsUnavailable(t *testing.T) {
	cases := []struct {
		name   string
		answer proxyAnswer
	}{
		{
			"service missing",
			proxyAnswer{http.StatusNotFound, `{"kind":"Status","message":"services \"opencost\" not found"}`},
		},
		{
			"no ready endpoints",
			proxyAnswer{
				http.StatusServiceUnavailable,
				`{"kind":"Status","message":"no endpoints available for service \"opencost\""}`,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, _ := kubeServer(t, tc.answer)
			err := projectOne(t, srv)
			require.Equal(t, codes.Unavailable, status.Code(err))
			require.Contains(t, status.Convert(err).Message(), "opencost")
			require.Contains(t, status.Convert(err).Message(), "cost")
		})
	}

	t.Run("API server not reachable names context, namespace and service", func(t *testing.T) {
		dead := httptest.NewServer(http.NotFoundHandler())
		url := dead.URL
		dead.Close()
		cfg := kubetest.WriteKubeconfig(t, kubetest.Entry{Name: "prod", Server: url, Token: kubeToken})
		base, rt, err := kubeproxy.Resolve(kubeproxy.Target{
			Namespace: "cost", Service: "opencost", Port: 9003, Kubeconfig: cfg,
		})
		require.NoError(t, err)
		cli, err := allocation.NewClient(t.Context(),
			allocation.Config{BaseURL: base, Currency: "EUR"}, allocation.WithTransport(rt))
		require.NoError(t, err)

		err = projectOne(t, server.New(cli))
		require.Equal(t, codes.Unavailable, status.Code(err))
		for _, want := range []string{"prod", "cost", "opencost"} {
			require.Contains(t, status.Convert(err).Message(), want)
		}
		require.NotContains(t, status.Convert(err).Message(), kubeToken)
	})

	t.Run("OpenCost answered with no row is not unavailable", func(t *testing.T) {
		srv, _ := kubeServer(t, proxyAnswer{http.StatusOK, `{"code":200,"data":[{}]}`})
		resp, err := srv.GetActualCost(t.Context(), actualWindow("namespace/oc-test"))
		require.NoError(t, err)
		require.Empty(t, resp.GetResults())
	})

	t.Run("an answer from OpenCost keeps its own mapping", func(t *testing.T) {
		srv, _ := kubeServer(t, proxyAnswer{http.StatusNotFound, "page not found"})
		err := projectOne(t, srv)
		require.Error(t, err)
		require.Contains(t, status.Convert(err).Message(), "allocation API error: status=404")
		require.NotContains(t, status.Convert(err).Message(), "unreachable through context")
	})
}

func TestKubeHealthProbeUsesProxy(t *testing.T) {
	t.Run("healthy", func(t *testing.T) {
		srv, lastPath := kubeServer(t, proxyAnswer{http.StatusOK, "ok"})
		require.NoError(t, srv.Check(t.Context()))
		require.Equal(t, "/api/v1/namespaces/cost/services/opencost:9003/proxy/healthz", lastPath())
	})

	t.Run("forbidden", func(t *testing.T) {
		srv, _ := kubeServer(t, proxyAnswer{http.StatusForbidden, `{"kind":"Status"}`})
		err := srv.Check(t.Context())
		require.Equal(t, codes.PermissionDenied, status.Code(err))
		require.Contains(t, status.Convert(err).Message(), "services/proxy")
	})
}
