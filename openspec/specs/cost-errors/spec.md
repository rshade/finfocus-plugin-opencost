# cost-errors Specification

## Purpose

Map cost RPC failures to explicit gRPC codes, and return a real zero cost instead of hiding it.

## Requirements

### Requirement: Cost RPCs use explicit status codes

A cost request with no window SHALL be `InvalidArgument`. An unsupported
resource type SHALL be `InvalidArgument`. An allocation backend status of 500
SHALL be `Unavailable`. A caller deadline SHALL be `DeadlineExceeded` and
SHALL NOT be `OK`.

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

Tested by: TestCostRPCErrorMapping

### Requirement: A missing allocation row is no data

On `GetProjectedCost`, `GetPricingSpec`, and `EstimateCost`, an
allocation body with no matching row SHALL be `NotFound`, and the message
SHALL include `NO_COST_DATA` and `no cost data`. On `GetActualCost` it SHALL
succeed with no results and `FALLBACK_HINT_RECOMMENDED`.

#### Scenario: No matching row

- **WHEN** `GetProjectedCost` reads an allocation body with an empty data array
- **THEN** the status is `NotFound`
- **AND** the message includes `NO_COST_DATA` and `no cost data`

#### Scenario: No matching row on actual cost

- **WHEN** `GetActualCost` reads an allocation body with an empty data array
- **THEN** the call succeeds with no results
- **AND** the fallback hint is `FALLBACK_HINT_RECOMMENDED`

Tested by: TestCostRPCErrorMapping, TestGetActualCostMissingDescriptorIsNoData

### Requirement: Backend failures keep their cause

An HTTP 500
body that names an invalid filter SHALL stay `Unavailable`, SHALL include the
rejected field name, and SHALL NOT be `NO_COST_DATA`. A backend HTTP 401 on
any backend call SHALL be `Unauthenticated` and a backend HTTP 403 SHALL be
`PermissionDenied`; the message SHALL name the `KUBECOST_API_TOKEN` check and
SHALL NOT contain the token. Other 4xx and 5xx statuses SHALL stay
`Unavailable`.

#### Scenario: Invalid filter body

- **WHEN** `GetActualCost` for `controller/oc-test/fixed` receives the captured HTTP 500 body `error-controller-filter.json`
- **THEN** the status is `Unavailable`
- **AND** the message includes `expect filter field` and `controller`
- **AND** the message does not include `NO_COST_DATA`

#### Scenario: Backend rejects the token

- **WHEN** a budget, allocation, prediction, or health call gets HTTP 401
- **THEN** the status is `Unauthenticated`
- **AND** the message names the `KUBECOST_API_TOKEN` check
- **AND** the message does not contain the token

#### Scenario: Backend forbids the token

- **WHEN** a budget, allocation, or prediction call gets HTTP 403
- **THEN** the status is `PermissionDenied`

#### Scenario: Other backend errors stay Unavailable

- **WHEN** a backend call gets HTTP 400, 404, or 500
- **THEN** the status is `Unavailable`

Tested by: TestCapturedControllerFilterBodies, TestBackendAuthMaps401And403, TestBackendAuthErrorOmitsToken, TestBackendStatusErrorIsTyped

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

Tested by: TestCostRPCErrorMapping
