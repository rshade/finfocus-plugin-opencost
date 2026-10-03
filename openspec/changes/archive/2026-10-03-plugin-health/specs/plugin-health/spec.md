## ADDED Requirements

### Requirement: HealthCheck follows the backend

HealthCheck SHALL send `GET /healthz` to the configured allocation base URL.
A status below 400 SHALL be healthy.
A status of 400 or higher SHALL fail, and the error SHALL include that status code.
The probe SHALL NOT use the allocation cache and SHALL NOT use the allocation rate limit.
When the context has an SDK trace id, the probe SHALL send that id on the request.

#### Scenario: Backend returns 200

- **WHEN** the backend health path returns 200
- **THEN** the health check succeeds

#### Scenario: Backend returns 500

- **WHEN** the backend health path returns 500 after an earlier 200
- **THEN** the health check fails
- **AND** the failure includes 500
- **AND** the backend received both calls

#### Scenario: Rate limit does not hide the backend

- **WHEN** the allocation rate burst is 1 and the backend returns 200 then 500
- **THEN** both health checks reach the backend
- **AND** the second check fails

### Requirement: RPC logs report the trace, the count, and the latency

Each health check and each cost-source RPC SHALL log the SDK trace id, a running request count, and the call latency.
The count SHALL increase for a health check and for `Supports`.

#### Scenario: Health then Supports

- **WHEN** a health check runs and then `Supports` runs with the same trace id
- **THEN** the health log contains that trace id, a request count, and a latency
- **AND** a later health check reports a higher count
