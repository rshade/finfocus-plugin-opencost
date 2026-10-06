package allocation_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
)

func TestBackendStatusErrorIsTyped(t *testing.T) {
	t.Parallel()

	denied := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "denied", http.StatusUnauthorized)
	}))
	t.Cleanup(denied.Close)
	cli, clientErr := allocation.NewClient(t.Context(), allocation.Config{BaseURL: denied.URL})
	require.NoError(t, clientErr)

	cases := []struct {
		name string
		call func() error
		want string
	}{
		{
			name: "budgets",
			call: func() error { _, err := cli.ListBudgets(t.Context()); return err },
			want: "budget API error: status=401",
		},
		{
			name: "allocation",
			call: func() error {
				_, err := cli.GetDetailedAllocation(t.Context(), allocation.Query{Window: "60m"})
				return err
			},
			want: "allocation API error: status=401, body=denied",
		},
		{
			name: "prediction",
			call: func() error {
				_, err := cli.PredictSpecCost(t.Context(), allocation.PredictionRequest{WorkloadSpec: "{}"})
				return err
			},
			want: "prediction API error: status=401",
		},
		{
			name: "health",
			call: func() error { return cli.Probe(t.Context()) },
			want: "allocation health: status=401",
		},
		{
			name: "decode",
			call: func() error {
				_, err := allocation.DecodeAllocationBody(http.StatusUnauthorized, []byte("denied"))
				return err
			},
			want: "allocation API error: status=401, body=denied",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			callErr := tc.call()
			require.Error(t, callErr)
			require.Equal(t, tc.want, callErr.Error(), "message text must not change")

			var statusErr *allocation.StatusError
			require.ErrorAs(t, callErr, &statusErr, "error must carry the HTTP status")
			require.Equal(t, http.StatusUnauthorized, statusErr.Code)

			code, ok := allocation.HTTPStatus(callErr)
			require.True(t, ok)
			require.Equal(t, http.StatusUnauthorized, code)
		})
	}

	t.Run("budget envelope code", func(t *testing.T) {
		t.Parallel()
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":403,"data":null}`))
		}))
		t.Cleanup(backend.Close)
		envCli, envErr := allocation.NewClient(t.Context(), allocation.Config{BaseURL: backend.URL})
		require.NoError(t, envErr)
		_, listErr := envCli.ListBudgets(t.Context())
		require.Error(t, listErr)
		require.Equal(t, "budget API error: status=403", listErr.Error())
		code, ok := allocation.HTTPStatus(listErr)
		require.True(t, ok)
		require.Equal(t, http.StatusForbidden, code)
	})

	t.Run("non-status errors have no code", func(t *testing.T) {
		t.Parallel()
		_, ok := allocation.HTTPStatus(errors.New("boom"))
		require.False(t, ok)
	})
}
