package kind_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func chartVersion(t *testing.T, override string) string {
	t.Helper()
	cmd := exec.Command("bash", filepath.Join(kindDir(t), "chart-version.sh"))
	cmd.Env = append(os.Environ(), "OC_CHART_VERSION="+override)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("chart-version.sh: %v", err)
	}
	return string(out)
}

func TestChartVersionDefaultsToThePin(t *testing.T) {
	t.Parallel()

	pin, err := os.ReadFile(filepath.Join(kindDir(t), "versions.env"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(pin), "\nOPENCOST_CHART_VERSION=2.5.32\n") {
		t.Fatalf("versions.env does not pin OPENCOST_CHART_VERSION: %s", pin)
	}
	if got := chartVersion(t, ""); got != "2.5.32\n" {
		t.Fatalf("default: %q", got)
	}
}

func TestChartVersionOverride(t *testing.T) {
	t.Parallel()

	if got := chartVersion(t, "2.6.0"); got != "2.6.0\n" {
		t.Fatalf("override: %q", got)
	}
	if got := chartVersion(t, "latest"); got != "" {
		t.Fatalf("latest must print nothing so helm omits --version: %q", got)
	}
}

// TestKindScriptsReadTheChartPin keeps the OpenCost chart version in one place,
// where Renovate's regex manager can bump it.
func TestKindScriptsReadTheChartPin(t *testing.T) {
	t.Parallel()

	root := filepath.Join(kindDir(t), "..", "..")
	for _, name := range []string{"hack/kind/e2e-up.sh", "test/e2e/oracle_test.go"} {
		body, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(body), "2.5.32") {
			t.Fatalf("%s hard-codes the chart version; read hack/kind/versions.env", name)
		}
		if !strings.Contains(string(body), "chart-version.sh") {
			t.Fatalf("%s does not call chart-version.sh", name)
		}
	}
}
