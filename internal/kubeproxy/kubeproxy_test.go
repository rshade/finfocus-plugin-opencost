package kubeproxy_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/client-go/rest"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-plugin-opencost/internal/kubeproxy"
	"github.com/rshade/finfocus-plugin-opencost/internal/kubeproxy/kubetest"
)

type seen struct {
	mu    sync.Mutex
	path  string
	query string
	auth  string
}

func (s *seen) handler(status int, body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.path, s.query, s.auth = r.URL.Path, r.URL.RawQuery, r.Header.Get("Authorization")
		s.mu.Unlock()
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	})
}

func (s *seen) get() (string, string, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.path, s.query, s.auth
}

func resolve(t *testing.T, target kubeproxy.Target) *http.Client {
	t.Helper()
	base, rt, err := kubeproxy.Resolve(target)
	require.NoError(t, err)
	require.NotContains(t, base[len("https://"):], "/")
	return &http.Client{Transport: rt}
}

func do(t *testing.T, base, path string, c *http.Client) (*http.Response, error) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, base+path, nil)
	require.NoError(t, err)
	return c.Do(req)
}

func TestKubeTargetBuildsServiceProxyURL(t *testing.T) {
	t.Run("explicit namespace and port", func(t *testing.T) {
		var s seen
		api := kubetest.Serve(t, s.handler(http.StatusOK, "{}"))
		cfg := kubetest.WriteKubeconfig(t, kubetest.Entry{Name: "c", Server: api.URL, Token: "t"})
		target := kubeproxy.Target{Namespace: "cost", Service: "oc", Port: 8080, Kubeconfig: cfg}
		base, rt, err := kubeproxy.Resolve(target)
		require.NoError(t, err)
		resp, err := do(t, base, "/allocation?window=30d&aggregate=namespace", &http.Client{Transport: rt})
		require.NoError(t, err)
		resp.Body.Close()
		path, query, _ := s.get()
		require.Equal(t, "/api/v1/namespaces/cost/services/oc:8080/proxy/allocation", path)
		require.Equal(t, "window=30d&aggregate=namespace", query)
	})

	t.Run("defaults reach the chart's service", func(t *testing.T) {
		var s seen
		api := kubetest.Serve(t, s.handler(http.StatusOK, "{}"))
		cfg := kubetest.WriteKubeconfig(t, kubetest.Entry{Name: "c", Server: api.URL, Token: "t"})
		ns, port := allocation.Kube{Service: "opencost"}.Target()
		base, rt, err := kubeproxy.Resolve(
			kubeproxy.Target{Namespace: ns, Service: "opencost", Port: port, Kubeconfig: cfg},
		)
		require.NoError(t, err)
		resp, err := do(t, base, "/allocation", &http.Client{Transport: rt})
		require.NoError(t, err)
		resp.Body.Close()
		path, _, _ := s.get()
		require.Equal(t, "/api/v1/namespaces/opencost/services/opencost:9003/proxy/allocation", path)
	})

	t.Run("server address with a path prefix", func(t *testing.T) {
		var s seen
		api := kubetest.Serve(t, s.handler(http.StatusOK, "{}"))
		cfg := kubetest.WriteKubeconfig(t, kubetest.Entry{Name: "c", Server: api.URL + "/k8s/clusters/c-1", Token: "t"})
		c := resolve(t, kubeproxy.Target{Namespace: "n", Service: "s", Port: 1, Kubeconfig: cfg})
		base, _, err := kubeproxy.Resolve(kubeproxy.Target{Namespace: "n", Service: "s", Port: 1, Kubeconfig: cfg})
		require.NoError(t, err)
		resp, err := do(t, base, "/healthz", c)
		require.NoError(t, err)
		resp.Body.Close()
		path, _, _ := s.get()
		require.Equal(t, "/k8s/clusters/c-1/api/v1/namespaces/n/services/s:1/proxy/healthz", path)
	})

	t.Run("same query as baseUrl", func(t *testing.T) {
		const body = `{"code":200,"data":[{}]}`
		var direct, proxied seen
		plain := httptest.NewServer(direct.handler(http.StatusOK, body))
		t.Cleanup(plain.Close)
		api := kubetest.Serve(t, proxied.handler(http.StatusOK, body))
		cfg := kubetest.WriteKubeconfig(t, kubetest.Entry{Name: "c", Server: api.URL, Token: "t"})
		base, rt, err := kubeproxy.Resolve(
			kubeproxy.Target{Namespace: "opencost", Service: "opencost", Port: 9003, Kubeconfig: cfg},
		)
		require.NoError(t, err)

		q := allocation.Query{
			Window:      "30d",
			Filter:      map[string]string{"namespace": "oc-test", "node": "n1"},
			AggregateBy: []string{"namespace"},
		}
		viaURL, err := allocation.NewClient(
			t.Context(),
			allocation.Config{BaseURL: plain.URL, Currency: "EUR", CacheTTL: -1},
		)
		require.NoError(t, err)
		viaKube, err := allocation.NewClient(
			t.Context(),
			allocation.Config{BaseURL: base, Currency: "EUR", CacheTTL: -1},
			allocation.WithTransport(rt),
		)
		require.NoError(t, err)
		_, err = viaURL.GetDetailedAllocation(t.Context(), q)
		require.NoError(t, err)
		_, err = viaKube.GetDetailedAllocation(t.Context(), q)
		require.NoError(t, err)

		directPath, directQuery, _ := direct.get()
		proxyPath, proxyQuery, _ := proxied.get()
		require.Equal(t, directQuery, proxyQuery)
		require.Equal(t, "/api/v1/namespaces/opencost/services/opencost:9003/proxy"+directPath, proxyPath)
	})
}

