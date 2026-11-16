package allocation_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKubecostProfileWarnsAtStartup(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "finfocus-plugin-opencost")
	build := exec.Command("go", "build", "-o", bin, "./cmd/finfocus-plugin-opencost")
	build.Dir = moduleRoot(t)
	out, err := build.CombinedOutput()
	require.NoError(t, err, "go build: %s", out)

	const warning = "profile kubecost is experimental"
	kubecost := runStartupFor(t, bin, []string{
		"KUBECOST_BASE_URL=http://127.0.0.1:9",
		"OPENCOST_PROFILE=kubecost",
		"PATH=" + os.Getenv("PATH"),
	}, warning)
	require.Equal(t, 1, strings.Count(kubecost, warning), kubecost)
	require.Contains(t, kubecost, `"level":"warn"`)

	opencost := runStartupFor(t, bin, []string{
		"KUBECOST_BASE_URL=http://127.0.0.1:9",
		"OPENCOST_PROFILE=opencost",
		"PATH=" + os.Getenv("PATH"),
	}, warning)
	require.NotContains(t, opencost, warning)
}
