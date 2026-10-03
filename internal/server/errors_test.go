package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-plugin-opencost/internal/server"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func TestCostRPCErrorMapping(t *testing.T) {
	t.Parallel()

	t.Run("empty window", func(t *testing.T) {
		t.Parallel()
		srv := newServer(t, "http://127.0.0.1:9")
		_, err := srv.GetActualCost(t.Context(), &pbc.GetActualCostRequest{
			ResourceId: "namespace/oc-test",
		})
		require.Error(t, err)
		require.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("unsupported type", func(t *testing.T) {
		t.Parallel()
		srv := newServer(t, "http://127.0.0.1:9")
		_, err := srv.GetProjectedCost(t.Context(), &pbc.GetProjectedCostRequest{
			Resource: &pbc.ResourceDescriptor{ResourceType: "kubernetes:core/v1:Service"},
		})
		require.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("backend failure", func(t *testing.T) {
		t.Parallel()
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "backend down", http.StatusInternalServerError)
		}))
		t.Cleanup(backend.Close)
		srv := newServer(t, backend.URL)
		_, err := srv.GetActualCost(t.Context(), actualRequest())
		require.Equal(t, codes.Unavailable, status.Code(err))
	})

	t.Run("timeout", func(t *testing.T) {
		t.Parallel()
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
			http.Error(w, "slow", http.StatusGatewayTimeout)
		}))
		t.Cleanup(backend.Close)
		srv := newServer(t, backend.URL)
		ctx, cancel := context.WithTimeout(t.Context(), time.Nanosecond)
		t.Cleanup(cancel)
		time.Sleep(time.Millisecond)
		_, err := srv.GetActualCost(ctx, actualRequest())
		require.Equal(t, codes.DeadlineExceeded, status.Code(err))
		require.NotEqual(t, codes.OK, status.Code(err))
	})

	t.Run("no match", func(t *testing.T) {
		t.Parallel()
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":200,"data":[]}`))
		}))
		t.Cleanup(backend.Close)
		srv := newServer(t, backend.URL)
		_, err := srv.GetActualCost(t.Context(), actualRequest())
		require.Equal(t, codes.NotFound, status.Code(err))
		require.Contains(t, err.Error(), "NO_COST_DATA")
		require.Contains(t, err.Error(), "no cost data")
	})

	t.Run("real zero", func(t *testing.T) {
		t.Parallel()
		srv := newServer(t, allocationBackend(t, 0))
		resp, err := srv.GetActualCost(t.Context(), actualRequest())
		require.NoError(t, err)
		require.Len(t, resp.GetResults(), 1)
		require.InDelta(t, 0.0, resp.GetResults()[0].GetCost(), 1e-9)
	})

	t.Run("nonzero is not rewritten to zero", func(t *testing.T) {
		t.Parallel()
		srv := newServer(t, allocationBackend(t, 1.25))
		resp, err := srv.GetActualCost(t.Context(), actualRequest())
		require.NoError(t, err)
		require.InDelta(t, 1.25, resp.GetResults()[0].GetCost(), 1e-9)
	})
}

func newServer(t *testing.T, baseURL string) *server.Server {
	t.Helper()
	cli, err := allocation.NewClient(t.Context(), allocation.Config{BaseURL: baseURL, Currency: "EUR"})
	require.NoError(t, err)
	return server.New(cli)
}

func actualRequest() *pbc.GetActualCostRequest {
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	return &pbc.GetActualCostRequest{
		ResourceId: "namespace/oc-test",
		Start:      timestamppb.New(start),
		End:        timestamppb.New(end),
	}
}

func allocationBackend(t *testing.T, total float64) string {
	t.Helper()
	body := `{"code":200,"data":[{"oc-test":{"name":"oc-test","window":{"start":"2026-10-03T10:00:00Z"},"totalCost":0}}]}`
	if total != 0 {
		body = `{"code":200,"data":[{"oc-test":{"name":"oc-test","window":{"start":"2026-10-03T10:00:00Z"},"totalCost":1.25}}]}`
	}
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(backend.Close)
	return backend.URL
}