func TestKubeCredentialSources(t *testing.T) {
	newAPI := func(t *testing.T) (*httptest.Server, *seen) {
		var s seen
		return kubetest.Serve(t, s.handler(http.StatusOK, "{}")), &s
	}
	target := func(cfg string) kubeproxy.Target {
		return kubeproxy.Target{Namespace: "n", Service: "s", Port: 1, Kubeconfig: cfg}
	}

	t.Run("explicit kubeconfig path sends its token", func(t *testing.T) {
		api, s := newAPI(t)
		cfg := kubetest.WriteKubeconfig(t, kubetest.Entry{Name: "c", Server: api.URL, Token: "file-token"})
		base, rt, err := kubeproxy.Resolve(target(cfg))
		require.NoError(t, err)
		resp, err := do(t, base, "/x", &http.Client{Transport: rt})
		require.NoError(t, err)
		resp.Body.Close()
		_, _, auth := s.get()
		require.Equal(t, "Bearer file-token", auth)
	})

	t.Run("KUBECONFIG is used when no path is set", func(t *testing.T) {
		api, s := newAPI(t)
		cfg := kubetest.WriteKubeconfig(t, kubetest.Entry{Name: "c", Server: api.URL, Token: "env-token"})
		t.Setenv("KUBECONFIG", cfg)
		base, rt, err := resolveAt(target(""), filepath.Join(t.TempDir(), "none"))
		require.NoError(t, err)
		resp, err := do(t, base, "/x", &http.Client{Transport: rt})
		require.NoError(t, err)
		resp.Body.Close()
		_, _, auth := s.get()
		require.Equal(t, "Bearer env-token", auth)
	})

	t.Run("the default path is used when KUBECONFIG is unset", func(t *testing.T) {
		api, s := newAPI(t)
		cfg := kubetest.WriteKubeconfig(t, kubetest.Entry{Name: "c", Server: api.URL, Token: "home-token"})
		home := t.TempDir() + "/.kube/config"
		require.NoError(t, copyFile(cfg, home))
		t.Setenv("KUBECONFIG", "")
		base, rt, err := resolveAt(target(""), home)
		require.NoError(t, err)
		resp, err := do(t, base, "/x", &http.Client{Transport: rt})
		require.NoError(t, err)
		resp.Body.Close()
		_, _, auth := s.get()
		require.Equal(t, "Bearer home-token", auth)
	})

	t.Run("context selects the second cluster", func(t *testing.T) {
		first, firstSeen := newAPI(t)
		second, secondSeen := newAPI(t)
		cfg := kubetest.WriteKubeconfig(t,
			kubetest.Entry{Name: "dev", Server: first.URL, Token: "dev-token"},
			kubetest.Entry{Name: "prod", Server: second.URL, Token: "prod-token"})
		tg := target(cfg)
		tg.Context = "prod"
		base, rt, err := kubeproxy.Resolve(tg)
		require.NoError(t, err)
		resp, err := do(t, base, "/x", &http.Client{Transport: rt})
		require.NoError(t, err)
		resp.Body.Close()
		_, _, auth := secondSeen.get()
		require.Equal(t, "Bearer prod-token", auth)
		path, _, _ := firstSeen.get()
		require.Empty(t, path)
	})

	t.Run("no kubeconfig and no pod service account", func(t *testing.T) {
		_, _, err := resolveBare(t, target(""))
		require.Error(t, err)
		require.Contains(t, err.Error(), "no kubeconfig found")
	})
}

