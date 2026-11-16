# Design

## Context

`allocation.Entry` decodes `cpuCoreRequestAverage`, `cpuCoreUsageAverage`, `ramByteRequestAverage`, and `ramByteUsageAverage` (#136), plus `cpuCost`, `cpuCoreHours`, `ramCost`, `ramByteHours`, `minutes`, and `totalCost`. On a namespace-and-controller aggregate the request and usage averages are totals across the controller's pods. The SDK provides `ApplyRecommendationVisibility`, `ApplyRecommendationFilter`, `PaginateRecommendations`, and `CalculateRecommendationSummary`, and its server answers an `Unimplemented` handler error with an empty response.

## Goals / Non-Goals

**Goals:** one `ADJUST_REQUESTS` recommendation per over-provisioned controller on profile `opencost`, priced only from the same row.

**Non-Goals:** profile `kubecost`; per-container or per-pod requests; limits; dismissal state; `usage_profile` (ignored; the operator's headroom is the one policy input); a confidence score.

## Decisions

1. **Headroom is required, not defaulted.** With `recommendations` on, `recommendationHeadroom` must be above zero, or start-up fails. No headroom constant ships without a documented source (#138 boundary check).
2. **Target = usage average × (1 + headroom), rounded up** to 1m CPU and 1Mi memory, and never below that unit. A zero request would remove the request and change the pod's QoS class, so the smallest recommendation is one unit.
3. **Savings use the row's own unit rates.** CPU rate = `cpuCost / cpuCoreHours`, memory rate = `ramCost / ramByteHours`. Savings per hour = CPU cut × CPU rate + memory cut × memory rate, then × 24, 730, or 8760 for `daily`, `monthly` (default), or `annual`. A resource whose request is at or below its target, or whose rate cannot be computed, is not cut. A controller with no cut is not recommended.
4. **Current and projected cost.** Current = `totalCost / hours × period hours`; projected = current − savings. This is labelled an estimate: OpenCost bills max(request, usage), so the saving assumes the new request is what gets billed.
5. **Identity.** Id `opencost/rightsize/<namespace>/<controller>`; resource id `controller/<namespace>/<controller>`, the same form `GetActualCost` accepts; provider `kubernetes`; resource type from `controllerKind` (`deployment` → `kubernetes:apps/v1:Deployment`, and so on; unknown → `k8s-controller`).
6. **Request scope.** Rows are built, then `target_resources` (controller refs only; other kinds match nothing), excluded ids, the filter, a stable sort by savings descending then id, the summary of that full list, and paging. More than 100 target resources, or an unknown `projection_period`, is `InvalidArgument`.
7. **Off means the SDK default.** With the flag off the handler returns `Unimplemented`, and the SDK server answers with its empty response. A direct caller sees the status.

## Risks / Trade-offs

- [Request averages are controller totals, not per-pod values] → the description and metadata say so; `KubernetesAction.container_name` stays empty.
- [A short-lived controller has few samples] → the 30-day window is the same one projected cost uses; the minutes are in the reasoning lines.
- [`kubecost` users enable the flag] → start-up error names the profile.
