package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeOpenCost serves the recorded namespace and idle allocations and the
// recorded plain-text 400 bodies, the way OpenCost answered when they were captured.
func fakeOpenCost(t *testing.T, errorContentType string) *httptest.Server {
	t.Helper()
	read := func(name string) []byte {
		body, err := os.ReadFile(filepath.Join(recordedDir, name))
		require.NoError(t, err)
		return body
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		switch {
		case q.Get("window") == "":
			w.Header().Set("Content-Type", errorContentType)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write(read("error-no-window.json"))
		case q.Get("window") == "notawindow":
			w.Header().Set("Content-Type", errorContentType)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write(read("error-bad-window.json"))
		case q.Get("includeIdle") == "true":
			_, _ = w.Write(read("allocation-namespace-idle.json"))
		default:
			_, _ = w.Write(read("allocation-namespace-60m.json"))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func copyAllocations(t *testing.T, extra string) string {
	t.Helper()
	dir := t.TempDir()
	matches, err := filepath.Glob(filepath.Join(recordedDir, "allocation-*.json"))
	require.NoError(t, err)
	for _, name := range matches {
		body, readErr := os.ReadFile(name)
		require.NoError(t, readErr)
		require.NoError(t, os.WriteFile(filepath.Join(dir, filepath.Base(name)), body, 0o600))
	}
	if extra != "" {
		require.NoError(t, os.WriteFile(filepath.Join(dir, "allocation-extra.json"), []byte(extra), 0o600))
	}
	return dir
}

func TestRunRequiresExactlyOneSource(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	require.ErrorContains(t, run(&out, "", recordedDir, ""), "exactly one")
	require.ErrorContains(t, run(&out, "http://127.0.0.1:9", recordedDir, t.TempDir()), "exactly one")
	require.Empty(t, out.String())
}

func TestRunResponsesMatchingTheRecordings(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	require.NoError(t, run(&out, "", recordedDir, copyAllocations(t, "")))
	require.Equal(t, "added: none\nremoved: none\nkeys match (55)\n", out.String())
}

func TestRunResponsesWithANewKeyFail(t *testing.T) {
	t.Parallel()

	extra := `{"code":200,"data":[{"oc-test":{"newCostField":1}}]}`
	var out bytes.Buffer
	err := run(&out, "", recordedDir, copyAllocations(t, extra))
	require.ErrorContains(t, err, "keys differ")
	require.Equal(t, "added: newCostField\nremoved: none\n", out.String())
}

func TestRunAgainstALiveBase(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	require.NoError(t, run(&out, fakeOpenCost(t, "text/plain; charset=utf-8").URL, recordedDir, ""))
	require.Equal(t, "added: none\nremoved: none\nkeys match (55)\n", out.String())
}

func TestRunAgainstALiveBaseRejectsJSONErrors(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	err := run(&out, fakeOpenCost(t, "application/json").URL, recordedDir, "")
	require.ErrorContains(t, err, "error-bad-window.json")
	require.ErrorContains(t, err, "not plain text")
	require.Equal(t, "added: none\nremoved: none\n", out.String())
}

func TestRunAgainstALiveBaseReportsAFailedQuery(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	var out bytes.Buffer
	require.ErrorContains(t, run(&out, srv.URL, recordedDir, ""), "status 500")
	require.Empty(t, out.String())
}
