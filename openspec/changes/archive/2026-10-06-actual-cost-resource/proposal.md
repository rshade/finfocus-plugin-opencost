# Proposal

## Why

Core v0.4.2 sends `GetActualCostRequest.resource` on actual-cost calls. This plugin still resolves only `resource_id` and tags, so a descriptor whose attributes name a different object is priced as the tag path.

## What Changes

- When `resource` is set, resolve actual cost from that descriptor. `metadata.name` and `metadata.namespace` in attributes replace the same tag keys.
- When `resource` is unset, keep the resource id and tag path.
- A missing row for a set descriptor names the descriptor id in `NO_COST_DATA`.
- No `finfocus-spec` bump. The module already requires v0.7.5.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `actual-cost`: a set descriptor selects the allocation object, and attributes replace tags.

## Impact

- `GetActualCost` in `internal/server` reads `req.GetResource()`.
- The conformance adapter rewrites the suite's aws/ec2 probe on actual cost the same way projected cost already does. A direct `ec2` call stays `InvalidArgument`.
- `testdata/opencost-real/` is unchanged.
