# Tasks

## 1. Config surface

- [x] 1.1 Add a `Kube` struct (`context`, `namespace`, `service`, `port`, `kubeconfig`) to `allocation.Config` with YAML keys; verify `TestConfigDefaults` still passes and a YAML file with a `kube` block loads
- [x] 1.2 Load `OPENCOST_KUBE_CONTEXT`, `_NAMESPACE`, `_SERVICE`, `_PORT` over the file; verify `TestKubeEnvOverrides` covers precedence and a non-integer port naming `OPENCOST_KUBE_PORT`
- [x] 1.3 Extend `Config.Validate`: exactly one of `baseUrl` or `kube.service`, reject `tlsSkipVerify`, `caCertFile`, `allowInsecureHttp` next to a kube target, require profile `opencost`, bound `kube.port`; verify `TestKubeConfigValidation` and the existing `TestConfigValidate` pass
- [x] 1.4 List every new key in `config.example.yaml` and the README configuration reference; verify `readme_test.go` (every `allocation.Config` key is listed) passes

## 2. Kube resolver

- [x] 2.1 Add `k8s.io/client-go` and run `go mod tidy` in the plugin module and in `test/drift`; record binary size before and after in the task notes and verify `go build ./...` and `govulncheck ./...` are clean (binary 28,411,205 bytes before, 34,669,538 after, +6.3 MB; govulncheck reports no called vulnerability, and two `golang.org/x/net` findings fixed in v0.62.0 that the code does not call)
- [x] 2.2 Create `internal/kubeproxy` with `Resolve(Target)` returning the proxy base URL and an authenticated `http.RoundTripper`, using only `clientcmd` and `rest`; verify `TestKubeTargetBuildsServiceProxyURL` and `TestKubeDefaultsNamespaceAndPort` pass
- [x] 2.3 Implement credential selection (`kube.kubeconfig`, `KUBECONFIG`, default path, then in-cluster) and context selection; verify `TestKubeCredentialSources` covers each source and the two-context case
- [x] 2.4 Verify an exec-plugin user authenticates against a fake API server using a stub script; `TestKubeExecPluginCredentials` passes and the failure message names the command, never its output
- [x] 2.5 Fail with a one-line reason for an unknown context or no credentials at all; verify `TestKubeMissingContextNamesContext` passes
- [x] 2.6 Add a break check for each test in this group (a one-line change that makes it fail) and note the result

## 3. Client wiring and errors

- [x] 3.1 Read `internal/allocation/statuserror.go` and add an `allocation.NewClient` option that accepts a transport; verify existing allocation tests pass unchanged
- [x] 3.2 Wire `main.go`: resolve the kube target after validation, copy the base URL into the config, log one line with context, namespace, service, and port; verify `TestKubeCredentialsNeverLogged` and `TestBinaryExitsOnInvalidConfig` pass
- [x] 3.3 Map API server `403`, `404` and `503` `Status` bodies and connection errors to `PermissionDenied` and `Unavailable` in kube mode only, and leave OpenCost's own `404` and `400` mapping alone; verify `TestKubeForbiddenIsPermissionDenied` and `TestKubeUnreachableServiceIsUnavailable` pass
- [x] 3.4 Confirm `Probe` goes through the proxy path and reports the same reasons; verify `TestKubeHealthProbeUsesProxy` passes
- [x] 3.5 Verify the cache key holds no credential and the rate limit still applies in kube mode with a test assertion, and add a break check for each test in this group

## 4. Kind end to end

- [x] 4.1 Add `TestE2EActualCostThroughServiceProxy` to `test/e2e`: start the plugin with the `kube` block for `kind-oc-e2e`, no `baseUrl`, and assert the oracle comparison and `Check`; verify it passes under `make e2e-kind`
- [x] 4.2 Add the wrong-service-name case and verify it returns `Unavailable` naming the service
- [x] 4.3 Run the new test with the port-forward off to prove it does not depend on one, then break the namespace in the config and verify the test fails

## 5. Documentation

- [x] 5.1 Write `docs/production.md`: the RBAC `Role` and `RoleBinding` for `get services/proxy` on one service, plus setup for a CI runner, a workstation, and a pod running `finfocus --web`, including the exec-plugin CLI prerequisite and the gateway timeout note; verify `npx markdownlint-cli2 "**/*.md"` is clean
- [x] 5.2 State in the README that a port-forward is a development path and link the production guide; verify `readme_test.go` and markdownlint pass
- [x] 5.3 Update `CLAUDE.md` for the new keys, the seam in `main`, and the error mapping; verify no machine-specific absolute path appears in any new doc

## 6. Integration

- [x] 6.1 Run the exact CI commands: `go test ./...`, `go vet ./...`, `make lint`, `npx markdownlint-cli2 "**/*.md"`, `mise exec -- openspec validate --all --strict`, and `make e2e-kind`; verify all exit 0

## Workflow follow-up

- Archive the change in the same commit as the code once the owner approves.
- Do not edit `CHANGELOG.md`. Release Please generates it.
