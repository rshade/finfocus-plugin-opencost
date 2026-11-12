package server_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	requirementHeading = regexp.MustCompile(`(?m)^### Requirement: (.+)$`)
	testedByLine       = regexp.MustCompile(`(?m)^Tested by: (.+)$`)
	goTestFunc         = regexp.MustCompile(`(?m)^func (Test\w+)\(`)
	testName           = regexp.MustCompile(`^Test\w+$`)
)

func repoTestNames(t *testing.T, root string) map[string]bool {
	t.Helper()

	names := map[string]bool{}
	require.NoError(t, filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && (entry.Name() == ".git" || entry.Name() == "node_modules") {
			return filepath.SkipDir
		}
		if entry.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, match := range goTestFunc.FindAllStringSubmatch(string(body), -1) {
			names[match[1]] = true
		}
		return nil
	}))
	return names
}

func TestEverySpecRequirementNamesAResolvingTest(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	specs, err := filepath.Glob(filepath.Join(root, "openspec", "specs", "*", "spec.md"))
	require.NoError(t, err)
	require.NotEmpty(t, specs)
	known := repoTestNames(t, root)

	for _, spec := range specs {
		body, readErr := os.ReadFile(spec)
		require.NoError(t, readErr)
		capability := filepath.Base(filepath.Dir(spec))

		headings := requirementHeading.FindAllSubmatchIndex(body, -1)
		for i, heading := range headings {
			end := len(body)
			if i+1 < len(headings) {
				end = headings[i+1][0]
			}
			block := string(body[heading[1]:end])
			title := string(body[heading[2]:heading[3]])
			label := capability + ": " + title

			lines := testedByLine.FindAllStringSubmatch(block, -1)
			require.Len(t, lines, 1, "%s needs exactly one `Tested by:` line", label)

			value := strings.TrimSpace(lines[0][1])
			if strings.HasPrefix(value, "none (") && strings.HasSuffix(value, ")") {
				continue
			}
			for _, name := range strings.Split(value, ",") {
				name = strings.TrimSpace(name)
				require.Regexp(t, testName, name, "%s: `Tested by:` takes Go test names or `none (reason)`", label)
				require.True(t, known[name], "%s names %s, which is not a func in any _test.go file", label, name)
			}
		}
	}
}

func TestSpecTestLinkCheckerRejectsUnknownNames(t *testing.T) {
	t.Parallel()

	known := repoTestNames(t, repoRoot(t))
	require.True(t, known["TestEverySpecRequirementNamesAResolvingTest"])
	require.False(t, known["TestNoSuchTestAnywhere"])
}
