package drift_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
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
	require.NoError(t, os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755))
	return bin, state
}

func readState(t *testing.T, state string) string {
	t.Helper()
	body, err := os.ReadFile(state)
	require.NoError(t, err)
	return string(body)
}

const runURL = "Run: https://github.com/example/repo/actions/runs/4242"

func TestOpenIssueCreatesThenUpdatesOneIssue(t *testing.T) {
	t.Parallel()

	bin, state := stubGH(t)
	log := "added: none\nremoved: cpuCost\n"
	require.Equal(t, "https://github.com/example/repo/issues/67\n", runOpenIssue(t, bin, log), "create url")
	require.Equal(t, "https://github.com/example/repo/issues/67\n", runOpenIssue(t, bin, log), "update url")
	text := readState(t, state)
	require.Equal(t, 1, strings.Count(text, "issue create"), text)
	require.Equal(t, 1, strings.Count(text, "issue comment"), text)
	require.Contains(t, text, "--title live drift: allocation key set differs from testdata/opencost-real")
	require.Contains(t, text, "--label bug", "create has no bug label")
	require.Equal(t, 2, strings.Count(text, runURL), "create and comment must each carry the run URL")
}

func TestOpenIssueIgnoresAMatchingKeySet(t *testing.T) {
	t.Parallel()

	bin := t.TempDir()
	stub := filepath.Join(bin, "gh")
	require.NoError(t, os.WriteFile(stub, []byte("#!/bin/sh\nexit 99\n"), 0o755))
	require.Equal(t, "no key-set difference\n", runOpenIssue(t, bin, "added: none\nremoved: none\nkeys match (55)\n"))
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
	require.Equal(t, 1, strings.Count(text, "issue create"), text)
	require.Equal(t, 1, strings.Count(text, "issue comment"), text)
	require.Contains(t, text, "--title live oracle: cost differs from testdata/opencost-real/expected.json")
	require.Contains(t, text, "either a plugin regression or a deliberate OpenCost pricing change")
	require.Contains(t, text, "oracle match relative=2.3", "oracle body carries the test output")
	require.Equal(t, 2, strings.Count(text, runURL), "create and comment must each carry the run URL")
}

func TestOpenIssueIgnoresAPassingOracle(t *testing.T) {
	t.Parallel()

	bin := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(bin, "gh"), []byte("#!/bin/sh\nexit 99\n"), 0o755))
	got := runOpenIssue(t, bin, "--- PASS: TestE2EActualCostMatchesOracle (200.00s)\nok\n", "oracle")
	require.Equal(t, "no oracle failure\n", got)
}

func runOpenIssue(t *testing.T, bin, log string, args ...string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime caller")
	cmd := exec.Command("bash", append([]string{filepath.Join(filepath.Dir(file), "open-issue.sh")}, args...)...)
	cmd.Stdin = strings.NewReader(log)
	cmd.Env = append(os.Environ(),
		"PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"GITHUB_REPOSITORY=example/repo",
		"GITHUB_SERVER_URL=https://github.com",
		"GITHUB_RUN_ID=4242",
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "open-issue: %s", out)
	return string(out)
}
