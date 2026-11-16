package server

import (
	"strconv"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const idleAllocation = "__idle__"

// withUsageColumns adds the row's efficiency and request/usage averages as
// extended columns. Idle capacity has no requests or usage, so the __idle__ row
// gets none. efficiency.gpu needs a GPU on the row, or a zero would read as waste.
func withUsageColumns(record *pbc.FocusCostRecord, key string, entry allocation.Entry) *pbc.FocusCostRecord {
	if key == idleAllocation {
		return record
	}
	if record == nil {
		record = &pbc.FocusCostRecord{}
	}
	if record.ExtendedColumns == nil {
		record.ExtendedColumns = map[string]string{}
	}
	columns := map[string]float64{
		"efficiency.cpu":    entry.CPUEfficiency,
		"efficiency.ram":    entry.RAMEfficiency,
		"efficiency.total":  entry.TotalEfficiency,
		"request.cpu_cores": entry.CPURequestAverage,
		"usage.cpu_cores":   entry.CPUUsageAverage,
		"request.ram_bytes": entry.RAMRequestAverage,
		"usage.ram_bytes":   entry.RAMUsageAverage,
	}
	if entry.GPUCount > 0 || entry.GPUHours > 0 {
		columns["efficiency.gpu"] = entry.GPUEfficiency
	}
	for name, value := range columns {
		record.ExtendedColumns[name] = strconv.FormatFloat(value, 'g', -1, 64)
	}
	return record
}
