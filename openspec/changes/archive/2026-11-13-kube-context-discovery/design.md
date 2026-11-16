# Design

## Context

See proposal.md for motivation. Current state:

- `allocation.Config.BaseURL` is the only address. `httpClient(cfg, roots)`
  in `internal/allocation/client.go` builds the one `http.Transport`, and
  `Probe` in `health.go` and the allocation URL builder both start from
  `cfg.BaseURL`.
- `Config.Validate` requires `baseUrl` to be `https` with a host or loopback
  `http`. `main.go` runs load, validate, then `NewClient`.
- The cache key is the request URL, the rate limit sits in front of the
  transport, and profile `opencost` sends no `Authorization` header.

## Goals / Non-Goals

**Goals:**

- One place resolves a kube target into a base URL and an authenticated
  `http.RoundTripper`. Nothing downstream learns that kube exists.
- A missing context fails at startup. A missing grant or service fails with
  its own error on the first request.

**Non-Goals:**

- Discovering the OpenCost service by label or by scanning namespaces. The
  caller names it, and defaults match the OpenCost Helm chart.
- Watching or refreshing the kubeconfig while running. Credentials load at
  startup. An exec plugin still refreshes its own token through client-go.
- Supporting a custom `Host` or path rewrite for OpenCost behind an ingress.

## Decisions

### 1. Use the API server service proxy

Requests go to `<apiserver>/api/v1/namespaces/<ns>/services/<svc>:<port>/proxy/...`.
The plugin sends plain HTTPS with kubeconfig credentials, so there is no
process to start, no local port to clash on, and nothing to clean up when a
CI job ends. RBAC is one grant: `get` on `services/proxy` for one service.

Alternatives:

- **Spawn `kubectl port-forward`.** Needs `kubectl` on the runner, a free
  port, and a child process to supervise and kill. Rejected by the user and
  by the proposal's out-of-scope list.
- **client-go SPDY port-forward in process.** Still a local listener and a
  goroutine lifecycle, and it needs `pods/portforward`, a broader grant than
  `services/proxy`. Rejected.
- **Ingress plus `baseUrl`.** Already works and stays supported. It does not
  help a CI runner with only a kubeconfig, and profile `opencost` sends no
  token for an authenticated ingress.

### 2. A narrow seam in `NewClient`, resolved in `main`

New package `internal/kubeproxy` exposes
`Resolve(Target) (baseURL string, rt http.RoundTripper, err error)`. `main`
calls it only when `cfg.Kube.Service` is set, copies the result into
`cfg.BaseURL`, and passes the transport to `allocation.NewClient` as an
option. The allocation package never imports client-go, so its tests and
the URL builder stay as they are.

The allocation package sets `u.Path` absolutely at five call sites
(`allocation.go`, `health.go`, `budget.go`, and `PredictSpecCost`), so a base
URL that carries the proxy path would lose it. `Resolve` therefore returns the
API server origin as the base URL, and the transport it returns prepends the
proxy path to every request. `PredictSpecCost`, `Probe`, and the URL builder
need no change, and the cache key stays the pre-rewrite URL. The same
transport turns an API server failure into a gRPC status error (decision 4).
`internal/server` changes by one line: `mapBackendError` returns the clean
status from a wrapped gRPC error instead of the wrapped text.

Alternative: put `if cfg.Kube.Service != ""` inside `httpClient`. Rejected,
because it couples the allocation package to client-go and mixes two
connection models in one function.

### 3. Depend on client-go's `clientcmd` and `rest` only

`clientcmd` loads the kubeconfig and honours `KUBECONFIG` and the default
path. `rest.InClusterConfig` covers the pod. `rest.TransportFor` returns a
transport that already handles certificates, tokens, and exec plugins. The
plugin does not use the typed clientset, which keeps the module graph
smaller. A hand-written kubeconfig parser cannot run exec plugins, and the
user needs both exec-plugin and static credentials.

### 4. Tell API server errors from OpenCost errors by the body

The API server answers its own failures with a `metav1.Status` JSON body
(`kind: Status`). The proxy passes OpenCost's own errors through unchanged,
including the plain-text HTTP 400 bodies recorded under
`testdata/opencost-real/`. The mapping, applied only in kube mode:

| Response | gRPC code |
|---|---|
| `401` | `Unauthenticated`, names the context |
| `403` | `PermissionDenied`, names `services/proxy`, namespace, service |
| `404`, `503` or `504` with a `Status` body | `Unavailable`, names context, namespace, service |
| connection error to the API server | `Unavailable`, names the context |
| any other status | existing mapping in `statuserror.go` |

A `404` without a `Status` body came from OpenCost and keeps its current
meaning. The implementation task reads `statuserror.go` first and reuses its
helpers rather than adding a parallel mapping.

### 5. Validation is mutually exclusive and strict

`baseUrl`, `tlsSkipVerify`, `caCertFile`, and `allowInsecureHttp` are errors
next to a kube target, because the kubeconfig owns that connection's
address and trust and a silently ignored security setting is a hidden
failure. Profile `kubecost` is rejected for the bearer-token collision.

### 6. Environment names

`OPENCOST_KUBE_CONTEXT`, `OPENCOST_KUBE_NAMESPACE`, `OPENCOST_KUBE_SERVICE`,
and `OPENCOST_KUBE_PORT`, matching `OPENCOST_PROFILE` and `OPENCOST_CURRENCY`.
CI sets these without writing a file. The kubeconfig path itself uses the
standard `KUBECONFIG` so existing runner setup keeps working. These names
are my call and can change before apply.

## Risks / Trade-offs

- **The proxy may not accept the allocation query** (filter encoding, long
  windows) → a fake server cannot prove it, so `make e2e-kind` runs the
  oracle through the real proxy. This is the only check that closes it.
- **Binary and dependency growth** (`client-go`, `apimachinery`) → record the
  size before and after, run `govulncheck`, re-tidy `test/drift`. Use only
  `clientcmd` and `rest`.
- **Exec plugins need their CLI on the runner** (`aws`,
  `gke-gcloud-auth-plugin`) → the error names the command and never its
  output, which can carry a token. The production guide lists the
  prerequisite.
- **Gateway timeouts** → the API server or a fronting proxy can cut a long
  30-day query before `timeout`. A `504` is `Unavailable`. The guide says so.
- **Wider blast radius for a leaked kubeconfig** than a bare OpenCost URL →
  the guide recommends a dedicated service account with only
  `get services/proxy` on the one service.
- **Cluster API server load** → requests are cached for `cacheTTL` and
  limited by `requestsPerSecond`, the same as today.

## Migration Plan

Additive. `baseUrl` deployments are untouched, and rollback is removing the
`kube` block. No data migration.
