## MODIFIED Requirements

### Requirement: Allocation metadata is carried on the result

`GetActualCost` SHALL copy labels, controller kind, and annotations from the
matched allocation onto `ActualCostResult.focus_record`. Labels SHALL be the
focus `tags`. A non-empty controller kind SHALL be extended column
`controllerKind`. Each annotation SHALL be extended column `annotation.<key>`.
Each returned row SHALL set `focus_record.billing_currency` from the allocation
body when that body names one currency, otherwise from pricing config. The
plugin SHALL NOT assume USD. Two different currency values in one body SHALL
be `FailedPrecondition`. When the body and the config both omit a currency,
`GetActualCost` SHALL return `FailedPrecondition`. A row with none of the three
metadata fields SHALL still carry `billing_currency`. An empty match SHALL stay
`NotFound` and SHALL NOT require a currency.

#### Scenario: Recorded pod labels and controller kind

- **WHEN** the resource id is `pod/oc-test/fixed-7996d494d-fbz99` and the body is `allocation-pod-60m.json`
- **THEN** the result tags equal that row's `properties.labels`
- **AND** extended column `controllerKind` equals that row's `properties.controllerKind`
- **AND** no extended column key starts with `annotation.`

#### Scenario: Annotation on the recorded pod

- **WHEN** that same row also has annotation `team` equal to `platform`
- **THEN** extended column `annotation.team` is `platform`
- **AND** the labels and controller kind stay equal to the recorded row

#### Scenario: Non-USD pricing config

- **WHEN** `GetActualCost` reads `allocation-namespace-60m.json`, which has no currency key, and pricing config currency is `EUR`
- **THEN** the result `billing_currency` is `EUR`

#### Scenario: Response currency overrides config

- **WHEN** that body names currency `GBP` and the config currency is `EUR`
- **THEN** `billing_currency` is `GBP`

#### Scenario: Missing currency

- **WHEN** the body names no currency and the config currency is empty
- **THEN** the RPC is `FailedPrecondition`
- **AND** the status message does not contain `USD`
