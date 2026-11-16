package allocation

const (
	reasonUsage        = "usage statistic, not a billed total"
	reasonEfficiency   = "efficiency is not a cost"
	reasonNetworkSplit = "network split already inside networkCost"
)

// ConsumedFields are the allocation object keys this plugin reads.
// IgnoredFields holds every other key the recorded OpenCost responses contain.
//
//nolint:gochecknoglobals // key sets compared with every recorded allocation
var (
	ConsumedFields = map[string]string{
		"name":                  "allocation name",
		"start":                 "window start",
		"end":                   "window end",
		"minutes":               "billed minutes",
		"cpuCost":               "CPU cost",
		"ramCost":               "memory cost",
		"gpuCost":               "GPU cost",
		"pvCost":                "persistent volume cost",
		"networkCost":           "network cost",
		"loadBalancerCost":      "load balancer cost",
		"sharedCost":            "shared cost",
		"externalCost":          "external cost",
		"totalCost":             "total cost",
		"properties":            "namespace, labels, annotations, and controller kind",
		"cpuCoreRequestAverage": "includeUsage column request.cpu_cores",
		"cpuCoreUsageAverage":   "includeUsage column usage.cpu_cores",
		"ramByteRequestAverage": "includeUsage column request.ram_bytes",
		"ramByteUsageAverage":   "includeUsage column usage.ram_bytes",
		"cpuEfficiency":         "includeUsage column efficiency.cpu",
		"ramEfficiency":         "includeUsage column efficiency.ram",
		"gpuEfficiency":         "includeUsage column efficiency.gpu",
		"totalEfficiency":       "includeUsage column efficiency.total",
		"gpuCount":              "whether the row has a GPU, for efficiency.gpu",
		"gpuHours":              "whether the row has a GPU, for efficiency.gpu",
		"cpuCoreHours":          "per-core-hour rate for recommendation savings",
		"ramByteHours":          "per-byte-hour rate for recommendation savings",
	}

	IgnoredFields = map[string]string{
		"cpuCoreLimitAverage":            reasonUsage,
		"cpuCores":                       reasonUsage,
		"cpuCostAdjustment":              "adjustment already inside cpuCost",
		"cpuCostIdle":                    "idle split already inside cpuCost",
		"gpuAllocation":                  "per-GPU breakdown not priced here",
		"gpuCostAdjustment":              "adjustment already inside gpuCost",
		"gpuCostIdle":                    "idle split already inside gpuCost",
		"lbAllocations":                  "per-balancer breakdown not priced here",
		"loadBalancerCostAdjustment":     "adjustment already inside loadBalancerCost",
		"networkCostAdjustment":          "adjustment already inside networkCost",
		"networkCrossRegionCost":         reasonNetworkSplit,
		"networkCrossZoneCost":           reasonNetworkSplit,
		"networkInternetCost":            reasonNetworkSplit,
		"networkNatGatewayEgressCost":    reasonNetworkSplit,
		"networkNatGatewayIngressCost":   reasonNetworkSplit,
		"networkReceiveBytes":            "byte count, not a cost",
		"networkTransferBytes":           "byte count, not a cost",
		"proportionalAssetResourceCosts": "idle-only asset split, not a billed total",
		"pvByteHours":                    reasonUsage,
		"pvBytes":                        reasonUsage,
		"pvCostAdjustment":               "adjustment already inside pvCost",
		"pvs":                            "per-volume breakdown not priced here",
		"ramByteLimitAverage":            reasonUsage,
		"ramBytes":                       reasonUsage,
		"ramCostAdjustment":              "adjustment already inside ramCost",
		"ramCostIdle":                    "idle split already inside ramCost",
		"rawAllocationOnly":              "duplicate of the typed cost fields",
		"sharedCostBreakdown":            "idle-only shared split, not a billed total",
		"window":                         "start and end are read instead",
	}
)
