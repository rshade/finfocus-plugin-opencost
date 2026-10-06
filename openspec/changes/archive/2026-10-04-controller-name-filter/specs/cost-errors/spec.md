## MODIFIED Requirements

### Requirement: Cost RPCs use explicit status codes

A cost request with no window SHALL be `InvalidArgument`. An unsupported
resource type SHALL be `InvalidArgument`. An allocation backend status of 500
SHALL be `Unavailable`. A caller deadline SHALL be `DeadlineExceeded` and
SHALL NOT be `OK`. An allocation body with no matching row SHALL be `NotFound`,
and the message SHALL include `NO_COST_DATA` and `no cost data`. An HTTP 500
body that names an invalid filter SHALL stay `Unavailable`, SHALL include the
rejected field name, and SHALL NOT be `NO_COST_DATA`.

#### Scenario: Empty window

- **WHEN** `GetActualCost` is called for `namespace/oc-test` without a window
- **THEN** the status is `InvalidArgument`

#### Scenario: Unsupported type

- **WHEN** `GetProjectedCost` is called for `kubernetes:core/v1:Service`
- **THEN** the status is `InvalidArgument`

#### Scenario: Backend failure

- **WHEN** the allocation backend returns HTTP 500
- **THEN** the status is `Unavailable`

#### Scenario: Invalid filter body

- **WHEN** `GetActualCost` for `controller/oc-test/fixed` receives the captured HTTP 500 body `error-controller-filter.json`
- **THEN** the status is `Unavailable`
- **AND** the message includes `expect filter field` and `controller`
- **AND** the message does not include `NO_COST_DATA`

#### Scenario: Caller deadline

- **WHEN** the caller deadline has already expired
- **THEN** the status is `DeadlineExceeded`
- **AND** the status is not `OK`

#### Scenario: No matching row

- **WHEN** the allocation body has an empty data array
- **THEN** the status is `NotFound`
- **AND** the message includes `NO_COST_DATA` and `no cost data`
