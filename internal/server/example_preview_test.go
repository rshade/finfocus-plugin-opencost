package server_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKubernetesExamplePreviewFixture(t *testing.T) {
	t.Parallel()

	tokens := []string{
		"kubernetes:core/v1:Namespace",
		"kubernetes:apps/v1:Deployment",
		"kubernetes:core/v1:Service",
		"kubernetes:apps/v1:DaemonSet",
	}
	program := string(readRepoFile(t, "examples/kubernetes/Pulumi.yaml"))
	preview := readRepoFile(t, "examples/kubernetes/preview.json")
	require.True(t, json.Valid(preview))
	text := string(preview)
	for _, token := range tokens {
		require.Contains(t, program, token)
		require.Contains(t, text, token)
	}

	var plan struct {
		Steps []struct {
			Type     string `json:"type"`
			NewState struct {
				Type string `json:"type"`
			} `json:"newState"`
		} `json:"steps"`
	}
	require.NoError(t, json.Unmarshal(preview, &plan))
	seen := map[string]bool{}
	for _, step := range plan.Steps {
		// pulumi preview --json puts the type on newState. FinFocus reads that
		// field first and falls back to the step type.
		resourceType := step.NewState.Type
		if resourceType == "" {
			resourceType = step.Type
		}
		seen[resourceType] = true
	}
	for _, token := range tokens {
		require.Truef(t, seen[token], "preview steps missing %s", token)
	}
}
