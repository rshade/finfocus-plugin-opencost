package allocation_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
)

func writeConfigFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	return path
}

func TestConfigFileParseErrorNamesPath(t *testing.T) {
	t.Run("malformed YAML", func(t *testing.T) {
		path := writeConfigFile(t, "baseUrl: [unclosed\n")
		_, err := allocation.LoadConfigFromEnvOrFile(path)
		require.Error(t, err)
		require.Contains(t, err.Error(), path)
	})

	t.Run("wrong YAML type", func(t *testing.T) {
		path := writeConfigFile(t, "timeout: [1, 2]\n")
		_, err := allocation.LoadConfigFromEnvOrFile(path)
		require.Error(t, err)
		require.Contains(t, err.Error(), path)
	})

	t.Run("unknown keys ignored", func(t *testing.T) {
		path := writeConfigFile(t, "notAKey: 1\nbaseUrl: http://file:9090\n")
		cfg, err := allocation.LoadConfigFromEnvOrFile(path)
		require.NoError(t, err)
		require.Equal(t, "http://file:9090", cfg.BaseURL)
	})
}

func TestEnvParseErrorNamesVariable(t *testing.T) {
	cases := []struct {
		name  string
		key   string
		value string
		fail  bool
	}{
		{"timeout without a unit", "KUBECOST_TIMEOUT", "30", true},
		{"timeout with a unit", "KUBECOST_TIMEOUT", "30s", false},
		{"skip verify not a boolean", "KUBECOST_TLS_SKIP_VERIFY", "yes", true},
		{"skip verify numeric", "KUBECOST_TLS_SKIP_VERIFY", "1", true},
		{"skip verify true", "KUBECOST_TLS_SKIP_VERIFY", "true", false},
		{"skip verify false", "KUBECOST_TLS_SKIP_VERIFY", "false", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(tc.key, tc.value)
			_, err := allocation.LoadConfigFromEnvOrFile("")
			if tc.fail {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.key)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestConfigValidate(t *testing.T) {
	valid := allocation.Config{BaseURL: "http://localhost:9090"}
	require.NoError(t, valid.Validate())

	cases := []struct {
		name string
		cfg  allocation.Config
		want string
	}{
		{"empty base URL", allocation.Config{}, "baseUrl"},
		{"non-http scheme", allocation.Config{BaseURL: "ftp://example.com"}, "baseUrl"},
		{"no host", allocation.Config{BaseURL: "http://"}, "baseUrl"},
		{"bare host", allocation.Config{BaseURL: "localhost:9090"}, "baseUrl"},
		{"unknown profile", allocation.Config{BaseURL: "http://h:1", Profile: "opensost"}, "profile"},
		{"lowercase currency", allocation.Config{BaseURL: "http://h:1", Currency: "eur"}, "currency"},
		{"short currency", allocation.Config{BaseURL: "http://h:1", Currency: "EU"}, "currency"},
		{"negative timeout", allocation.Config{BaseURL: "http://h:1", Timeout: -time.Second}, "timeout"},
		{"negative rate", allocation.Config{BaseURL: "http://h:1", RequestsPerSecond: -1}, "requestsPerSecond"},
		{"negative burst", allocation.Config{BaseURL: "http://h:1", RateBurst: -1}, "rateBurst"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.want)
		})
	}

	valids := []struct {
		name string
		cfg  allocation.Config
	}{
		{"empty profile", allocation.Config{BaseURL: "http://h:1"}},
		{"opencost profile", allocation.Config{BaseURL: "http://h:1", Profile: allocation.ProfileOpenCost}},
		{"kubecost profile", allocation.Config{BaseURL: "https://h:1", Profile: allocation.ProfileKubecost}},
		{"uppercase currency", allocation.Config{BaseURL: "http://h:1", Currency: "EUR"}},
		{"zero rate values", allocation.Config{BaseURL: "http://h:1", Timeout: 0, RequestsPerSecond: 0, RateBurst: 0}},
	}
	for _, tc := range valids {
		t.Run("valid "+tc.name, func(t *testing.T) {
			require.NoError(t, tc.cfg.Validate())
		})
	}
}

func TestConfigErrorsNeverContainToken(t *testing.T) {
	const token = "oc73-secret-token"
	t.Setenv("KUBECOST_API_TOKEN", token)

	badFile := writeConfigFile(t, "baseUrl: [unclosed\n")
	_, err := allocation.LoadConfigFromEnvOrFile(badFile)
	require.Error(t, err)
	require.NotContains(t, err.Error(), token)

	t.Setenv("KUBECOST_TIMEOUT", "30")
	_, err = allocation.LoadConfigFromEnvOrFile("")
	require.Error(t, err)
	require.NotContains(t, err.Error(), token)

	t.Setenv("KUBECOST_TIMEOUT", "")
	t.Setenv("KUBECOST_TLS_SKIP_VERIFY", "yes")
	_, err = allocation.LoadConfigFromEnvOrFile("")
	require.Error(t, err)
	require.NotContains(t, err.Error(), token)

	for _, cfg := range []allocation.Config{
		{},
		{BaseURL: "ftp://example.com"},
		{BaseURL: "http://h:1", Profile: "opensost"},
		{BaseURL: "http://h:1", Currency: "eur"},
	} {
		validateErr := cfg.Validate()
		require.Error(t, validateErr)
		require.NotContains(t, validateErr.Error(), token)
	}
}
