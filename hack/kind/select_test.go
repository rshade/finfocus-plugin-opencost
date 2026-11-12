package kind_test

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSelectPRPaths(t *testing.T) {
	t.Parallel()
	script := filepath.Join(kindDir(t), "select-pr-paths.sh")
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{name: "readme", in: "README.md\n", want: false},
		{name: "server", in: "internal/server/server.go\n", want: true},
		{name: "e2e", in: "docs/readme.md\ntest/e2e/oracle_test.go\n", want: true},
		{name: "kind", in: "hack/kind/workload.yaml\n", want: true},
		{name: "drift script", in: "hack/drift/live-compare.sh\n", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cmd := exec.Command("bash", script)
			cmd.Stdin = strings.NewReader(tc.in)
			err := cmd.Run()
			if tc.want && err != nil {
				t.Fatalf("expected a match: %v", err)
			}
			if !tc.want && err == nil {
				t.Fatal("expected no match")
			}
		})
	}
}

func kindDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller")
	}
	return filepath.Dir(file)
}
