# cost-errors Specification

## Purpose

Map cost RPC failures to explicit gRPC codes, and return a real zero cost instead of hiding it.

## Requirements

### Requirement: Cost RPCs use explicit status codes

A cost request with no window SHALL be `InvalidArgument`. An unsupported
resource type SHALL be `InvalidArgument`. An allocation backend status of 500
SHALL be `Unavailable`. A caller deadline SHALL be `DeadlineExceeded` and
SHALL NOT be `OK`. An allocation body with no matching row SHALL be `NotFound`,
and the message SHALL include `NO_COST_DATA` and `no cost data`.

#### Scenario: Empty window

- **WHEN** `GetActualCost` is called for `namespace/oc-test` without a window
- **THEN** the status is `InvalidArgument`

#### Scenario: Unsupported type

- **WHEN** `GetProjectedCost` is called for `kubernetes:core/v1:Service`
- **THEN** the status is `InvalidArgument`

#### Scenario: Backend failure

- **WHEN** the allocation backend returns HTTP 500
- **THEN** the status is `Unavailable`

#### Scenario: Caller deadline

- **WHEN** the caller deadline has already expired
- **THEN** the status is `DeadlineExceeded`
- **AND** the status is not `OK`

#### Scenario: No matching row

- **WHEN** the allocation body has an empty data array
- **THEN** the status is `NotFound`
- **AND** the message includes `NO_COST_DATA` and `no cost data`

### Requirement: A real zero stays zero

A matching allocation whose total is zero SHALL return one result with cost
zero. A matching nonzero total SHALL be returned unchanged.

#### Scenario: Zero total

- **WHEN** the matching allocation total is 0
- **THEN** the call succeeds
- **AND** the single result cost is 0

#### Scenario: Nonzero total

- **WHEN** the matching allocation total is 1.25
- **THEN** the single result cost is 1.25
