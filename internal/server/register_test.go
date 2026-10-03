package server_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDependencyDashboardIsOwnerClose(t *testing.T) {
	t.Parallel()

	body, err := os.ReadFile(filepath.Join(repoRoot(t), "TASKS.md"))
	require.NoError(t, err)
	text := string(body)
	heading := "## 8. Not delivered"
	start := strings.Index(text, heading)
	require.NotEqual(t, -1, start)
	section := text[start:]
	require.Contains(t, section, "obsolete, Renovate handles it; owner closes")
	require.Contains(t, section, "renovate.json")
	require.Contains(t, section, "config:recommended")
}
