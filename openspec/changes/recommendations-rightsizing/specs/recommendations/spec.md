# Spec Delta

## Purpose

Return stateless right-sizing recommendations for Kubernetes controllers whose OpenCost request averages exceed usage plus an operator-set headroom, priced from the same allocation row.

## ADDED Requirements

### Requirement: Recommendations are off unless enabled

With `recommendations` off, `GetRecommendations` through the SDK server SHALL return an empty list and no error, and SHALL NOT call the allocation backend.

#### Scenario: Default configuration

- **WHEN** `GetRecommendations` is called through the SDK server with the default configuration
- **THEN** the response has no recommendations and no error
- **AND** the backend receives no request

Tested by: TestRecommendationsOffAnswersEmpty

### Requirement: One recommendation per over-provisioned controller

With `recommendations` on, the plugin SHALL query the 30-day allocation by namespace and controller and return one `ADJUST_REQUESTS` recommendation, category `COST`, for each controller whose CPU or memory request average exceeds usage × (1 + headroom). The recommended request SHALL be that target rounded up to 1m CPU or 1Mi memory, never below one unit.

#### Scenario: Recorded fixed controller

- **WHEN** the recorded controller allocation is served and the headroom is 0.2
- **THEN** `oc-test/fixed` is recommended with current requests `1` CPU and `1Gi` memory and recommended requests `1m` and `3Mi`

Tested by: TestRecommendationsFromRecordedControllers

### Requirement: Savings use the row's observed rates

Savings per hour SHALL be the CPU cut × (`cpuCost` / `cpuCoreHours`) plus the memory cut × (`ramCost` / `ramByteHours`), scaled to `daily` (24), `monthly` (730, the default), or `annual` (8760) hours. Current cost SHALL be `totalCost` per hour scaled the same way, and projected cost SHALL be current minus savings. Currency SHALL follow the actual cost rule.

#### Scenario: Monthly savings for fixed

- **WHEN** the recorded `oc-test/fixed` row is recommended with headroom 0.2
- **THEN** the estimated savings equal the cuts priced at that row's rates over 730 hours, in currency `EUR`

Tested by: TestRecommendationsFromRecordedControllers

### Requirement: Rows with nothing to lower are skipped

A row SHALL NOT produce a recommendation when it has no controller (`__unallocated__`, `__idle__`), no positive minutes, or no resource whose request exceeds its target with a computable rate.

#### Scenario: Controllers without requests

- **WHEN** the recorded allocation has controllers with zero requests, such as `prometheus-server`
- **THEN** none of them is recommended

Tested by: TestRecommendationsSkipRowsWithNothingToLower

### Requirement: The request scope is honoured

The response SHALL drop `excluded_recommendation_ids`, apply `filter`, keep only controllers named in `target_resources` when it is set, sort by savings descending, and page with `page_size` and `page_token`. More than 100 target resources, or a `projection_period` other than empty, `daily`, `monthly`, or `annual`, SHALL be `InvalidArgument`.

#### Scenario: Excluded id and target

- **WHEN** the request excludes `opencost/rightsize/oc-test/fixed` or targets only `kube-system/coredns`
- **THEN** `oc-test/fixed` is absent, and the target request returns only `coredns`

Tested by: TestRecommendationsHonourRequestScope
