package drift_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type workflowStep struct {
	ID  string            `yaml:"id"`
	If  string            `yaml:"if"`
	Run string            `yaml:"run"`
	Env map[string]string `yaml:"env"`
}

func workflowSteps(t *testing.T, name, job string) []workflowStep {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	body, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "..", ".github", "workflows", name))
	require.NoError(t, err)
	var workflow struct {
		Jobs map[string]struct {
			Steps []workflowStep `yaml:"steps"`
		} `yaml:"jobs"`
	}
	require.NoError(t, yaml.Unmarshal(body, &workflow))
	require.Contains(t, workflow.Jobs, job)
	return workflow.Jobs[job].Steps
}

func stepRunning(steps []workflowStep, fragment string) (workflowStep, bool) {
	for _, step := range steps {
		if strings.Contains(step.Run, fragment) {
			return step, true
		}
	}
	return workflowStep{}, false
}

// TestLiveDriftRunsTheOracleOnTheLatestChart pins the nightly oracle step and its
// issue reporter. kind.yml stays on the versions.env pin and is the regression gate.
func TestLiveDriftRunsTheOracleOnTheLatestChart(t *testing.T) {
	t.Parallel()

	steps := workflowSteps(t, "live-drift.yml", "drift")
	oracle, ok := stepRunning(steps, "-tags e2e")
	require.True(t, ok, "live-drift.yml runs no e2e oracle step")
	require.Equal(t, "oracle", oracle.ID)
	require.Contains(t, oracle.Run, "TestE2EActualCostMatchesOracle")
	require.Equal(t, map[string]string{
		"OC_E2E":           "1",
		"OC_CHART_VERSION": "latest",
		"OC_E2E_CLUSTER":   "oc-drift",
	}, oracle.Env)
	require.Contains(t, oracle.If, "steps.up.outcome == 'success'", "the oracle runs even when the key set differs")

	report, ok := stepRunning(steps, "open-issue.sh oracle")
	require.True(t, ok, "an oracle failure opens no issue")
	require.Contains(t, report.If, "steps.oracle.outcome == 'failure'")
	_, ok = stepRunning(steps, "open-issue.sh keyset")
	require.True(t, ok, "a key-set difference opens no issue")

	for _, step := range workflowSteps(t, "kind.yml", "e2e") {
		require.NotContains(t, step.Env, "OC_CHART_VERSION", "kind.yml must stay on the pinned chart")
	}
}
