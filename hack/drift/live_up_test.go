package drift_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func writeStub(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestLiveUpWaitsForOpenCostBeforePortForward(t *testing.T) {
	t.Parallel()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}
	script := filepath.Join(filepath.Dir(file), "live-up.sh")

	bin := t.TempDir()
	calls := filepath.Join(t.TempDir(), "calls")
	writeStub(t, bin, "kind", "exit 0\n")
	writeStub(t, bin, "helm", "exit 0\n")
	writeStub(t, bin, "kubectl",
		"printf '%s\\n' \"$*\" >> "+calls+"\n"+
			"case \"$*\" in\n"+
			"  *port-forward*) echo 'Forwarding from 127.0.0.1:18003 -> 9003'; sleep 30 ;;\n"+
			"esac\n")
	writeStub(t, bin, "curl",
		"while [ $# -gt 0 ]; do\n"+
			"  if [ \"$1\" = -o ]; then\n"+
			"    echo '{\"data\":[{\"oc-test\":{\"name\":\"oc-test\"}}]}' > \"$2\"\n"+
			"  fi\n"+
			"  shift\n"+
			"done\n"+
			"printf 200\n")

	cmd := exec.Command("bash", script)
	cmd.Env = append(os.Environ(),
		"PATH="+bin+":"+os.Getenv("PATH"),
		"KUBECONFIG="+filepath.Join(t.TempDir(), "kubeconfig"),
		"OC_KEEP_FAILED_CLUSTER=1",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("live-up.sh failed: %v\n%s", err, out)
	}

	body, err := os.ReadFile(calls)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(body)), "\n")

	forward := -1
	for i, line := range lines {
		if strings.Contains(line, "port-forward") {
			forward = i
			break
		}
	}
	if forward < 0 {
		t.Fatalf("no port-forward call recorded:\n%s", body)
	}
	before := strings.Join(lines[:forward], "\n")
	for _, want := range []string{
		"rollout status deployment/fixed",
		"wait --for=condition=Ready pod -l app.kubernetes.io/instance=opencost",
	} {
		if !strings.Contains(before, want) {
			t.Errorf("no %q before the port-forward, so the forward can hit a Pending pod:\n%s", want, body)
		}
	}
}
