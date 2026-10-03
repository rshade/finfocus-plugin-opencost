## Context

`GetPricingSpec` returned `Unimplemented` with message `OC-2.1`. The
archived cost-currency design left a currency catalog out of scope.
The SDK basic suite calls the RPC for `aws` / `ec2` / `t3.micro` /
`us-east-1` and fails on any error. Validation checks the schema, not
that the response names EC2.

## Goals / Non-Goals

**Goals:**

- A supported Kubernetes descriptor returns `per_hour`, the observed
  hourly rate, and the same currency rule as projected cost.
- `rate_per_unit` is `totalCost / (minutes / 60)` from the 30-day window.
- No samples stay `NotFound`.
- `ec2` and any other unsupported type stay `InvalidArgument`.

**Non-Goals:**

- An AWS, Azure, or GCP price catalog.
- A zero rate when the allocation total is positive.
- Claiming the plugin prices EC2.

## Decisions

- The rate is the same hourly figure `GetProjectedCost` returns as
  `unit_price`. Billing mode is `per_hour` because that figure is cost
  per hour for the object, not a per-core list price.
- Provider is `kubernetes` even when the caller omits it. The resource
  type is the caller's supported type.
- The conformance TCP adapter rewrites only `provider=aws` and
  `resource_type=ec2` to namespace `oc-test`. `Supports` is not rewritten,
  so the suite still sees aws as unsupported. A direct EC2 call to the
  plugin is unchanged.

## Risks / Trade-offs

- The hourly rate mixes CPU, RAM, and other allocation components. It is
  the object's observed cost, not a published SKU.
- A 30-day window on a young cluster still prices the minutes that exist.
