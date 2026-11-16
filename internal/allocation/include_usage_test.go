package allocation_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
)

func TestIncludeUsageConfig(t *testing.T) {
	writeConfig := func(t *testing.T, body string) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "config.yaml")
		require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
		return path
	}
	cases := []struct {
		name string
		file string
		env  string
		want bool
	}{
		{name: "default", want: false},
		{name: "file on", file: "includeUsage: true\n", want: true},
		{name: "env on", env: "true", want: true},
		{name: "env overrides file off", file: "includeUsage: true\n", env: "false", want: false},
		{name: "env overrides file on", file: "includeUsage: false\n", env: "true", want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("OPENCOST_INCLUDE_USAGE", tc.env)
			path := ""
			if tc.file != "" {
				path = writeConfig(t, tc.file)
			}
			cfg, err := allocation.LoadConfigFromEnvOrFile(path)
			require.NoError(t, err)
			require.Equal(t, tc.want, cfg.IncludeUsage)
		})
	}

	t.Run("invalid env", func(t *testing.T) {
		t.Setenv("OPENCOST_INCLUDE_USAGE", "maybe")
		_, err := allocation.LoadConfigFromEnvOrFile("")
		require.ErrorContains(t, err, "OPENCOST_INCLUDE_USAGE")
	})
}