func TestKubeExecPluginCredentials(t *testing.T) {
	const token = "exec-minted-token"

	t.Run("token from the plugin is sent", func(t *testing.T) {
		var s seen
		api := kubetest.Serve(t, s.handler(http.StatusOK, "{}"))
		script := kubetest.WriteScript(
			t,
			`printf '{"apiVersion":"client.authentication.k8s.io/v1","kind":"ExecCredential","status":{"token":"`+token+`"}}'`,
		)
		cfg := kubetest.WriteKubeconfig(t, kubetest.Entry{Name: "c", Server: api.URL, ExecCommand: script})
		base, rt, err := kubeproxy.Resolve(kubeproxy.Target{Namespace: "n", Service: "s", Port: 1, Kubeconfig: cfg})
		require.NoError(t, err)
		resp, err := do(t, base, "/x", &http.Client{Transport: rt})
		require.NoError(t, err)
		resp.Body.Close()
		_, _, auth := s.get()
		require.Equal(t, "Bearer "+token, auth)
	})

	t.Run("a failing plugin names the command and not its output", func(t *testing.T) {
		const leaked = "LEAKED-PLUGIN-OUTPUT"
		var s seen
		api := kubetest.Serve(t, s.handler(http.StatusOK, "{}"))
		script := kubetest.WriteScript(t, "echo "+leaked+"; exit 3")
		cfg := kubetest.WriteKubeconfig(t, kubetest.Entry{Name: "c", Server: api.URL, ExecCommand: script})
		base, rt, err := kubeproxy.Resolve(kubeproxy.Target{Namespace: "n", Service: "s", Port: 1, Kubeconfig: cfg})
		require.NoError(t, err)
		_, err = do(t, base, "/x", &http.Client{Transport: rt})
		require.Error(t, err)
		require.Equal(t, codes.Unavailable, status.Code(err))
		require.Contains(t, err.Error(), "credential plugin")
		require.Contains(t, err.Error(), script)
		require.NotContains(t, err.Error(), leaked)
		path, _, _ := s.get()
		require.Empty(t, path)
	})
}

func TestKubeMissingContextNamesContext(t *testing.T) {
	api := kubetest.Serve(t, http.NotFoundHandler())
	cfg := kubetest.WriteKubeconfig(t, kubetest.Entry{Name: "dev", Server: api.URL, Token: "t"})

	t.Run("unknown context", func(t *testing.T) {
		_, _, err := kubeproxy.Resolve(
			kubeproxy.Target{Context: "prod-east", Namespace: "n", Service: "s", Port: 1, Kubeconfig: cfg},
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "prod-east")
		require.Contains(t, err.Error(), "is not in the kubeconfig")
		require.NotContains(t, err.Error(), "\n")
	})

	t.Run("context with no kubeconfig", func(t *testing.T) {
		_, _, err := resolveBare(t, kubeproxy.Target{Context: "prod-east", Namespace: "n", Service: "s", Port: 1})
		require.Error(t, err)
		require.Contains(t, err.Error(), "prod-east")
	})

	t.Run("no kubeconfig at all", func(t *testing.T) {
		_, _, err := resolveBare(t, kubeproxy.Target{Namespace: "n", Service: "s", Port: 1})
		require.Error(t, err)
		require.Contains(t, err.Error(), "no kubeconfig found")
	})
}

