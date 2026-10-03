## Context

`BatchCost` lists `ResourceDescriptor` values. `EstimateCost` sends a Pulumi
type and a struct of attributes. Core does not send a separate resource id on
an estimate. A namespace name is `metadata.name`.

The SDK calls `BatchCost` when the plugin implements `BatchCostHandler`.
Otherwise it fans out and would query once per resource.

## Goals / Non-Goals

**Goals:**

- One `/allocation` request for a non-empty batch of one kind.
- `results[i]` matches `resources[i]`.
- A missing row is `NotFound` with `NO_COST_DATA` on that item. The RPC succeeds.
- Estimate monthly cost uses the same 730-hour projection as `GetProjectedCost`.

**Non-Goals:**

- Assuming USD.
- Dry-run field mappings.
- A mixed-kind batch that needs two aggregates. Kinds that differ are queried
  once without an aggregate.

## Decisions

- Estimate and projected batches use window `30d`. Actual batches use the
  request start and end.
- The query has no filter. The server selects each row locally.
- `MaxBatchSize` is the SDK default.

## Risks / Trade-offs

- An actual batch of namespaces uses the namespace aggregate, so pod rows are
  not in that response.

## Migration Plan

- None. Both RPCs replace unimplemented stubs.

## Open Questions

- None.
