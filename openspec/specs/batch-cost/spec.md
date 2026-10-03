# batch-cost Specification

## Purpose

Estimate one resource and batch many resources from one allocation query.

## Requirements

### Requirement: Batch results keep request order

`BatchCost` SHALL query allocation once for a batch of resources of one kind.
`results[i]` SHALL correspond to `resources[i]`. A resource with no allocation
row SHALL be a per-item `NotFound` containing `NO_COST_DATA`, and the RPC
SHALL still succeed. Estimate costs SHALL use the 730-hour projection of that
row. Actual costs SHALL use that row's `totalCost`.

#### Scenario: Estimate batch with one missing namespace

- **WHEN** `BatchCost` estimates three namespaces from `allocation-namespace-60m.json` and the middle name is absent
- **THEN** the backend is called once with window `30d` and aggregate `namespace`
- **AND** the first and third results are the projected monthly costs of those namespaces
- **AND** the middle result is `NotFound` with `NO_COST_DATA` and no cost data

#### Scenario: Actual batch order

- **WHEN** `BatchCost` requests actual cost for two namespaces in a chosen order
- **THEN** the result order matches that request
- **AND** each total equals that namespace's recorded `totalCost`

### Requirement: Estimate one recorded namespace

`EstimateCost` SHALL read `metadata.name` for a namespace and return that
namespace's projected monthly cost.

#### Scenario: Namespace attributes

- **WHEN** `EstimateCost` is called for `kubernetes:core/v1:Namespace` with `metadata.name` set to a recorded namespace
- **THEN** `cost_monthly` equals that namespace's `totalCost / (minutes / 60) * 730`