func TestKubeTransportMapsAPIServerFailures(t *testing.T) {
	const token = "never-in-output"
	cases := []struct {
		name   string
		status int
		body   string
		code   codes.Code
		want   []string
	}{
		{
			"forbidden",
			http.StatusForbidden,
			`{"kind":"Status","message":"forbidden"}`,
			codes.PermissionDenied,
			[]string{"services/proxy", "opencost", "ctx"},
		},
		{"unauthorized", http.StatusUnauthorized, `{"kind":"Status"}`, codes.Unauthenticated, []string{"ctx"}},
		{
			"missing service",
			http.StatusNotFound,
			`{"kind":"Status","message":"services \"opencost\" not found"}`,
			codes.Unavailable,
			[]string{"opencost", "ctx"},
		},
		{
			"no endpoints",
			http.StatusServiceUnavailable,
			`{"kind":"Status","message":"no endpoints available"}`,
			codes.Unavailable,
			[]string{"opencost"},
		},
		{
			"gateway timeout",
			http.StatusGatewayTimeout,
			`{"kind":"Status","message":"timeout"}`,
			codes.Unavailable,
			[]string{"opencost"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var s seen
			api := kubetest.Serve(t, s.handler(tc.status, tc.body))
			cfg := kubetest.WriteKubeconfig(t, kubetest.Entry{Name: "ctx", Server: api.URL, Token: token})
			base, rt, err := kubeproxy.Resolve(
				kubeproxy.Target{Namespace: "ns", Service: "opencost", Port: 9003, Kubeconfig: cfg},
			)
			require.NoError(t, err)
			_, err = do(t, base, "/allocation", &http.Client{Transport: rt})
			require.Equal(t, tc.code, status.Code(err))
			for _, w := range tc.want {
				require.Contains(t, err.Error(), w)
			}
			require.NotContains(t, err.Error(), token)
		})
	}

	t.Run("an answer from OpenCost passes through", func(t *testing.T) {
		bodies := []string{"Invalid 'window' parameter", `{"code":404,"message":"not an API server Status"}`}
		for _, code := range []int{http.StatusNotFound, http.StatusServiceUnavailable} {
			for _, body := range bodies {
				var s seen
				api := kubetest.Serve(t, s.handler(code, body))
				cfg := kubetest.WriteKubeconfig(t, kubetest.Entry{Name: "ctx", Server: api.URL, Token: token})
				base, rt, err := kubeproxy.Resolve(
					kubeproxy.Target{Namespace: "ns", Service: "opencost", Port: 9003, Kubeconfig: cfg},
				)
				require.NoError(t, err)
				resp, err := do(t, base, "/allocation", &http.Client{Transport: rt})
				require.NoError(t, err)
				require.Equal(t, code, resp.StatusCode)
				resp.Body.Close()
			}
		}
	})

	t.Run("unreachable API server", func(t *testing.T) {
		api := kubetest.Serve(t, http.NotFoundHandler())
		cfg := kubetest.WriteKubeconfig(t, kubetest.Entry{Name: "ctx", Server: api.URL, Token: token})
		base, rt, err := kubeproxy.Resolve(
			kubeproxy.Target{Namespace: "ns", Service: "opencost", Port: 9003, Kubeconfig: cfg},
		)
		require.NoError(t, err)
		api.Close()
		_, err = do(t, base, "/allocation", &http.Client{Transport: rt})
		require.Equal(t, codes.Unavailable, status.Code(err))
		require.Contains(t, err.Error(), "ctx")
		require.NotContains(t, err.Error(), token)
	})

	t.Run("canceled request stays canceled", func(t *testing.T) {
		var s seen
		api := kubetest.Serve(t, s.handler(http.StatusOK, "{}"))
		cfg := kubetest.WriteKubeconfig(t, kubetest.Entry{Name: "ctx", Server: api.URL, Token: token})
		base, rt, err := kubeproxy.Resolve(
			kubeproxy.Target{Namespace: "ns", Service: "opencost", Port: 9003, Kubeconfig: cfg},
		)
		require.NoError(t, err)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/x", nil)
		require.NoError(t, err)
		_, err = (&http.Client{Transport: rt}).Do(req)
		require.ErrorIs(t, err, context.Canceled)
	})
}

