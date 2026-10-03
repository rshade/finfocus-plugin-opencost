# Proposal

## Why

The cost, budget, health, and query specs describe the RPCs that read OpenCost
or Kubecost. They do not describe plugin identity, gRPC error mapping, the
Kubecost contract source list, or the kind oracle. Those behaviours already
ship, and the spec set should name them.

## What Changes

- Record `GetPluginInfo` and `Supports` as their own capability.
- Record the gRPC codes for an empty window, an unsupported type, a backend
  failure, a timeout, a miss, and a real zero.
- Record the Kubecost contract source list. Kubecost has no upstream module
  to marshal, and the fixtures are not verified against live Kubecost.
- Record the kind oracle: `GetActualCost` for namespace `oc-test` stays inside
  the tolerance in `testdata/opencost-real/expected.json`, and a CPU rate of
  9.0 falls outside it.

## Capabilities

### New Capabilities

- `plugin-info`: name, spec version, capabilities, and supported resource types.
- `cost-errors`: gRPC status mapping for cost RPCs, including a real zero.
- `contract-sources`: one source row for each Kubecost contract fixture.
- `kind-suite`: the live kind oracle and the fixed-rate break.

### Modified Capabilities

- None.

## Impact

- No production code changes. The existing tests are the proof.
- Main specs gain four capabilities. Issues #18, #21, #14, and #48 stay open.
- The plugin still does not implement AllocatorService and does not report
  `ALLOCATION`.
