package kind_test

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func chartVersion(t *testing.T, override string) string {
	t.Helper()
	cmd := exec.Command("bash", filepath.Join(kindDir(t), "chart-version.sh"))
	cmd.Env = append(os.Environ(), "OC_CHART_VERSION="+override)
	out, err := cmd.Output()
	require.NoError(t, err, "chart-version.sh")
	return string(out)
}

// chartPin reads OPENCOST_CHART_VERSION from versions.env, so a Renovate bump
// needs no test edit.
func chartPin(t *testing.T) string {
	t.Helper()
	file, err := os.Open(filepath.Join(kindDir(t), "versions.env"))
	require.NoError(t, err)
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if value, ok := strings.CutPrefix(scanner.Text(), "OPENCOST_CHART_VERSION="); ok {
			require.NotEmpty(t, value)
			return value
		}
	}
	require.NoError(t, scanner.Err())
	t.Fatal("versions.env does not set OPENCOST_CHART_VERSION")
	return ""
}

func TestChartVersionDefaultsToThePin(t *testing.T) {
	t.Parallel()

	require.Equal(t, chartPin(t)+"\n", chartVersion(t, ""))
}

func TestChartVersionOverride(t *testing.T) {
	t.Parallel()

	require.Equal(t, "2.6.0\n", chartVersion(t, "2.6.0"))
	require.Empty(t, chartVersion(t, "latest"), "latest must print nothing so helm omits --version")
}

// TestKindScriptsReadTheChartPin keeps the OpenCost chart version in one place,
// where Renovate's regex manager can bump it.
func TestKindScriptsReadTheChartPin(t *testing.T) {
	t.Parallel()

	pin := chartPin(t)
	root := filepath.Join(kindDir(t), "..", "..")
	for _, name := range []string{"hack/kind/e2e-up.sh", "test/e2e/oracle_test.go"} {
		body, err := os.ReadFile(filepath.Join(root, name))
		require.NoError(t, err)
		require.NotContains(t, string(body), pin, "%s hard-codes the chart version; read hack/kind/versions.env", name)
		require.Contains(t, string(body), "chart-version.sh", "%s does not call chart-version.sh", name)
	}
}
