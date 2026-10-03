## Context

`pluginsdk.HealthChecker` is `Check(ctx) error`. `HealthHandler` returns HTTP 200
when Check is nil and HTTP 503 when Check returns an error. The default serve
path is gRPC. That path does not call Check, and the web health endpoint is off.

## Goals / Non-Goals

**Goals:**

- Check sends GET /healthz to the allocation base URL.
- A status below 400 is healthy. A status of 400 or higher is an error that includes the status code.
- The probe does not read the allocation cache and does not take a rate-limit token.
- A context trace id is sent on the outbound request as the SDK trace metadata key.
- Health and cost RPCs log that trace id, a running request count, and the latency.
- The count includes HealthCheck and Supports.

**Non-Goals:**

- Prometheus `/metrics`, Go runtime metrics, OpenTelemetry, Jaeger, alerting, or SLI/SLO.
- Switching the plugin process from gRPC to the SDK web server.
- Closing issue #11.

## Decisions

- `Server.Check` calls `Client.Probe`. Probe uses the shared HTTP client and the same profile authentication as an allocation request.
- Probe does not call `GetDetailedAllocation`, so a cached 200 cannot hide a later 500 and a busy limiter cannot fail the check.
- The request count is an atomic counter on the server. It is a zerolog field, not a Prometheus metric.
- zerolog `Info` is a pointer method. The logger is stored in a local before the call.
- Each cost RPC is wrapped so the counter advances on success and on error without a named result.

## Risks / Trade-offs

- A probe of the plugin's gRPC port does not run this check. A host that wants HTTP health uses the SDK handler. This task does not turn that handler on in main.
- The count is process-local and resets on restart.
