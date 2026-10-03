# Proposal

## Why

Every cost RPC called the allocation API again, even for the same window.
The HTTP client did not keep a connection pool or a request limit.

## What Changes

- Cache a successful allocation body by its URL for a short TTL.
- Set `expires_at` on actual, projected, and estimate results to that deadline.
- Pool connections and set dial, TLS handshake, idle, and header timeouts.
- Limit outbound allocation requests without a new module.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `allocation-query`: cached windows, `expires_at`, rate limit, and pooled timeouts.

## Impact

- `internal/allocation` client, cache, and limiter.
- `internal/server` copies the cache deadline onto cost results.
- `testdata/opencost-real/` stays unchanged.
- Issue #12 stays open. LRU, metrics, and incoming gRPC limits are not this change.
