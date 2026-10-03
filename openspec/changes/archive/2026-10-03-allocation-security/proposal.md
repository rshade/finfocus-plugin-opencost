# Proposal

## Why

The HTTP client ignored `tlsSkipVerify`, a YAML file could set the API token,
and a namespace containing a filter metacharacter was sent through as a filter.

## What Changes

- Verify TLS certificates unless `tlsSkipVerify` is set.
- Read the API token only from `KUBECOST_API_TOKEN`. Ignore YAML `apiToken`.
- Leave the token out of request logs and allocation error strings.
- Reject filter keys and values that contain `"`, `+`, `\`, whitespace, `(`, or `)`.
- Return `InvalidArgument` for those values and do not call the backend.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `allocation-query`: TLS verification, env-only token, and hostile filter rejection.

## Impact

- `internal/allocation` client, config, and filter encoding.
- `internal/server` maps a hostile filter to `InvalidArgument`.
- `testdata/opencost-real/` stays unchanged.
