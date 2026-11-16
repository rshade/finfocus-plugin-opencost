# Tasks

## 1. DryRun handler

- [x] 1.1 Write `TestDryRunRejectsMissingResource`, `TestDryRunMapsFocusFieldsForEverySupportedType`, `TestDryRunUnsupportedTypeIsAResponse`, `TestDryRunReportsInvalidConfiguration`, `TestDryRunMakesNoBackendCall`, and `TestDryRunThroughSDKServer` in `internal/server/dryrun_test.go` and watch them fail to compile, then add `internal/server/dryrun.go` (`HandleDryRun`). Verify: `go test -count=1 ./internal/server/ -run DryRun` exits 0. Break check: make `HandleDryRun` return `codes.Unimplemented` and `TestDryRunThroughSDKServer` fails; set `billing_currency` to `UNSUPPORTED` and `TestDryRunMapsFocusFieldsForEverySupportedType` fails.
- [ ] 1.2 Add `DRY_RUN` to `Info`, `plugin.manifest.json`, the pins in `plugininfo_test.go` and `naming_test.go`, and the README capability list with one `DryRun` paragraph. Verify: `go test -count=1 ./internal/server/ -run 'PluginInfo|Kubecost'` and `npx markdownlint-cli2 README.md` exit 0. Break check: drop `DRY_RUN` from `Info` and `TestGetPluginInfoCapabilitiesFollowProfile` fails.

## 2. Kube client keeps the operator config

- [ ] 2.1 Write `TestKubeClientKeepsOperatorConfig` in `cmd/finfocus-plugin-opencost` (client built by `newClient` against a `kubetest` API server: `GetConfig().BaseURL` is empty, `GetConfig().Validate()` is nil, `DryRun` reports a valid configuration, and an allocation request still reaches the API server proxy path), then add `allocation.WithBaseURL` and the client-held base URL, and pass the origin from `newClient`. Verify: `go test -count=1 ./cmd/... ./internal/allocation/ ./internal/kubeproxy/...` exits 0. Break check: restore `cfg.BaseURL = conn.BaseURL` in `newClient` and the test fails.

## 3. Integration

- [ ] 3.1 Run `go test ./...`, `go vet ./...`, `make lint`, `make test-race`, `npx markdownlint-cli2 "**/*.md"`, and `mise exec -- openspec validate --all --strict`; each exits 0.
- [ ] 3.2 Build the plugin, install it with `make install`, and run `finfocus plugin inspect opencost kubernetes:apps/v1:Deployment`; it exits 0 and prints the field mappings. Run it for `kubernetes:core/v1:Service`; it exits 0 with no mappings. Break check: with the handler disabled the command prints "capability discovery not implemented".
- [ ] 3.3 Run `make e2e-kind` (the allocation client changed) and `OC_E2E=1 go test -tags conformance ./test/conformance`; both report 0 failed.
- [ ] 3.4 Update `CLAUDE.md` with the `DryRun` behaviour and the `WithBaseURL` wiring.

## Workflow follow-up

- Archive the change in the same commit as the last code task, after verify passes.
- After release, update the `opencost` capability list in the `rshade/finfocus` registry entry.
