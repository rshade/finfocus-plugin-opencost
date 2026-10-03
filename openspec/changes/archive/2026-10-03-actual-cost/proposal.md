# Proposal

## Why

`GetActualCost` must return the recorded total for the requested Kubernetes
object, not every row in the allocation envelope.

## What Changes

- Map `namespace`, `controller`, `pod`, and `node` resource ids to an OpenCost
  filter and aggregate.
- Keep only the allocation rows that match that object, including a real zero.
- Return `NotFound` with `NO_COST_DATA` when no row matches.

## Capabilities

### New Capabilities

- `actual-cost`: Filter recorded allocation envelopes by resource id.

### Modified Capabilities

- None.

## Impact

- `internal/server` `GetActualCost` uses `GetDetailedAllocation`.
- `testdata/opencost-real/` stays read-only.
