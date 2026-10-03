package server_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRemainingBehaviourSpecsExist(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	checks := map[string]string{
		"openspec/specs/plugin-info/spec.md":      "SHALL NOT include `ALLOCATION`",
		"openspec/specs/cost-errors/spec.md":      "NO_COST_DATA",
		"openspec/specs/contract-sources/spec.md": "no upstream module",
		"openspec/specs/kind-suite/spec.md":       "namespace/oc-test",
	}
	for rel, needle := range checks {
		body, err := os.ReadFile(filepath.Join(root, rel))
		require.NoError(t, err, rel)
		require.Contains(t, string(body), needle, rel)
	}
}
