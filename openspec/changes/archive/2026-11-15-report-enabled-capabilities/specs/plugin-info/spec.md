## MODIFIED Requirements

### Requirement: Plugin identity lists the implemented capabilities

`GetPluginInfo` SHALL report name `opencost`, provider `kubernetes`, and the
SDK spec version. That version SHALL start with `v`. The capability set SHALL
list only what the configured profile can serve: `PROJECTED_COSTS`,
`ACTUAL_COSTS`, `PRICING_SPEC`, `ESTIMATE_COST`, and `BATCH_COST`, plus
`BUDGETS` only on profile `kubecost`. It SHALL NOT include `ALLOCATION`.

#### Scenario: Info response

- **WHEN** `GetPluginInfo` is called with profile `opencost` or no profile
- **THEN** the name is `opencost`
- **AND** the spec version starts with `v` and equals the SDK spec version
- **AND** the capability set is the five cost capabilities
- **AND** `BUDGETS` and `ALLOCATION` are absent

#### Scenario: Profile kubecost

- **WHEN** `GetPluginInfo` is called with profile `kubecost`
- **THEN** the capability set is the five cost capabilities plus `BUDGETS`
- **AND** `ALLOCATION` is absent

Tested by: TestGetPluginInfoReportsSpecAndCapabilities, TestGetPluginInfoCapabilitiesFollowProfile

## ADDED Requirements

### Requirement: The manifest lists every capability the binary can serve

`plugin.manifest.json` SHALL list the superset of capabilities across all
profiles, so it includes `BUDGETS` even though profile `opencost` does not
report it at runtime.

#### Scenario: Manifest superset

- **WHEN** the manifest capabilities are read
- **THEN** they are the five cost capabilities and `BUDGETS`

Tested by: TestDocsLimitKubecostToProfileAndHistory
