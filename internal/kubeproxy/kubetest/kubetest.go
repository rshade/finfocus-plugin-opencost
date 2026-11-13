// Package kubetest builds fake API servers and kubeconfig files for tests.
package kubetest

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Entry is one kubeconfig context with its own cluster and user.
type Entry struct {
	Name   string
	Server string
	// Token is a static bearer token. ExecCommand overrides it.
	Token       string
	ExecCommand string
}

// Serve starts a TLS server that stands in for a Kubernetes API server.
func Serve(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	srv := httptest.NewTLSServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

// WriteKubeconfig writes a kubeconfig whose current context is the first
// entry, and returns its path. Clusters skip TLS verification because the
// fake API server has a throwaway certificate.
func WriteKubeconfig(t *testing.T, entries ...Entry) string {
	t.Helper()
	var b strings.Builder
	b.WriteString("apiVersion: v1\nkind: Config\ncurrent-context: " + entries[0].Name + "\nclusters:\n")
	for _, e := range entries {
		fmt.Fprintf(
			&b,
			"- name: %s\n  cluster:\n    server: %s\n    insecure-skip-tls-verify: true\n",
			e.Name,
			e.Server,
		)
	}
	b.WriteString("users:\n")
	for _, e := range entries {
		fmt.Fprintf(&b, "- name: %s\n  user:\n", e.Name)
		if e.ExecCommand != "" {
			fmt.Fprintf(
				&b,
				"    exec:\n      apiVersion: client.authentication.k8s.io/v1\n      command: %s\n      interactiveMode: Never\n",
				e.ExecCommand,
			)
			continue
		}
		fmt.Fprintf(&b, "    token: %s\n", e.Token)
	}
	b.WriteString("contexts:\n")
	for _, e := range entries {
		fmt.Fprintf(&b, "- name: %s\n  context:\n    cluster: %s\n    user: %s\n", e.Name, e.Name, e.Name)
	}
	path := filepath.Join(t.TempDir(), "kubeconfig")
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// WriteScript writes an executable shell script and returns its path.
func WriteScript(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "plugin.sh")
	//nolint:gosec // an exec credential plugin must be executable
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}
