# Proposal

## Why

The client always calls Kubecost's `/model/allocation` path and builds filter
parameters from a Go map, so the query string changes between runs and misses
OpenCost's `/allocation` parameters.

## What Changes

- Add an `opencost` profile: `GET /allocation` with `includeIdle` and `shareIdle`, and no required token.
- Add a `kubecost` profile: `GET /model/allocation` with `idle`, `accumulate`, and `shareIdle`, plus a bearer token when one is configured.
- Sort filter keys so the same query builds the same URL every time.
- Default the empty profile to `opencost`.

## Capabilities

### New Capabilities

- `allocation-query`: Choose the allocation endpoint and build a stable query string.

### Modified Capabilities

- None.

## Impact

- `internal/allocation` config and URL builder.
- Existing Kubecost-path tests must select the `kubecost` profile.
- Kind end-to-end calls use the `opencost` profile.
