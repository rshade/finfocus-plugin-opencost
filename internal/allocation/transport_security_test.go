package allocation_test

import (
	"bytes"
	"encoding/pem"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
)

const insecureHTTPError = "base URL uses http; use https or set allowInsecureHttp"

func TestInsecureHTTPRejectsNonLoopback(t *testing.T) {
	rejected := allocation.Config{BaseURL: "http://example.com"}
	err := rejected.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), insecureHTTPError)

	allowed := []string{
		"http://localhost",
		"http://localhost:9090",
		"http://127.0.0.1:9090",
		"http://127.1.2.3:9090",
		"http://[::1]:9090",
		"https://example.com",
		"https://example.com:443",
	}
	for _, baseURL := range allowed {
		t.Run(baseURL, func(t *testing.T) {
			require.NoError(t, allocation.Config{BaseURL: baseURL}.Validate())
		})
	}

	optIn := allocation.Config{BaseURL: "http://example.com", AllowInsecureHTTP: true}
	require.NoError(t, optIn.Validate())

	t.Setenv("KUBECOST_BASE_URL", "http://example.com")
	t.Setenv("KUBECOST_ALLOW_INSECURE_HTTP", "")
	loaded, loadErr := allocation.LoadConfigFromEnvOrFile("")
	require.NoError(t, loadErr)
	require.ErrorContains(t, loaded.Validate(), insecureHTTPError)

	t.Setenv("KUBECOST_ALLOW_INSECURE_HTTP", "true")
	loaded, loadErr = allocation.LoadConfigFromEnvOrFile("")
	require.NoError(t, loadErr)
	require.True(t, loaded.AllowInsecureHTTP)
	require.NoError(t, loaded.Validate())
}

func TestTLSSkipVerifyWarnsAtStartup(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "finfocus-plugin-opencost")
	build := exec.Command("go", "build", "-o", bin, "./cmd/finfocus-plugin-opencost")
	build.Dir = moduleRoot(t)
	out, err := build.CombinedOutput()
	require.NoError(t, err, "go build: %s", out)

	const warning = "TLS certificate verification is disabled"
	const token = "oc72-startup-token"
	const userinfo = "oc72-url-secret"

	on := runStartup(t, bin, []string{
		"KUBECOST_BASE_URL=http://alice:" + userinfo + "@127.0.0.1:9",
		"KUBECOST_TLS_SKIP_VERIFY=true",
		"KUBECOST_API_TOKEN=" + token,
		"PATH=" + os.Getenv("PATH"),
	})
	require.Equal(t, 1, strings.Count(on, warning))
	require.NotContains(t, on, token)
	require.NotContains(t, on, userinfo)

	off := runStartup(t, bin, []string{
		"KUBECOST_BASE_URL=http://127.0.0.1:9",
		"KUBECOST_TLS_SKIP_VERIFY=false",
		"KUBECOST_API_TOKEN=" + token,
		"PATH=" + os.Getenv("PATH"),
	})
	require.Equal(t, 0, strings.Count(off, warning))
	require.NotContains(t, off, token)
}

func TestCACertFileTrustsPrivateCA(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	without, err := allocation.NewClient(t.Context(), allocation.Config{BaseURL: server.URL})
	require.NoError(t, err)
	require.Error(t, without.Probe(t.Context()))

	pemPath := filepath.Join(t.TempDir(), "ca.pem")
	var body bytes.Buffer
	require.NoError(t, pem.Encode(&body, &pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}))
	require.NoError(t, os.WriteFile(pemPath, body.Bytes(), 0o600))

	withCA, err := allocation.NewClient(t.Context(), allocation.Config{
		BaseURL:    server.URL,
		CACertFile: pemPath,
	})
	require.NoError(t, err)
	require.NoError(t, withCA.Probe(t.Context()))

	missing := allocation.Config{BaseURL: "https://example.com", CACertFile: filepath.Join(t.TempDir(), "missing.pem")}
	require.ErrorContains(t, missing.Validate(), "caCertFile")

	garbage := filepath.Join(t.TempDir(), "garbage.pem")
	require.NoError(t, os.WriteFile(garbage, []byte("not a certificate"), 0o600))
	bad := allocation.Config{BaseURL: "https://example.com", CACertFile: garbage}
	require.ErrorContains(t, bad.Validate(), "caCertFile")
}

func runStartup(t *testing.T, bin string, env []string) string {
	t.Helper()
	port := freePort(t)
	proc := exec.Command(bin, "--port", port)
	proc.Env = env
	var stderr bytes.Buffer
	proc.Stderr = &stderr
	require.NoError(t, proc.Start())
	deadline := time.Now().Add(3 * time.Second)
	const warning = "TLS certificate verification is disabled"
	for time.Now().Before(deadline) && !strings.Contains(stderr.String(), warning) {
		time.Sleep(20 * time.Millisecond)
	}
	if !strings.Contains(stderr.String(), warning) {
		time.Sleep(1500 * time.Millisecond)
	}
	if proc.Process != nil {
		_ = proc.Process.Kill()
	}
	_ = proc.Wait()
	return stderr.String()
}

func freePort(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	_, port, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	return port
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
