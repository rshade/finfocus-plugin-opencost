## MODIFIED Requirements

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
