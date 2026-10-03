# Proposal

## Why

A matched allocation row carries Kubernetes labels, a controller kind, and
sometimes annotations. `GetActualCost` was returning only the cost.

## What Changes

- Copy labels onto `ActualCostResult.focus_record.tags`.
- Copy controller kind to extended column `controllerKind`.
- Copy each annotation to extended column `annotation.<key>`.
- Leave `focus_record` unset when the row has none of those three.
- Do not set a currency on that record.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `actual-cost`: Actual cost results carry allocation metadata.

## Impact

- `internal/server` `GetActualCost` and batch actual results, which share
  `resultsFor`.
- `testdata/opencost-real/` stays unchanged. Recorded rows have no annotations.
