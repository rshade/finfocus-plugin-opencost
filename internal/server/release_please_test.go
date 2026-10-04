package server_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReleasePleaseInitialVersionIs010(t *testing.T) {
	t.Parallel()

	body, err := os.ReadFile(filepath.Join(repoRoot(t), "release-please-config.json"))
	require.NoError(t, err)

	var cfg struct {
		Packages map[string]struct {
			InitialVersion string `json:"initial-version"`
		} `json:"packages"`
	}
	require.NoError(t, json.Unmarshal(body, &cfg))
	pkg, ok := cfg.Packages["."]
	require.True(t, ok)
	require.Equal(t, "0.1.0", pkg.InitialVersion)
}
