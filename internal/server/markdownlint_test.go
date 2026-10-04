package server_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMarkdownlintIgnoresGeneratedChangelog(t *testing.T) {
	t.Parallel()

	body, err := os.ReadFile(filepath.Join(repoRoot(t), ".markdownlint-cli2.jsonc"))
	require.NoError(t, err)

	var kept strings.Builder
	for _, line := range strings.Split(string(body), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "//") {
			continue
		}
		kept.WriteString(line)
		kept.WriteByte('\n')
	}

	var cfg struct {
		Ignores []string `json:"ignores"`
	}
	require.NoError(t, json.Unmarshal([]byte(kept.String()), &cfg))
	require.Contains(t, cfg.Ignores, "CHANGELOG.md")
	require.Contains(t, cfg.Ignores, ".claude/**")
}
