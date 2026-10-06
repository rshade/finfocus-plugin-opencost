# actual-cost Specification

## Purpose

Return recorded allocation totals for one namespace, controller, pod, or node.

## Requirements

### Requirement: Resource id selects one allocation object

`GetActualCost` SHALL query the OpenCost allocation API with a filter and
aggregate derived from the resource id, and SHALL return only rows for that
object. Each result cost SHALL be that row's typed `totalCost`. A real zero
SHALL be returned. No match SHALL be `NotFound` with `NO_COST_DATA`.
A controller resource SHALL filter on `controllerName` and SHALL aggregate
by `namespace,controller`.

#### Scenario: Namespace total from the recorded envelope

- **WHEN** the resource id is `namespace/oc-test` and the body is `allocation-namespace-60m.json`
- **THEN** the query filter is `namespace:"oc-test"` and the aggregate is `namespace`
- **AND** the response has one result whose cost is 0.1406

#### Scenario: Controller total

- **WHEN** the resource id is `controller/oc-test/fixed` and the body is `allocation-controller-60m.json`
- **THEN** the query filter is `controllerName:"fixed"+namespace:"oc-test"` and the aggregate is `namespace,controller`
- **AND** the response has one result for allocation `oc-test/deployment:fixed`

#### Scenario: Pod total

- **WHEN** the resource id is `pod/oc-test/fixed-7996d494d-fbz99` and the body is `allocation-pod-60m.json`
- **THEN** the query filter is `namespace:"oc-test"+pod:"fixed-7996d494d-fbz99"` and the aggregate is `namespace,pod`
- **AND** the response has one result for that pod

#### Scenario: Node rows from a namespace recording

- **WHEN** the resource id is `node/oc-spike-control-plane` and the body is `allocation-namespace-60m.json`
- **THEN** the query filter is `node:"oc-spike-control-plane"` and the aggregate is `node`
- **AND** the result count and cost sum equal the recorded rows whose `properties.node` is that node

#### Scenario: Per-step namespace totals include a real zero

- **WHEN** the resource id is `namespace/oc-test` and the body is `allocation-namespace-step-1m.json`
- **THEN** each step that contains `oc-test` produces one result, in step order, including a cost of 0

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
