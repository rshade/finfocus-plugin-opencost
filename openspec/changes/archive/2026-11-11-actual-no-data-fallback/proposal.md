# Proposal

## Why

`GetActualCost` returned `NotFound` with `NO_COST_DATA` when OpenCost had no
allocation row for the object. FinFocus core treats every gRPC error as a
failed plugin call, so `finfocus cost actual` listed an absent workload under
ERRORS. The finfocus-spec SDK defines a no-data shape for actual cost: no
results and `FALLBACK_HINT_RECOMMENDED` (`pluginsdk.WithFallbackHint`,
`pluginsdk.WithResults`, and the `ADVANCED_PATTERNS.md` table, which lists an
error as the wrong answer for "no billing data available"). Core reads empty
results as no data.

## What Changes

- `GetActualCost` with no matching row returns
  `pluginsdk.NewActualCostResponse` with no results and
  `FALLBACK_HINT_RECOMMENDED`. It does not ask for a currency.
- An actual `BatchCost` item with no row is `ActualCostData` with no results
  and `FALLBACK_HINT_RECOMMENDED`, not a per-item `NotFound`.
- `GetProjectedCost`, `GetPricingSpec`, `EstimateCost`, and estimate or
  projected batch items keep `NotFound` with `NO_COST_DATA`. Their responses
  have no fallback hint, and core prices every OK response, so an empty
  response would be a silent zero.
- Backend failures, timeouts, and currency errors stay gRPC errors.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `actual-cost`: no match is the SDK no-data response.
- `cost-errors`: `NotFound` for a missing row applies to projected, pricing,
  and estimate only.
- `batch-cost`: an actual item with no row carries the no-data response.

## Impact

- `internal/server`: `GetActualCost`, `actualItem`, and a `noActualData`
  helper.
- `README.md` and `CLAUDE.md`: API notes, batch notes, troubleshooting.
