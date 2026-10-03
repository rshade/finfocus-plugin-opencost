# Proposal

## Why

`GetProjectedCost` must price the requested Kubernetes object. Averaging the
whole cluster hides which workload the caller asked about.

## What Changes

- Query a 30-day allocation window for the resource named by the descriptor.
- Project that object's observed hourly rate to a 730-hour month.
- State that method in `billing_detail` and split the month across
  `cost_breakdown`.

## Capabilities

### New Capabilities

- `projected-cost`: Project one resource's trailing allocation to a month.

### Modified Capabilities

- None.

## Impact

- `internal/server` `GetProjectedCost`.
- Currency stays empty until OC-3.8. Do not assume USD.
- `testdata/opencost-real/` stays read-only.
