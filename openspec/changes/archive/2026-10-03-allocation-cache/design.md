## Context

Cost RPCs call `GetDetailedAllocation`. That method builds one URL per window
and filter. The transport created in OC-4.1 verified TLS and did not set pool
timeouts. `expires_at` on cost results was unset.

## Goals / Non-Goals

**Goals:**

- Two identical queries inside the TTL make one backend request.
- The cache key is the allocation URL. The API token is not part of the key and is not stored.
- Failed responses are not cached.
- `expires_at` is the cache deadline, in the future, and the same on a cache hit.
- A negative `cacheTTL` disables the cache.
- Outbound requests default to 10 per second with a burst of 20.
- Over the limit, the backend is not called and the status is `ResourceExhausted`.
- The transport keeps idle connections and sets dial, handshake, idle, and header timeouts.
- TLS verification stays on unless `tlsSkipVerify` is set.

**Non-Goals:**

- LRU eviction, cache metrics, circuit breakers, or incoming gRPC rate limits.
- `golang.org/x/time/rate`.
- Closing issue #12.

## Decisions

- Default cache TTL is 30 seconds. `cacheTTL` overrides it. Zero means that default.
- The limiter is a token bucket in this module. `requestsPerSecond` and `rateBurst` override the defaults. Zero means the default.
- Cache hits do not consume a token.
- Header timeout uses the client timeout when that timeout is positive, otherwise 15 seconds.
- The pool allows 10 idle connections per host.

## Risks / Trade-offs

- A cost can be up to 30 seconds old. `expires_at` tells the host when to refetch.
- A burst above 20 waits for the limiter instead of calling the backend.
