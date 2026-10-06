package server_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-plugin-opencost/internal/server"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const authTestToken = "oc95-secret-token"

func kubecostServer(t *testing.T, baseURL string) *server.Server {
	t.Helper()
	cli, err := allocation.NewClient(t.Context(), allocation.Config{
		BaseURL:          baseURL,
		Profile:          allocation.ProfileKubecost,
		Currency:         "EUR",
		DefaultNamespace: "web",
		APIToken:         authTestToken,
	})
	require.NoError(t, err)
	return server.New(cli)
}

func statusBackend(t *testing.T, status int) string {
	t.Helper()
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "backend response", status)
	}))
	t.Cleanup(backend.Close)
	return backend.URL
}

func estimateRequest(t *testing.T) *pbc.EstimateCostRequest {
	t.Helper()
	attrs, err := structpb.NewStruct(map[string]any{
		"metadata": map[string]any{"name": "web-deploy", "namespace": "web"},
	})
	require.NoError(t, err)
	return &pbc.EstimateCostRequest{
		ResourceType: "kubernetes:apps/v1:Deployment",
		Attributes:   attrs,
	}
}

func TestBackendAuthMaps401And403(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		status int
		want   codes.Code
	}{
		{"401 is Unauthenticated", http.StatusUnauthorized, codes.Unauthenticated},
		{"403 is PermissionDenied", http.StatusForbidden, codes.PermissionDenied},
		{"500 stays Unavailable", http.StatusInternalServerError, codes.Unavailable},
		{"404 stays Unavailable", http.StatusNotFound, codes.Unavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := kubecostServer(t, statusBackend(t, tc.status))

			_, err := srv.GetBudgets(t.Context(), &pbc.GetBudgetsRequest{})
			require.Equal(t, tc.want, status.Code(err), "budgets")

			_, err = srv.GetActualCost(t.Context(), actualRequest())
			require.Equal(t, tc.want, status.Code(err), "allocation")

			_, err = srv.EstimateCost(t.Context(), estimateRequest(t))
			require.Equal(t, tc.want, status.Code(err), "prediction")

			err = srv.Check(t.Context())
			require.Error(t, err)
			require.Equal(t, tc.want, status.Code(err), "health")
		})
	}
}

func TestBackendAuthErrorOmitsToken(t *testing.T) {
	t.Parallel()

	for _, httpStatus := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		srv := kubecostServer(t, statusBackend(t, httpStatus))
		var logs bytes.Buffer
		srv.SetLogger(zerolog.New(&logs))

		_, err := srv.GetBudgets(t.Context(), &pbc.GetBudgetsRequest{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "KUBECOST_API_TOKEN")
		require.NotContains(t, err.Error(), authTestToken)
		require.NotContains(t, logs.String(), authTestToken)
	}
}