func TestKubeModeKeepsCacheAndRateLimit(t *testing.T) {
	const token = "cache-key-must-not-hold-this"
	var hits int
	var mu sync.Mutex
	api := kubetest.Serve(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		hits++
		mu.Unlock()
		_, _ = w.Write([]byte(`{"code":200,"data":[{}]}`))
	}))
	cfg := kubetest.WriteKubeconfig(t, kubetest.Entry{Name: "c", Server: api.URL, Token: token})
	base, rt, err := kubeproxy.Resolve(
		kubeproxy.Target{Namespace: "opencost", Service: "opencost", Port: 9003, Kubeconfig: cfg},
	)
	require.NoError(t, err)

	var logs bytes.Buffer
	cli, err := allocation.NewClient(t.Context(),
		allocation.Config{BaseURL: base, Currency: "EUR", RequestsPerSecond: 0.001, RateBurst: 1},
		allocation.WithTransport(rt))
	require.NoError(t, err)
	cli.SetLogger(zerolog.New(&logs))

	first := allocation.Query{Window: "30d", Filter: map[string]string{"namespace": "a"}}
	_, err = cli.GetDetailedAllocation(t.Context(), first)
	require.NoError(t, err)
	_, err = cli.GetDetailedAllocation(t.Context(), first)
	require.NoError(t, err, "a repeat inside the cache TTL does not use the limiter")

	_, err = cli.GetDetailedAllocation(
		t.Context(),
		allocation.Query{Window: "30d", Filter: map[string]string{"namespace": "b"}},
	)
	require.True(t, allocation.IsRateLimited(err), "burst of one allows one backend call")

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, 1, hits)
	require.NotContains(t, logs.String(), token)
}

func TestKubeCredentialSourcesInCluster(t *testing.T) {
	var s seen
	api := kubetest.Serve(t, s.handler(http.StatusOK, "{}"))
	//nolint:unparam // the signature is the one Resolve expects
	pod := func() (*rest.Config, error) {
		return &rest.Config{
			Host:            api.URL,
			BearerToken:     "pod-token",
			TLSClientConfig: rest.TLSClientConfig{Insecure: true},
		}, nil
	}
	t.Setenv("KUBECONFIG", "")
	home := filepath.Join(t.TempDir(), "none")

	base, rt, err := kubeproxy.ResolveWith(
		kubeproxy.Target{Namespace: "opencost", Service: "opencost", Port: 9003},
		home,
		pod,
	)
	require.NoError(t, err)
	resp, err := do(t, base, "/healthz", &http.Client{Transport: rt})
	require.NoError(t, err)
	resp.Body.Close()
	path, _, auth := s.get()
	require.Equal(t, "Bearer pod-token", auth)
	require.Equal(t, "/api/v1/namespaces/opencost/services/opencost:9003/proxy/healthz", path)

	t.Run("a requested context cannot apply in a pod", func(t *testing.T) {
		_, _, ctxErr := kubeproxy.ResolveWith(
			kubeproxy.Target{Context: "prod-east", Namespace: "opencost", Service: "opencost", Port: 9003},
			home,
			pod,
		)
		require.ErrorContains(t, ctxErr, "prod-east")
	})
}
