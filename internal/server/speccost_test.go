package server_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-plugin-opencost/internal/server"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// contract-fixture: not verified against live Kubecost.
func TestEstimateCostKubecostUsesSpecCostContract(t *testing.T) {
	t.Parallel()
	t.Attr("label", "contract-fixture")

	fixture := readRepoFile(t, "testdata/kubecost-contract/speccost.json")
	citation := string(readRepoFile(t, "testdata/kubecost-contract/speccost.citation"))
	require.Contains(t, citation, "pkg/query/prediction_speccost.go")
	require.Contains(t, citation, "1f45d3085b2ffa84758bfa8131ea8b7784cd8ed1")
	require.Contains(t, citation, "not verified against live Kubecost")

	var rows []struct {
		CostAfter struct {
			TotalMonthlyRate float64 `json:"totalMonthlyRate"`
		} `json:"costAfter"`
	}
	require.NoError(t, json.Unmarshal(fixture, &rows))
	require.NotEmpty(t, rows)

	attrs, err := structpb.NewStruct(map[string]any{
		"metadata": map[string]any{"name": "fixed", "namespace": "web"},
		"spec":     map[string]any{"replicas": float64(2)},
	})
	require.NoError(t, err)
	wantBody, err := protojson.Marshal(attrs)
	require.NoError(t, err)

	var mu sync.Mutex
	var got specCostCall
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, readErr := io.ReadAll(r.Body)
		if readErr != nil {
			http.Error(w, readErr.Error(), http.StatusInternalServerError)
			return
		}
		q := r.URL.Query()
		mu.Lock()
		got = specCostCall{
			method: r.Method,
			path:   r.URL.Path,
			query: map[string]string{
				"clusterID":          q.Get("clusterID"),
				"defaultNamespace":   q.Get("defaultNamespace"),
				"windowAvgUsage":     q.Get("windowAvgUsage"),
				"windowResourceCost": q.Get("windowResourceCost"),
				"noUsage":            q.Get("noUsage"),
				"window":             q.Get("window"),
			},
			auth: r.Header.Get("Authorization"),
			body: string(body),
		}
		mu.Unlock()
		if r.Method != http.MethodPost || r.URL.Path != "/model/prediction/speccost" {
			http.Error(w, "wrong endpoint", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture)
	}))
	t.Cleanup(backend.Close)

	cli, err := allocation.NewClient(t.Context(), allocation.Config{
		BaseURL:          backend.URL,
		Profile:          allocation.ProfileKubecost,
		Currency:         "EUR",
		ClusterID:        "production-cluster",
		DefaultNamespace: "web",
		PredictionWindow: "2d",
		APIToken:         "test-token",
	})
	require.NoError(t, err)
	resp, err := server.New(cli).EstimateCost(t.Context(), &pbc.EstimateCostRequest{
		ResourceType: "kubernetes:apps/v1:Deployment",
		Attributes:   attrs,
	})
	require.NoError(t, err)
	require.InDelta(t, rows[0].CostAfter.TotalMonthlyRate, resp.GetCostMonthly(), 1e-9)
	require.Equal(t, "EUR", resp.GetCurrency())

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, http.MethodPost, got.method)
	require.Equal(t, "/model/prediction/speccost", got.path)
	require.Equal(t, "production-cluster", got.query["clusterID"])
	require.Equal(t, "web", got.query["defaultNamespace"])
	require.Equal(t, "2d", got.query["windowAvgUsage"])
	require.Equal(t, "7d offset 48h", got.query["windowResourceCost"])
	require.Equal(t, "false", got.query["noUsage"])
	require.Empty(t, got.query["window"])
	require.Equal(t, "Bearer test-token", got.auth)
	require.JSONEq(t, string(wantBody), got.body)
}

func TestEstimateCostKubecostPredictionNeedsCurrency(t *testing.T) {
	t.Parallel()
	t.Attr("label", "contract-fixture")

	fixture := readRepoFile(t, "testdata/kubecost-contract/speccost.json")
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture)
	}))
	t.Cleanup(backend.Close)
	cli, err := allocation.NewClient(t.Context(), allocation.Config{
		BaseURL:  backend.URL,
		Profile:  allocation.ProfileKubecost,
		Currency: "",
	})
	require.NoError(t, err)
	attrs, err := structpb.NewStruct(map[string]any{
		"metadata": map[string]any{"name": "fixed", "namespace": "web"},
	})
	require.NoError(t, err)
	_, err = server.New(cli).EstimateCost(t.Context(), &pbc.EstimateCostRequest{
		ResourceType: "kubernetes:apps/v1:Deployment",
		Attributes:   attrs,
	})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
	require.NotContains(t, err.Error(), "USD")
}

type specCostCall struct {
	method string
	path   string
	query  map[string]string
	auth   string
	body   string
}
