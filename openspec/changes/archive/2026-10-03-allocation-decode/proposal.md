# Proposal

## Why

The allocation client must read the recorded OpenCost envelopes, including
plain-text HTTP 400 bodies, and name every wire key it does not use.

## What Changes

- Decode allocation envelopes into the typed cost fields.
- Return plain-text error bodies as errors instead of JSON syntax errors.
- Declare `ConsumedFields` and `IgnoredFields` so their union matches every
  key in `testdata/opencost-real/`.

## Capabilities

### New Capabilities

- `allocation-decode`: Decode recorded allocation envelopes and account for every wire key.

### Modified Capabilities

- None.

## Impact

- `internal/allocation` decode path and `wirefields.go`.
- `testdata/opencost-real/` stays read-only.
