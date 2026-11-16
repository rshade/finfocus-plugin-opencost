# Tasks

## 1. Labels

- [x] 1.1 Write `TestGetPluginInfoLabelsKubecostExperimental`, then add the metadata in `Info` on profile `kubecost`. Verify: `go test -count=1 ./internal/server/ -run PluginInfo` exits 0. Break check: drop the metadata and the test fails.
- [x] 1.2 Write `TestBudgetsCarryVerifiedFalse`, then set `verified=false` on each budget. Verify: `go test -count=1 ./internal/server/ -run Budgets` exits 0. Break check: drop the key and the test fails.
- [x] 1.3 Write `TestKubecostProfileWarnsAtStartup`, then add `Config.WarnIfExperimental` and call it from `main`. Verify: `go test -count=1 ./internal/allocation/ -run Kubecost` exits 0. Break check: drop the call in `main` and the test fails.

## 2. Docs

- [x] 2.1 Record the decision at the top of `docs/kubecost-kind-spike.md`, add the experimental banner to the README `kubecost` profile section with a link to it, and update CLAUDE.md. Verify: `npx markdownlint-cli2 "**/*.md"` and `go test ./internal/server/ -run Docs` exit 0.

## 3. Integration

- [x] 3.1 Run `go test ./...`, `go vet ./...`, `make lint`, `make test-race`, and `mise exec -- openspec validate --all --strict`; each exits 0.
