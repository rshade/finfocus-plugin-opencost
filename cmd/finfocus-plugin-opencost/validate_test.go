package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBinaryExitsOnInvalidConfig(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "finfocus-plugin-opencost")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Env = os.Environ()
	out, err := build.CombinedOutput()
	require.NoError(t, err, "go build: %s", out)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	proc := exec.CommandContext(ctx, bin)
	// No KUBECOST_BASE_URL: validation must stop startup.
	proc.Env = []string{}
	var stderr bytes.Buffer
	proc.Stderr = &stderr
	runErr := proc.Run()
	require.Error(t, runErr, "expected a non-zero exit, stderr: %s", stderr.String())
	require.NotContains(t, stderr.String(), "unsupported protocol scheme")

	var exitErr *exec.ExitError
	require.ErrorAs(t, runErr, &exitErr, "process did not exit on its own: %s", stderr.String())
	require.Equal(t, 1, exitErr.ExitCode())
	require.Contains(t, stderr.String(), "baseUrl")
}
