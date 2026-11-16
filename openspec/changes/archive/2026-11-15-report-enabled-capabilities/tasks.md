# Tasks

## 1. Capabilities follow the profile

- [x] 1.1 Write `TestGetPluginInfoCapabilitiesFollowProfile` (table: `opencost`, empty, `kubecost`) and watch it fail; verify the break check by making `Info` always add `BUDGETS`
- [x] 1.2 Change `Info` to take `allocation.Config` and add `BUDGETS` only for profile `kubecost`; verify the new test and `TestGetPluginInfoReportsSpecAndCapabilities` pass
- [x] 1.3 Pass the client config from `GetPluginInfo` and from `main`; verify `go build ./...` and `go test -count=1 ./internal/server/ ./cmd/...` pass

## 2. Docs and spec

- [x] 2.1 State per profile capabilities in the README capability list; verify markdownlint and `readme_test.go` pass
- [x] 2.2 Confirm `plugin.manifest.json` and its `naming_test.go` pin are unchanged and still pass

## 3. Verify

- [x] 3.1 Run `go test ./...`, `go vet ./...`, `make lint`, `make test-race`, `npx markdownlint-cli2 "**/*.md"`, and `mise exec -- openspec validate --all --strict`
- [x] 3.2 Run `OC_E2E=1 go test -tags conformance ./test/conformance` against kind `oc-e2e` and confirm 0 failed
- [x] 3.3 Update the stale conformance note in `CLAUDE.md`
