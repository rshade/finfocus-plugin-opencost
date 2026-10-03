package server_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
	"unicode"

	"github.com/stretchr/testify/require"
)

func TestDocsLimitKubecostToProfileAndHistory(t *testing.T) {
	t.Parallel()

	names := []string{"CLAUDE.md", "plugin.manifest.json"}
	roadmap := repoPath(t, "ROADMAP.md")
	_, err := os.Stat(roadmap)
	if err == nil {
		names = append(names, "ROADMAP.md")
	} else {
		require.ErrorIs(t, err, os.ErrNotExist)
	}

	mention := regexp.MustCompile(`(?i)kubecost`)
	for _, name := range names {
		body := string(readRepoFile(t, name))
		for _, loc := range mention.FindAllStringIndex(body, -1) {
			start := loc[0]
			require.Truef(t, kubecostMentionAllowed(body, start),
				"%s still names the product: %q", name, mentionWindow(body, start))
		}
	}

	var manifest struct {
		Name         string   `json:"name"`
		Capabilities []string `json:"capabilities"`
	}
	require.NoError(t, json.Unmarshal(readRepoFile(t, "plugin.manifest.json"), &manifest))
	require.Equal(t, "opencost", manifest.Name)
	require.Equal(t, []string{
		"PROJECTED_COSTS",
		"ACTUAL_COSTS",
		"PRICING_SPEC",
		"ESTIMATE_COST",
		"BATCH_COST",
		"BUDGETS",
	}, manifest.Capabilities)
}

func kubecostMentionAllowed(body string, start int) bool {
	const tokenLen = len("kubecost")
	token := body[start : start+tokenLen]
	if token == "KUBECOST" && start+tokenLen < len(body) && body[start+tokenLen] == '_' {
		return true
	}
	if token != "kubecost" {
		return false
	}
	before := start == 0 || !isNameByte(rune(body[start-1]))
	after := start+tokenLen == len(body) || !isNameByte(rune(body[start+tokenLen]))
	return before && after
}

func isNameByte(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func mentionWindow(body string, start int) string {
	from := start - 24
	if from < 0 {
		from = 0
	}
	to := start + len("kubecost") + 24
	if to > len(body) {
		to = len(body)
	}
	return body[from:to]
}

func repoPath(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Join(filepath.Dir(file), "..", "..", name)
}
