# kind-suite Specification

## Purpose

Compare live OpenCost cost for namespace oc-test with the owner oracle, and show that a changed CPU rate falls outside the tolerance.

## Requirements

### Requirement: The kind oracle matches namespace oc-test

When `OC_E2E` is set, the kind suite SHALL use cluster `oc-e2e` and SHALL call
`Supports` and `GetActualCost` on one plugin gRPC connection. `Supports` SHALL
accept `kubernetes:core/v1:Namespace` and SHALL reject
`kubernetes:core/v1:Service`. `GetActualCost` for `namespace/oc-test` SHALL
stay within `relative_tolerance` from `testdata/opencost-real/expected.json`
when the custom CPU rate is `2.0`. The currency SHALL be `EUR`. Setting that
CPU rate to `9.0` SHALL push the relative error higher than the tolerance. Restoring
`2.0` SHALL bring it back within the tolerance. `spotCPU` stays `2.0`. The
suite SHALL skip when `OC_E2E` is unset.

#### Scenario: Oracle rate

- **WHEN** `OC_E2E` is set and the custom CPU rate is `2.0`
- **THEN** `GetActualCost` for `namespace/oc-test` is within `relative_tolerance`
- **AND** the currency is `EUR`
- **AND** `Supports` accepts the namespace type and rejects the service type

#### Scenario: Broken CPU rate

- **WHEN** the custom CPU rate is `9.0` and the oracle still uses the recorded rate
- **THEN** the relative error is higher than `relative_tolerance`

#### Scenario: Restored CPU rate

- **WHEN** the custom CPU rate is restored to `2.0`
- **THEN** `GetActualCost` for `namespace/oc-test` is within `relative_tolerance` again

#### Scenario: Skipped without the live flag

- **WHEN** `OC_E2E` is unset
- **THEN** the kind oracle test skips
