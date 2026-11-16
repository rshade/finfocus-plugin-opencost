package drift_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// stubGH writes a gh stand-in that records each call in the returned state file.
// issue list finds an open issue only after issue create has run.
func stubGH(t *testing.T) (string, string) {
	t.Helper()
	bin := t.TempDir()
	state := filepath.Join(t.TempDir(), "state")
	script := "#!/bin/sh\n" +
		"state=" + state + "\n" +
		"printf '%s\\n' \"$*\" >> \"${state}\"\n" +
		"if [ \"$1\" = issue ] && [ \"$2\" = list ]; then\n" +
		"  if [ -f \"${state}.number\" ]; then\n" +
		"    echo 67\n" +
		"  fi\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = issue ] && [ \"$2\" = create ]; then\n" +
		"  echo 67 > \"${state}.number\"\n" +
		"  echo https://github.com/example/repo/issues/67\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = issue ] && [ \"$2\" = comment ]; then\n" +
		"  exit 0\n" +
		"fi\n" +
		"echo unexpected: \"$*\" >&2\n" +
		"exit 1\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin, state
}

func readState(t *testing.T, state string) string {
	t.Helper()
	body, err := os.ReadFile(state)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

const runURL = "Run: https://github.com/example/repo/actions/runs/4242"

func TestOpenIssueCreatesThenUpdatesOneIssue(t *testing.T) {
	t.Parallel()

	bin, state := stubGH(t)
	log := "added: none\nremoved: cpuCost\n"
	first := runOpenIssue(t, bin, log)
	if first != "https://github.com/example/repo/issues/67\n" {
		t.Fatalf("create url: %q", first)
	}
	second := runOpenIssue(t, bin, log)
	if second != "https://github.com/example/repo/issues/67\n" {
		t.Fatalf("update url: %q", second)
	}
	text := readState(t, state)
	if strings.Count(text, "issue create") != 1 {
		t.Fatalf("creates: %s", text)
	}
	if strings.Count(text, "issue comment") != 1 {
		t.Fatalf("comments: %s", text)
	}
	if !strings.Contains(text, "--title live drift: allocation key set differs from testdata/opencost-real") {
		t.Fatalf("key-set title: %s", text)
	}
	if !strings.Contains(text, "--label bug") {
		t.Fatalf("create has no bug label: %s", text)
	}
	if strings.Count(text, runURL) != 2 {
		t.Fatalf("create and comment must each carry the run URL: %s", text)
	}
}

func TestOpenIssueIgnoresAMatchingKeySet(t *testing.T) {
	t.Parallel()

	bin := t.TempDir()
	stub := filepath.Join(bin, "gh")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\nexit 99\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := runOpenIssue(t, bin, "added: none\nremoved: none\nkeys match (55)\n")
	if got != "no key-set difference\n" {
		t.Fatalf("got %q", got)
	}
}

func TestOpenIssueReportsAnOracleFailure(t *testing.T) {
	t.Parallel()

	bin, state := stubGH(t)
	log := "=== RUN   TestE2EActualCostMatchesOracle\n" +
		"    oracle_test.go:98: oracle match relative=2.3 got=0.1 want=0.03\n" +
		"--- FAIL: TestE2EActualCostMatchesOracle (200.00s)\nFAIL\n"
	runOpenIssue(t, bin, log, "oracle")
	runOpenIssue(t, bin, log, "oracle")

	text := readState(t, state)
	if strings.Count(text, "issue create") != 1 || strings.Count(text, "issue comment") != 1 {
		t.Fatalf("want one create and one comment: %s", text)
	}
	if !strings.Contains(text, "--title live oracle: cost differs from testdata/opencost-real/expected.json") {
		t.Fatalf("oracle title: %s", text)
	}
	if !strings.Contains(text, "either a plugin regression or a deliberate OpenCost pricing change") {
		t.Fatalf("oracle body must say what a failure means: %s", text)
	}
	if !strings.Contains(text, "oracle match relative=2.3") || strings.Count(text, runURL) != 2 {
		t.Fatalf("oracle body must carry the test output and run URL: %s", text)
	}
}

func TestOpenIssueIgnoresAPassingOracle(t *testing.T) {
	t.Parallel()

	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte("#!/bin/sh\nexit 99\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := runOpenIssue(t, bin, "--- PASS: TestE2EActualCostMatchesOracle (200.00s)\nok\n", "oracle")
	if got != "no oracle failure\n" {
		t.Fatalf("got %q", got)
	}
}

func runOpenIssue(t *testing.T, bin, log string, args ...string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller")
	}
	cmd := exec.Command("bash", append([]string{filepath.Join(filepath.Dir(file), "open-issue.sh")}, args...)...)
	cmd.Stdin = strings.NewReader(log)
	cmd.Env = append(os.Environ(),
		"PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"GITHUB_REPOSITORY=example/repo",
		"GITHUB_SERVER_URL=https://github.com",
		"GITHUB_RUN_ID=4242",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("open-issue: %v %s", err, out)
	}
	return string(out)
}
