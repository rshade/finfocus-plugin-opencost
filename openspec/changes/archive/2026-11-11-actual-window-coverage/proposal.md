# Proposal

## Why

OpenCost on kind cluster `oc-e2e` keeps 15 days of Prometheus data. For
namespace `oc-test` over 2026-10-12 to 2026-11-11 the allocation row was
`start=2026-10-27T06:05:00Z`, `end=2026-11-11T00:00:00Z`, `minutes=21235`:
49% of the 43200-minute window. `GetActualCost` returned `totalCost` 1054.50
with nothing that said half the window had no data. Core's Prometheus usage
source warns in the same case.

finfocus-spec v0.7.5 (and main) has no warning or coverage field on
`GetActualCostResponse`, `ActualCostResult`, or `ActualCostData`. The field
that states a cost's period is FOCUS `charge_period_start` and
`charge_period_end` on `focus_record`. Core reads those fields only to
decide whether a bucket is one day. It does not show them.

## What Changes

- Every actual result sets `focus_record.charge_period_start` and
  `charge_period_end` from the row's `start` and `end`.
- When the rows start later or end earlier than the window, the plugin
  reads the cluster-wide span over the same window. `GetActualCost` asks for
  aggregate `cluster` with no filter. The actual `BatchCost` path uses its
  own unfiltered body. A short cluster-wide span logs a `warn` event with the
  window, the data span, and the covered fraction.
- Uncovered time up to one hour, or up to one percent of the window when that
  is longer, does not count. A window that ends after the query is measured to
  the query time.
- The cost is the backend's `totalCost`. It is not extrapolated.

## Non-goals

- Projected cost divides by the row's own `minutes`, so a short span does not
  bias the rate. It gets no coverage signal.
- No change to finfocus-spec or core.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `actual-cost`: results state their charge period, and a backend data gap
  logs a warning.
- `batch-cost`: actual items do the same, from the batch body.

## Impact

- `internal/server`: `window_coverage.go`, `resultsFor`, `GetActualCost`,
  `BatchCost`.
- `README.md` and `CLAUDE.md`.
