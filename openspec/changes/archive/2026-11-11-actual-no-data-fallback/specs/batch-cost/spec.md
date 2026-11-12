# Spec Delta

## MODIFIED Requirements

### Requirement: Batch results keep request order

`BatchCost` SHALL query allocation once for a batch of resources of one kind.
`results[i]` SHALL correspond to `resources[i]`. On an estimate or projected batch, a
resource with no allocation row SHALL be a per-item `NotFound` containing
`NO_COST_DATA`. On an actual batch it SHALL be per-item `ActualCostData` with
no results and `FALLBACK_HINT_RECOMMENDED`. Either way the RPC SHALL still
succeed, including when neither the body nor the config names a currency. Estimate costs SHALL use the 730-hour projection of that row. Actual
costs SHALL use that row's `totalCost`. A present row SHALL carry the same
currency rule as `GetProjectedCost`. Estimate items SHALL set `currency`.
Actual items SHALL set `focus_record.billing_currency`. A present row whose
body and config both omit a currency SHALL be a per-item `FailedPrecondition`,
and the RPC SHALL still succeed.

#### Scenario: Estimate batch with one missing namespace

- **WHEN** `BatchCost` estimates three namespaces from `allocation-namespace-60m.json` and the middle name is absent
- **THEN** the backend is called once with window `30d` and aggregate `namespace`
- **AND** the first and third results are the projected monthly costs of those namespaces
- **AND** the middle result is `NotFound` with `NO_COST_DATA` and no cost data

#### Scenario: Actual batch order

- **WHEN** `BatchCost` requests actual cost for two namespaces in a chosen order
- **THEN** the result order matches that request
- **AND** each total equals that namespace's recorded `totalCost`

#### Scenario: Missing namespace without a currency

- **WHEN** `BatchCost` estimates `oc-test` and an absent namespace from `allocation-namespace-60m.json` and the config currency is empty
- **THEN** the RPC succeeds
- **AND** the absent namespace is `NotFound` with `NO_COST_DATA`
- **AND** the `oc-test` item is `FailedPrecondition`
- **AND** that item message does not contain `USD`

#### Scenario: Actual batch with one missing namespace

- **WHEN** `BatchCost` requests actual cost for `oc-test` and an absent namespace from `allocation-namespace-60m.json` and the config currency is empty
- **THEN** the RPC succeeds
- **AND** the absent namespace is actual cost data with no results and `FALLBACK_HINT_RECOMMENDED`
- **AND** it carries no per-item error
