package server_test

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-plugin-opencost/internal/server"
)

func TestHostileNamespaceIsRejected(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":200,"data":[]}`))
	}))
	t.Cleanup(backend.Close)
	cli, err := allocation.NewClient(t.Context(), allocation.Config{BaseURL: backend.URL, Currency: "EUR"})
	require.NoError(t, err)
	srv := server.New(cli)

	for _, name := range []string{`oc"test`, "oc+test", `oc\test`, "oc test", "oc(test", "oc)test"} {
		_, costErr := srv.GetActualCost(t.Context(), actualWindow("namespace/"+name))
		require.Equal(t, codes.InvalidArgument, status.Code(costErr), name)
		require.NotContains(t, costErr.Error(), name)
	}
	require.Zero(t, calls.Load())
}
