# Proposal

## Why

A namespace allocation sums every workload in it. A stack that prices both the Namespace and a Deployment in it, as the bundled example did, therefore counts the workload twice: `finfocus cost projected` showed $2,458.94 for the namespace and $2,458.95 for its only workload, a $4,917.89 total. Nothing in the response, docs or example said the figures overlap. Issue #149.

## What Changes

- `billing_detail` on a projected namespace row says the figure includes every workload in the namespace.
- The example stack drops the Namespace resource and names the namespace as a string, so its total is not double counted. `preview.json` is regenerated.
- The README, `docs/resource-mapping.md` and the example README state the overlap.
- No change to the cost arithmetic, the allocation query, or the other resource kinds.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `projected-cost`: a namespace row's `billing_detail` states that it includes the namespace's workloads.

## Impact

- `internal/server/project.go`, `internal/server/server.go`, `internal/server/batch.go`, their tests, `examples/kubernetes/`, README, `docs/resource-mapping.md`.
- The text of `billing_detail` changes for namespace rows. Other kinds keep the current text.
- Core still sums every row it receives. De-duplicating by parent needs a spec field and is out of scope here.
