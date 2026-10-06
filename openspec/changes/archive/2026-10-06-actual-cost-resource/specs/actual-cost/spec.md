# Spec Delta

## ADDED Requirements

### Requirement: Descriptor selects the allocation object

When `GetActualCostRequest.resource` is set, `GetActualCost` SHALL resolve
the allocation object from that descriptor. Attributes `metadata.name` and
`metadata.namespace` SHALL replace the same tag keys. An unset descriptor
SHALL keep the resource id and tag path. A missing row for a set descriptor
SHALL name the descriptor id in `NO_COST_DATA`.

#### Scenario: Attributes replace tags

- **WHEN** the descriptor id is a Pulumi URN, tags name namespace `tag-ns`, and attributes name namespace `attr-ns`
- **THEN** the query filter is `namespace:"attr-ns"`
- **AND** the returned cost is the `attr-ns` row and differs from the `tag-ns` row

#### Scenario: Unset descriptor keeps the resource id

- **WHEN** `resource` is unset and the resource id is `namespace/tag-ns`
- **THEN** the query filter is `namespace:"tag-ns"`
- **AND** the returned cost is the `tag-ns` row

#### Scenario: Missing descriptor row names the descriptor id

- **WHEN** the descriptor is set, its id is a Pulumi URN, and no row matches
- **THEN** the RPC is `NotFound`
- **AND** `NO_COST_DATA` contains that URN
