package allocation_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
)

func TestTLSVerifyIsOnUnlessSkipped(t *testing.T) {
	t.Parallel()

	backend := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":200,"data":[]}`))
	}))
	t.Cleanup(backend.Close)

	verified, err := allocation.NewClient(t.Context(), allocation.Config{BaseURL: backend.URL})
	require.NoError(t, err)
	_, err = verified.GetDetailedAllocation(t.Context(), allocation.Query{Window: "60m"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "certificate")

	skipped, err := allocation.NewClient(t.Context(), allocation.Config{
		BaseURL:       backend.URL,
		TLSSkipVerify: true,
	})
	require.NoError(t, err)
	_, err = skipped.GetDetailedAllocation(t.Context(), allocation.Query{Window: "60m"})
	require.NoError(t, err)
}

func TestTokenIsEnvOnlyAndAbsentFromLogs(t *testing.T) {
	const envToken = "oc41-env-token"
	const yamlToken = "oc41-yaml-token"
	t.Setenv("KUBECOST_API_TOKEN", "")

	file := writeTokenConfig(t, yamlToken)
	fromFile, err := allocation.LoadConfigFromEnvOrFile(file)
	require.NoError(t, err)
	require.Empty(t, fromFile.APIToken)
	require.False(t, fromFile.TLSSkipVerify)

	t.Setenv("KUBECOST_API_TOKEN", envToken)
	fromEnv, err := allocation.LoadConfigFromEnvOrFile(file)
	require.NoError(t, err)
	require.Equal(t, envToken, fromEnv.APIToken)

	var buf bytes.Buffer
	var gotAuth string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":200,"data":[]}`))
	}))
	t.Cleanup(backend.Close)

	fromEnv.BaseURL = backend.URL
	fromEnv.Profile = allocation.ProfileKubecost
	cli, err := allocation.NewClient(t.Context(), fromEnv)
	require.NoError(t, err)
	cli.SetLogger(zerolog.New(&buf))
	_, err = cli.GetDetailedAllocation(t.Context(), allocation.Query{Window: "60m"})
	require.NoError(t, err)
	require.Equal(t, "Bearer "+envToken, gotAuth)
	require.Contains(t, buf.String(), "allocation request")
	require.NotContains(t, buf.String(), envToken)
	require.NotContains(t, buf.String(), yamlToken)

	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "backend failed", http.StatusInternalServerError)
	}))
	t.Cleanup(failing.Close)
	fromEnv.BaseURL = failing.URL
	failingClient, err := allocation.NewClient(t.Context(), fromEnv)
	require.NoError(t, err)
	_, err = failingClient.GetDetailedAllocation(t.Context(), allocation.Query{Window: "60m"})
	require.Error(t, err)
	require.NotContains(t, err.Error(), envToken)
	require.NotContains(t, err.Error(), yamlToken)
}

func writeTokenConfig(t *testing.T, token string) string {
	t.Helper()
	path := t.TempDir() + "/opencost.yaml"
	body := "apiToken: " + token + "\ntlsSkipVerify: false\nprofile: kubecost\n"
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	return path
}
