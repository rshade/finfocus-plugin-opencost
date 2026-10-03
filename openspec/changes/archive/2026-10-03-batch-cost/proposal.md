# Proposal

## Why

A batch of resources must come back in request order. One missing object
must not fail the whole call, and the plugin must not query allocation once
per resource.

## What Changes

- `BatchCost` reads one allocation window for every resource in the batch.
- Results stay in request order. A missing object is a per-item `NotFound`.
- `EstimateCost` projects one namespace named by `metadata.name`.

## Capabilities

### New Capabilities

- `batch-cost`: Estimate and batch allocation costs in request order.

### Modified Capabilities

- None.

## Impact

- `internal/server` `EstimateCost` and `BatchCost`.
- `GetPluginInfo` reports `BATCH_COST`.
- Currency stays empty until OC-3.8.
