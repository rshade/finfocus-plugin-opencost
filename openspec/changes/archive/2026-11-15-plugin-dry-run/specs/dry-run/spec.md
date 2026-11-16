# Spec Delta

## Purpose

Let `finfocus plugin inspect opencost <type>` read which FOCUS fields the plugin fills for a Kubernetes resource type, and whether its configuration is valid, without calling the allocation backend.

## ADDED Requirements

### Requirement: DryRun answers through the SDK server

The plugin SHALL answer the `DryRun` RPC instead of `Unimplemented`. A request with no resource SHALL be `InvalidArgument` with no response, both through the SDK server and when the handler is called directly.

#### Scenario: plugin inspect request

- **WHEN** the SDK server receives `DryRun` with only resource type `kubernetes:apps/v1:Deployment`
- **THEN** no error is returned
- **AND** `resource_type_supported` is true and the field mappings are not empty

#### Scenario: No resource

- **WHEN** `DryRun` receives a nil request or a request with no resource
- **THEN** the status is `InvalidArgument` and the response is nil

Tested by: TestDryRunThroughSDKServer, TestDryRunRejectsMissingResource

### Requirement: FOCUS field mappings for a supported type

For every type `Supports` accepts, `DryRun` SHALL return one mapping per name in the SDK FOCUS field list, with no duplicates. `billing_currency` SHALL be `SUPPORTED`. `charge_period_start`, `charge_period_end`, `tags`, and `extended_columns` SHALL be `CONDITIONAL` with a non-empty condition. Every other field, including `billed_cost`, SHALL be `UNSUPPORTED`, because actual cost carries the amount outside the FOCUS record.

#### Scenario: Deployment mappings

- **WHEN** `DryRun` is called for `kubernetes:apps/v1:Deployment`
- **THEN** `billing_currency` is `SUPPORTED`
- **AND** `tags` is `CONDITIONAL` and names the OpenCost labels
- **AND** `billed_cost` and `region_id` are `UNSUPPORTED`

#### Scenario: Namespace tags

- **WHEN** `DryRun` is called for `kubernetes:core/v1:Namespace`
- **THEN** the `tags` condition names `namespaceLabels`

Tested by: TestDryRunMapsFocusFieldsForEverySupportedType

### Requirement: Unsupported type is a response, not an error

A type `Supports` declines, including `kubernetes:core/v1:Service` and `aws:ec2/instance:Instance`, SHALL return `resource_type_supported=false`, no field mappings, and no gRPC error.

#### Scenario: Service

- **WHEN** `DryRun` is called for `kubernetes:core/v1:Service`
- **THEN** no error is returned
- **AND** `resource_type_supported` is false and there are no field mappings

Tested by: TestDryRunUnsupportedTypeIsAResponse

### Requirement: Configuration validity comes from start-up validation

`configuration_valid` SHALL be the result of the config validation the binary runs before it serves, applied to the configuration as the operator gave it. When that validation fails, `configuration_errors` SHALL hold its message. A kube target SHALL report a valid configuration even though requests go to the API server origin.

#### Scenario: Invalid profile

- **WHEN** the server was built on a configuration with profile `bogus`
- **THEN** `configuration_valid` is false
- **AND** `configuration_errors` names the profile

#### Scenario: Kube target

- **WHEN** the client was built for `kube.service` through the API server proxy
- **THEN** `configuration_valid` is true and `configuration_errors` is empty

Tested by: TestDryRunReportsInvalidConfiguration, TestKubeClientKeepsOperatorConfig

### Requirement: DryRun never calls the backend

`DryRun` SHALL NOT send any request to the allocation backend, for a supported type, an unsupported type, or an invalid configuration.

#### Scenario: Every supported type

- **WHEN** `DryRun` is called for every supported type against a backend that counts requests
- **THEN** the backend receives no request

Tested by: TestDryRunMakesNoBackendCall
