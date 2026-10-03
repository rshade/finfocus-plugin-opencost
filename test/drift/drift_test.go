package drift_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/opencost/opencost/core/pkg/opencost"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
)

func TestMarshaledAllocationMatchesWireAndRecordings(t *testing.T) {
	// omitempty drops these maps when they are empty. The idle recording
	// sends both keys, so the marshaled value includes them.
	alloc := &opencost.Allocation{
		RawAllocationOnly: &opencost.RawAllocationOnlyData{},
		ProportionalAssetResourceCosts: opencost.ProportionalAssetResourceCosts{
			"cpu": {},
		},
		SharedCostBreakdown: opencost.SharedCostBreakdowns{
			"cpu": {},
		},
	}
	body, err := json.Marshal(alloc)
	if err != nil {
		t.Fatalf("marshal allocation: %v", err)
	}
	marshaled := objectKeys(t, body)
	wire := wireKeys(t)
	recorded := recordedKeys(t)

	if diff := setDiff("marshaled", marshaled, "wire", wire); diff != "" {
		t.Fatalf("upstream Allocation keys differ from ConsumedFields plus IgnoredFields:\n%s", diff)
	}
	if diff := setDiff("recorded", recorded, "wire", wire); diff != "" {
		t.Fatalf("recorded allocation keys differ from ConsumedFields plus IgnoredFields:\n%s", diff)
	}
}

func wireKeys(t *testing.T) map[string]struct{} {
	t.Helper()
	keys := map[string]struct{}{}
	for key := range allocation.ConsumedFields {
		keys[key] = struct{}{}
	}
	for key := range allocation.IgnoredFields {
		if _, dup := keys[key]; dup {
			t.Fatalf("key %s is both consumed and ignored", key)
		}
		keys[key] = struct{}{}
	}
	return keys
}

func recordedKeys(t *testing.T) map[string]struct{} {
	t.Helper()
	matches, globErr := filepath.Glob(filepath.Join(recordedDir(t), "allocation-*.json"))
	if globErr != nil {
		t.Fatalf("glob recordings: %v", globErr)
	}
	if len(matches) == 0 {
		t.Fatal("no recorded allocation files")
	}
	seen := map[string]struct{}{}
	for _, name := range matches {
		body, readErr := os.ReadFile(name)
		if readErr != nil {
			t.Fatalf("read %s: %v", name, readErr)
		}
		var envelope struct {
			Data []map[string]json.RawMessage `json:"data"`
		}
		if unmarshalErr := json.Unmarshal(body, &envelope); unmarshalErr != nil {
			t.Fatalf("decode %s: %v", name, unmarshalErr)
		}
		for _, step := range envelope.Data {
			for _, raw := range step {
				for key := range objectKeys(t, raw) {
					seen[key] = struct{}{}
				}
			}
		}
	}
	return seen
}

func objectKeys(t *testing.T, body []byte) map[string]struct{} {
	t.Helper()
	var object map[string]json.RawMessage
	if err := json.Unmarshal(body, &object); err != nil {
		t.Fatalf("decode object: %v", err)
	}
	keys := make(map[string]struct{}, len(object))
	for key := range object {
		keys[key] = struct{}{}
	}
	return keys
}

func setDiff(leftName string, left map[string]struct{}, rightName string, right map[string]struct{}) string {
	var onlyLeft, onlyRight []string
	for key := range left {
		if _, ok := right[key]; !ok {
			onlyLeft = append(onlyLeft, key)
		}
	}
	for key := range right {
		if _, ok := left[key]; !ok {
			onlyRight = append(onlyRight, key)
		}
	}
	if len(onlyLeft) == 0 && len(onlyRight) == 0 {
		return ""
	}
	slices.Sort(onlyLeft)
	slices.Sort(onlyRight)
	return fmt.Sprintf(
		"only in %s: %s\nonly in %s: %s",
		leftName,
		strings.Join(onlyLeft, ", "),
		rightName,
		strings.Join(onlyRight, ", "),
	)
}

func recordedDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "opencost-real")
}
