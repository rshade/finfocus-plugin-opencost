package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-plugin-opencost/pkg/version"
)

func TestVersionFlagsRunTheBuiltBinary(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "finfocus-plugin-opencost")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Env = os.Environ()
	out, err := build.CombinedOutput()
	require.NoError(t, err, "go build: %s", out)

	short := runFlag(t, bin, "-version")
	shortAgain := runFlag(t, bin, "-version")
	require.Equal(t, short, shortAgain)
	require.NotEmpty(t, strings.TrimSpace(short))
	require.Equal(t, version.String()+"\n", short)

	full := runFlag(t, bin, "-version-full")
	fullAgain := runFlag(t, bin, "-version-full")
	require.Equal(t, full, fullAgain)
	require.NotEmpty(t, strings.TrimSpace(full))
	require.Equal(t, version.FullString()+"\n", full)

	unknown := exec.Command(bin, "-not-a-flag")
	require.Error(t, unknown.Run())
}

func runFlag(t *testing.T, bin, name string) string {
	t.Helper()
	proc := exec.Command(bin, name)
	var stdout, stderr bytes.Buffer
	proc.Stdout = &stdout
	proc.Stderr = &stderr
	require.NoError(t, proc.Run(), "%s stderr: %s", name, stderr.String())
	return stdout.String()
}
