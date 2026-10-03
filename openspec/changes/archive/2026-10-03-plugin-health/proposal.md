# Proposal

## Why

The plugin can answer cost RPCs while the allocation backend is down.
Callers have no health signal, and request logs do not show a count, a latency, or the host trace id.

## What Changes

- Probe the backend health path and fail when that path returns an error status.
- Log a running request count, the call latency, and the SDK trace id.
- Copy that trace id onto the outbound request.
- Leave the allocation cache and the outbound rate limit out of the probe.

## Capabilities

### New Capabilities

- `plugin-health`: backend health and the request log fields.

### Modified Capabilities

- None.

## Impact

- `internal/allocation` probes `/healthz` without the cache or the rate limiter.
- `internal/server` implements the SDK health checker and counts cost RPCs.
- The process stays on gRPC and does not mount its own HTTP health endpoint.
- Issue #11 stays open. Prometheus, OpenTelemetry, and alerting are not this change.
