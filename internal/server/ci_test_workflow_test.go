package server_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type workflowStep struct {
	Name string            `yaml:"name"`
	Uses string            `yaml:"uses"`
	Run  string            `yaml:"run"`
	With map[string]string `yaml:"with"`
}

type testWorkflow struct {
	Jobs map[string]struct {
		Strategy map[string]any `yaml:"strategy"`
		Steps    []workflowStep `yaml:"steps"`
	} `yaml:"jobs"`
}

func loadTestWorkflow(t *testing.T) testWorkflow {
	t.Helper()

	var workflow testWorkflow
	require.NoError(t, yaml.Unmarshal(readRepoFile(t, ".github/workflows/test.yml"), &workflow))
	require.Contains(t, workflow.Jobs, "test")
	return workflow
}

func TestCITestJobRunsRaceAndPublishesCoverage(t *testing.T) {
	t.Parallel()

	steps := loadTestWorkflow(t).Jobs["test"].Steps
	var runs []string
	var uploadsCoverage bool
	for _, step := range steps {
		runs = append(runs, step.Run)
		if step.With["path"] == "coverage.out" {
			uploadsCoverage = true
		}
	}

	require.Contains(t, runs, "go test -race -covermode=atomic -coverprofile=coverage.out ./...")
	require.Contains(t, strings.Join(runs, "\n"), "$GITHUB_STEP_SUMMARY")
	require.True(t, uploadsCoverage, "coverage.out is uploaded as an artifact")
}

func TestCITestJobHasNoGoMatrix(t *testing.T) {
	t.Parallel()

	require.NotContains(t, loadTestWorkflow(t).Jobs["test"].Strategy, "matrix",
		"go.mod supports one Go version, so a matrix adds runs and no coverage")
}
