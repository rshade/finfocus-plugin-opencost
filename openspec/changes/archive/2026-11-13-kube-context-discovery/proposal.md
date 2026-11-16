# Proposal

## Why

The plugin only reaches OpenCost through `baseUrl`. A ClusterIP service is
not reachable from where FinFocus core runs, so the only documented route
is a hand-run `kubectl port-forward`. That fits the kind suite and a
developer laptop. It does not fit a CI runner, which is short-lived and
has no process to keep a forward alive, or a team that wants one config
that works from a workstation, a CI job, and a pod.

Every one of those environments already holds a kubeconfig or an
in-cluster service account. The plugin can use it.

## What Changes

- Add a `kube` config block (`context`, `namespace`, `service`, `port`,
  `kubeconfig`) as an alternative to `baseUrl`. When it is set, the plugin
  reaches OpenCost through the Kubernetes API server service proxy:
  `/api/v1/namespaces/<ns>/services/<name>:<port>/proxy/`.
- Authenticate with `k8s.io/client-go`, so a static certificate, a static
  token, and exec plugins (`aws eks get-token`, `gke-gcloud-auth-plugin`)
  all work. With no kubeconfig and no `context`, use in-cluster config.
- Startup validation requires exactly one of `baseUrl` and `kube.service`.
  Kube mode requires profile `opencost`, because the profile `kubecost`
  bearer token would collide with the kubeconfig credentials.
- Report a missing context, a missing RBAC grant (`get` on
  `services/proxy`), and an unreachable service as distinct, one-line
  errors that name the context, namespace, and service.
- Add a `make e2e-kind` path that prices `namespace/oc-test` through the
  real proxy on `oc-e2e`.
- Add a production guide (RBAC, CI, workstation, pod). The README states
  that a port-forward is a development path only.
- `baseUrl` behaviour is unchanged. Nothing here is **BREAKING**.

Out of scope: the plugin starting or managing `kubectl port-forward`,
authentication for an ingress (profile `opencost` sends no token), and
gRPC authentication (an existing non-goal).

## Capabilities

### New Capabilities

- `kube-service-proxy`: resolve the OpenCost service from a kube context or
  in-cluster config and send allocation and health requests through the API
  server service proxy, with distinct errors for each failure.

### Modified Capabilities

- `config-loading`: startup validation accepts a `kube` block as an
  alternative to `baseUrl`, rejects both or neither, and rejects kube mode
  with profile `kubecost`.
- `kind-suite`: add a requirement that the kind suite prices
  `namespace/oc-test` through the service proxy.

## Impact

- Code: new `internal/kubeproxy` package, a `Kube` field on
  `allocation.Config`, a transport seam in `allocation.NewClient`, and
  wiring in `cmd/finfocus-plugin-opencost/main.go`. The allocation URL
  building is untouched because the proxy path becomes the base URL.
- Dependency: `k8s.io/client-go` and its transitive modules. `test/drift`
  must be re-tidied. `govulncheck` in the Test workflow covers the new
  modules.
- Docs: README configuration reference, `config.example.yaml`, a new
  production guide under `docs/`, and `CLAUDE.md`.
- Tests: a fake API server for unit tests, plus `test/e2e`. A mock cannot
  prove the real proxy accepts the allocation query, so the e2e path is
  required.
