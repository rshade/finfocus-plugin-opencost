# pricing-spec Specification

## Purpose

Report the observed hourly cost of one Kubernetes resource. This is not an AWS price catalog.

## Requirements

### Requirement: Price a supported Kubernetes resource from allocation

`GetPricingSpec` SHALL query allocation window `30d` for the resource in
the descriptor. `provider` SHALL be `kubernetes`. `resource_type` SHALL be
the requested supported type. `billing_mode` SHALL be `per_hour`.
`rate_per_unit` SHALL be that object's `totalCost` divided by its hours
(`minutes / 60`). `currency` SHALL follow the allocation body when it names
one currency, otherwise the pricing config. The plugin SHALL NOT assume USD.
A positive total SHALL NOT return a zero rate. No samples SHALL be
`NotFound`. An unsupported type, including `ec2`, SHALL be `InvalidArgument`
and SHALL NOT return a price.

#### Scenario: Namespace rate matches the recorded hourly cost

- **WHEN** `GetPricingSpec` is called for a namespace in `allocation-namespace-60m.json` and the config currency is `EUR`
- **THEN** `provider` is `kubernetes`
- **AND** `resource_type` is `kubernetes:core/v1:Namespace`
- **AND** `billing_mode` is `per_hour`
- **AND** `currency` is `EUR`
- **AND** `rate_per_unit` equals that namespace's `totalCost / (minutes / 60)`
- **AND** `rate_per_unit` is greater than zero

#### Scenario: EC2 is not a catalog price

- **WHEN** `GetPricingSpec` is called for provider `aws` and resource type `ec2`
- **THEN** the status is `InvalidArgument`
- **AND** the message contains `ec2`
