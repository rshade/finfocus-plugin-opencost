# Design

## Context

See proposal.md. `GetProjectedCost` already calls `refForDescriptor`, which runs `resolveRef` on `tagsPreferringAttributes`. `GetActualCost` calls `resolveRef("", req.GetResourceId(), req.GetTags())` and never reads `req.GetResource()`. A kind-prefixed id still wins over metadata. An opaque id, including a Pulumi URN, uses `metadata.name` and `metadata.namespace`. An empty name on an opaque id is `InvalidArgument`.

## Goals / Non-Goals

**Goals:**

- A set descriptor selects the allocation object.
- Attributes replace the same flattened tags.
- An unset descriptor keeps today's resource id and tag path.
- Ignoring `resource` fails the descriptor tests, including `RPCCorrectness_GetActualCostWithResource`.

**Non-Goals:**

- A `finfocus-spec` bump.
- A second fallback when `metadata.name` is empty. That case stays `InvalidArgument`.
- Editing `testdata/opencost-real/` or the kind oracle.
- gRPC authentication.

## Decisions

- `GetActualCost` calls `refForDescriptor(req.GetResource())` when the descriptor is set, and the existing `resolveRef` call when it is not. The preference helper is shared with projected cost.
- A missing row uses `correlationID` for a set descriptor, so `NO_COST_DATA` names the descriptor id. An unset descriptor still names `resource_id`.
- The conformance TCP adapter rewrites only the suite aws/ec2 probe on `GetActualCost`, matching projected cost and pricing. The plugin still rejects a direct `ec2` descriptor.
- The untagged conformance test calls the shipped `GetActualCost`. The SDK case accepts a plugin that ignores the descriptor, so the local test asserts the attribute object's cost.

## Risks / Trade-offs

- [A kind-prefixed descriptor id still ignores attributes] → That matches projected cost. The attribute-wins test uses an opaque id.
- [Live basic conformance sends `resource_id` `test-resource` plus an aws/ec2 sample] → The adapter rewrite sends namespace `oc-test` before the plugin sees the probe. The plugin code does not special-case that id.
