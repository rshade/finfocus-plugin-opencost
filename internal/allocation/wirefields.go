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
		"name":             "allocation name",
		"start":            "window start",
		"end":              "window end",
		"minutes":          "billed minutes",
		"cpuCost":          "CPU cost",
		"ramCost":          "memory cost",
		"gpuCost":          "GPU cost",
		"pvCost":           "persistent volume cost",
		"networkCost":      "network cost",
		"loadBalancerCost": "load balancer cost",
		"sharedCost":       "shared cost",
		"externalCost":     "external cost",
		"totalCost":        "total cost",
		"properties":       "namespace, labels, annotations, and controller kind",
	}

	IgnoredFields = map[string]string{
		"cpuCoreHours":                   reasonUsage,
		"cpuCoreLimitAverage":            reasonUsage,
		"cpuCoreRequestAverage":          reasonUsage,
		"cpuCoreUsageAverage":            reasonUsage,
		"cpuCores":                       reasonUsage,
		"cpuCostAdjustment":              "adjustment already inside cpuCost",
		"cpuCostIdle":                    "idle split already inside cpuCost",
		"cpuEfficiency":                  reasonEfficiency,
		"gpuAllocation":                  "per-GPU breakdown not priced here",
		"gpuCostAdjustment":              "adjustment already inside gpuCost",
		"gpuCostIdle":                    "idle split already inside gpuCost",
		"gpuCount":                       reasonUsage,
		"gpuEfficiency":                  reasonEfficiency,
		"gpuHours":                       reasonUsage,
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
		"ramByteHours":                   reasonUsage,
		"ramByteLimitAverage":            reasonUsage,
		"ramByteRequestAverage":          reasonUsage,
		"ramByteUsageAverage":            reasonUsage,
		"ramBytes":                       reasonUsage,
		"ramCostAdjustment":              "adjustment already inside ramCost",
		"ramCostIdle":                    "idle split already inside ramCost",
		"ramEfficiency":                  reasonEfficiency,
		"rawAllocationOnly":              "duplicate of the typed cost fields",
		"sharedCostBreakdown":            "idle-only shared split, not a billed total",
		"totalEfficiency":                reasonEfficiency,
		"window":                         "start and end are read instead",
	}
)
