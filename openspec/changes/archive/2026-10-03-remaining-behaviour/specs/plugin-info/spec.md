# Spec Delta

## Purpose

Report the plugin name, the SDK spec version, the implemented capabilities, and which Kubernetes resource types the cost source accepts.

## ADDED Requirements

### Requirement: Plugin identity lists the implemented capabilities

`GetPluginInfo` SHALL report name `opencost`, provider `kubernetes`, and the
SDK spec version. That version SHALL start with `v`. The capability set SHALL
be `PROJECTED_COSTS`, `ACTUAL_COSTS`, `PRICING_SPEC`, `ESTIMATE_COST`,
`BATCH_COST`, and `BUDGETS`. It SHALL NOT include `ALLOCATION`.

#### Scenario: Info response

- **WHEN** `GetPluginInfo` is called
- **THEN** the name is `opencost`
- **AND** the spec version starts with `v` and equals the SDK spec version
- **AND** the capability set is those six values
- **AND** `ALLOCATION` is absent

### Requirement: Supports accepts the Kubernetes types core sends

`Supports` SHALL accept `kubernetes:core/v1:Namespace`,
`kubernetes:core/v1:Pod`, `kubernetes:core/v1:Node`,
`kubernetes:apps/v1:Deployment`, `kubernetes:apps/v1:StatefulSet`,
`kubernetes:apps/v1:DaemonSet`, `kubernetes:apps/v1:ReplicaSet`,
`kubernetes:batch/v1:Job`, `kubernetes:batch/v1:CronJob`, `k8s-namespace`,
`k8s-pod`, `k8s-controller`, and `k8s-node`. Any other type, including
`kubernetes:core/v1:Service`, SHALL be unsupported and SHALL return a reason.

#### Scenario: Deployment is supported

- **WHEN** `Supports` is called for `kubernetes:apps/v1:Deployment` on provider `kubernetes`
- **THEN** the response says the type is supported

#### Scenario: Service is declined

- **WHEN** `Supports` is called for `kubernetes:core/v1:Service`
- **THEN** the response says the type is not supported
- **AND** the reason is not empty
