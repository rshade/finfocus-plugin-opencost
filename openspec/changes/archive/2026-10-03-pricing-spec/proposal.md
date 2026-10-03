# Proposal

## Why

The SDK basic suite requires `GetPricingSpec` to return a schema-valid
rate. This plugin can price Kubernetes objects from allocation, and it
cannot invent an AWS EC2 catalog price.

## What Changes

- `GetPricingSpec` returns the observed hourly rate for a supported
  Kubernetes resource.
- Currency follows the allocation body, then pricing config.
- An unsupported type, including `ec2`, stays `InvalidArgument`.
- The conformance adapter sends the suite's `aws`/`ec2` probe as
  namespace `oc-test` over the live gRPC connection.

## Capabilities

### New Capabilities

- `pricing-spec`: Report the observed hourly rate for one Kubernetes resource.

### Modified Capabilities

- None.

## Impact

- `internal/server` `GetPricingSpec`.
- `test/conformance` rewrites only the suite's EC2 probe.
- This supersedes the cost-currency non-goal that left the RPC unimplemented.
- `testdata/opencost-real/` stays read-only.
