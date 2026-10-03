## MODIFIED Requirements

### Requirement: Batch results keep request order

`BatchCost` SHALL query allocation once for a batch of resources of one kind.
`results[i]` SHALL correspond to `resources[i]`. A resource with no allocation
row SHALL be a per-item `NotFound` containing `NO_COST_DATA`, and the RPC
SHALL still succeed, including when neither the body nor the config names a
currency. Estimate costs SHALL use the 730-hour projection of that row. Actual
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

### Requirement: Estimate one recorded namespace

`EstimateCost` SHALL read `metadata.name` for a namespace and return that
namespace's projected monthly cost. `currency` SHALL follow the same rule as
`GetProjectedCost`: the allocation body when it names one currency, otherwise
pricing config, and SHALL NOT assume USD.

#### Scenario: Namespace attributes

- **WHEN** `EstimateCost` is called for `kubernetes:core/v1:Namespace` with `metadata.name` set to a recorded namespace
- **THEN** `cost_monthly` equals that namespace's `totalCost / (minutes / 60) * 730`

#### Scenario: Non-USD config on an estimate

- **WHEN** `EstimateCost` reads `allocation-namespace-60m.json` for `oc-test` and the config currency is `EUR`
- **THEN** `currency` is `EUR`
