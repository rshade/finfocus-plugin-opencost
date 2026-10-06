package drift_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestOpenIssueCreatesThenUpdatesOneIssue(t *testing.T) {
	t.Parallel()

	bin := t.TempDir()
	state := filepath.Join(t.TempDir(), "state")
	stub := filepath.Join(bin, "gh")
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
	if err := os.WriteFile(stub, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	log := "added: none\nremoved: cpuCost\n"
	first := runOpenIssue(t, bin, log)
	if first != "https://github.com/example/repo/issues/67\n" {
		t.Fatalf("create url: %q", first)
	}
	second := runOpenIssue(t, bin, log)
	if second != "https://github.com/example/repo/issues/67\n" {
		t.Fatalf("update url: %q", second)
	}
	body, err := os.ReadFile(state)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if strings.Count(text, "issue create") != 1 {
		t.Fatalf("creates: %s", text)
	}
	if strings.Count(text, "issue comment") != 1 {
		t.Fatalf("comments: %s", text)
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

func runOpenIssue(t *testing.T, bin, log string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller")
	}
	cmd := exec.Command("bash", filepath.Join(filepath.Dir(file), "open-issue.sh"))
	cmd.Stdin = strings.NewReader(log)
	cmd.Env = append(os.Environ(),
		"PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"GITHUB_REPOSITORY=example/repo",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("open-issue: %v %s", err, out)
	}
	return string(out)
}
