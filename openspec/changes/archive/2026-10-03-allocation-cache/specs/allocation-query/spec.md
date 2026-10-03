## ADDED Requirements

### Requirement: Repeated allocation windows are cached

The client SHALL cache a successful allocation response by its request URL.
A second identical query inside the TTL SHALL NOT call the backend.
The default TTL SHALL be 30 seconds. `cacheTTL` MAY override it.
A negative `cacheTTL` SHALL disable the cache.
The cache SHALL NOT store the API token.
A failed response SHALL NOT be cached.

#### Scenario: Two identical calls

- **WHEN** `GetActualCost` is called twice for the same resource and window inside the TTL
- **THEN** the backend receives one request
- **AND** both results have the same cost

#### Scenario: Expired cache

- **WHEN** the same query runs again after the TTL
- **THEN** the backend receives another request

### Requirement: Cost results advertise expires_at

A successful actual, projected, or estimate result SHALL set `expires_at` to the cache deadline.
That time SHALL be in the future.
A cache hit SHALL return the same deadline.

#### Scenario: Actual cost hint

- **WHEN** `GetActualCost` returns a row
- **THEN** `expires_at` is set
- **AND** `expires_at` is after the call

### Requirement: Outbound allocation requests are rate limited

The client SHALL limit outbound allocation requests to 10 per second with a burst of 20 unless configured otherwise.
`requestsPerSecond` and `rateBurst` MAY override those defaults.
A request over the limit SHALL NOT call the backend.
The server SHALL return `ResourceExhausted` with message `allocation request rate limit exceeded`.
A cache hit SHALL NOT consume a token.

#### Scenario: Burst of one

- **WHEN** two different allocation queries run at once and `rateBurst` is 1
- **THEN** the second status is `ResourceExhausted`
- **AND** the backend receives one request

### Requirement: The HTTP client pools connections

The transport SHALL keep idle connections and set an idle timeout, a dial timeout, a TLS handshake timeout, and a response header timeout.
TLS verification SHALL follow `tlsSkipVerify`.

#### Scenario: Reused connection

- **WHEN** two uncached allocation requests go to the same host
- **THEN** they use one connection
