package main

import (
	"bytes"
	"encoding/json"
	"net"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-plugin-opencost/internal/kubeproxy/kubetest"
)

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "finfocus-plugin-opencost")
	out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput()
	require.NoError(t, err, "go build: %s", out)
	return bin
}

func freeTCPPort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func TestKubeCredentialsNeverLogged(t *testing.T) {
	const token = "static-token-that-must-stay-secret"
	bin := buildBinary(t)
	api := kubetest.Serve(t, nil)
	kubeconfig := kubetest.WriteKubeconfig(t, kubetest.Entry{Name: "prod", Server: api.URL, Token: token})

	for _, tc := range []struct {
		name       string
		contextEnv []string
	}{
		{"explicit context", []string{"OPENCOST_KUBE_CONTEXT=prod"}},
		{"current context of the kubeconfig", nil},
	} {
		t.Run("one startup line, "+tc.name, func(t *testing.T) {
			proc := exec.Command(bin, "--port", strconv.Itoa(freeTCPPort(t)))
			proc.Env = []string{
				"KUBECONFIG=" + kubeconfig,
				"OPENCOST_KUBE_NAMESPACE=cost",
				"OPENCOST_KUBE_SERVICE=opencost",
				"OPENCOST_KUBE_PORT=9100",
				"HOME=" + t.TempDir(),
			}
			proc.Env = append(proc.Env, tc.contextEnv...)
			var stderr syncBuffer
			proc.Stderr = &stderr
			require.NoError(t, proc.Start())
			t.Cleanup(func() { _ = proc.Process.Kill(); _ = proc.Wait() })

			deadline := time.Now().Add(10 * time.Second)
			for time.Now().Before(deadline) && !strings.Contains(stderr.String(), "kube_service") {
				time.Sleep(20 * time.Millisecond)
			}

			var named []map[string]any
			for _, line := range strings.Split(strings.TrimSpace(stderr.String()), "\n") {
				var entry map[string]any
				require.NoError(t, json.Unmarshal([]byte(line), &entry), line)
				if _, ok := entry["kube_service"]; ok {
					named = append(named, entry)
				}
			}
			require.Len(t, named, 1, stderr.String())
			require.Equal(t, "prod", named[0]["kube_context"])
			require.Equal(t, "cost", named[0]["kube_namespace"])
			require.Equal(t, "opencost", named[0]["kube_service"])
			require.EqualValues(t, 9100, named[0]["kube_port"])
			require.NotContains(t, stderr.String(), token)
		})
	}

	t.Run("failed start", func(t *testing.T) {
		proc := exec.Command(bin)
		proc.Env = []string{
			"KUBECONFIG=" + kubeconfig,
			"OPENCOST_KUBE_CONTEXT=prod-east",
			"OPENCOST_KUBE_SERVICE=opencost",
			"HOME=" + t.TempDir(),
		}
		var stderr syncBuffer
		proc.Stderr = &stderr
		err := proc.Run()
		var exitErr *exec.ExitError
		require.ErrorAs(t, err, &exitErr)
		require.Equal(t, 1, exitErr.ExitCode())
		require.Contains(t, stderr.String(), "prod-east")
		require.NotContains(t, stderr.String(), token)
	})
}
