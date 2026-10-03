# Proposal

## Why

Recorded allocation bodies do not name a currency, and `provenance/oc-values.yaml`
sets rates without a currency code. Returning costs with an assumed USD would
label those rates as dollars.

## What Changes

- Read `currency` from the allocation envelope and from each entry.
- Use that value when every named currency agrees.
- Otherwise use pricing config `currency` or `OPENCOST_CURRENCY`.
- Return `FailedPrecondition` when both are missing. Do not assume USD.
- Set projected and estimate `currency`, and actual `focus_record.billing_currency`.
- Keep a missing object as `NotFound` without requiring a currency.
- Leave `unit_price` as the observed hourly rate. Do not invent a unit string.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `actual-cost`: Actual results carry billing currency.
- `projected-cost`: Projected results name their currency.
- `batch-cost`: Batch and estimate results use the same currency rule.

## Impact

- `internal/allocation` config and allocation decode.
- `internal/server` actual, projected, estimate, and batch responses.
- `testdata/opencost-real/` stays unchanged. Those files have no currency key.
