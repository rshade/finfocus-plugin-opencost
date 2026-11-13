# Spec Delta

## ADDED Requirements

### Requirement: The kind suite prices oc-test through the service proxy

When `OC_E2E` is set, the kind suite SHALL start the plugin with a `kube`
block for context `kind-oc-e2e`, namespace `opencost`, and service
`opencost`, and no `baseUrl`. `GetActualCost` for `namespace/oc-test` SHALL
stay within `relative_tolerance` from `testdata/opencost-real/expected.json`
and `Check` SHALL report healthy.

#### Scenario: Oracle through the proxy

- **WHEN** `OC_E2E` is set and the plugin uses the `kube` block
- **THEN** `GetActualCost` for `namespace/oc-test` is within
  `relative_tolerance`
- **AND** `Check` reports healthy

#### Scenario: Wrong service name

- **WHEN** `kube.service` names a service that does not exist
- **THEN** a cost call fails with `Unavailable` naming that service

Tested by: TestE2EActualCostThroughServiceProxy
