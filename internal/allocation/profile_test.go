package allocation_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
)

func TestAllocationURLProfilesAreStable(t *testing.T) {
	t.Parallel()

	query := allocation.Query{
		Window: "60m",
		Filter: map[string]string{
			"namespace": "oc-test",
			"cluster":   "kind",
		},
		AggregateBy: []string{"namespace"},
	}

	opencost, err := allocation.NewClient(t.Context(), allocation.Config{
		BaseURL: "http://opencost.local:9003",
		Profile: allocation.ProfileOpenCost,
	})
	require.NoError(t, err)

	kubecost, err := allocation.NewClient(t.Context(), allocation.Config{
		BaseURL: "http://kubecost.local:9090",
		Profile: allocation.ProfileKubecost,
	})
	require.NoError(t, err)

	const wantOpenCost = "http://opencost.local:9003/allocation?aggregate=namespace&filter=cluster%3A%22kind%22%2Bnamespace%3A%22oc-test%22&includeIdle=false&shareIdle=false&window=60m"
	const wantKubecost = "http://kubecost.local:9090/model/allocation?accumulate=false&aggregate=namespace&filter=cluster%3A%22kind%22%2Bnamespace%3A%22oc-test%22&idle=false&shareIdle=false&window=60m"

	first, err := opencost.BuildAllocationURL(query)
	require.NoError(t, err)
	second, err := opencost.BuildAllocationURL(query)
	require.NoError(t, err)
	require.Equal(t, wantOpenCost, first)
	require.Equal(t, first, second)

	first, err = kubecost.BuildAllocationURL(query)
	require.NoError(t, err)
	second, err = kubecost.BuildAllocationURL(query)
	require.NoError(t, err)
	require.Equal(t, wantKubecost, first)
	require.Equal(t, first, second)
}

func TestProfileSendsTokenOnlyForKubecost(t *testing.T) {
	t.Parallel()

	var gotAuth string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":200,"data":[]}`))
	}))
	t.Cleanup(backend.Close)

	query := allocation.Query{Window: "60m"}
	opencost, err := allocation.NewClient(t.Context(), allocation.Config{
		BaseURL:  backend.URL,
		APIToken: "secret",
		Profile:  allocation.ProfileOpenCost,
	})
	require.NoError(t, err)
	_, err = opencost.GetDetailedAllocation(t.Context(), query)
	require.NoError(t, err)
	require.Empty(t, gotAuth)

	kubecost, err := allocation.NewClient(t.Context(), allocation.Config{
		BaseURL:  backend.URL,
		APIToken: "secret",
		Profile:  allocation.ProfileKubecost,
	})
	require.NoError(t, err)
	_, err = kubecost.GetDetailedAllocation(t.Context(), query)
	require.NoError(t, err)
	require.Equal(t, "Bearer secret", gotAuth)
}
