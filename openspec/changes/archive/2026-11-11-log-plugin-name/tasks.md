# Tasks

## 1. Plugin identity in logs

- [x] 1.1 Startup log lines carry `plugin_name` and `plugin_version` from `pluginsdk.NewPluginLogger`. Verify: `go test -count=1 ./cmd/finfocus-plugin-opencost/ -run TestBinaryExitsOnInvalidConfig` fails before, passes after.

## 2. Docs and close out

- [x] 2.1 README logging note and CLAUDE.md. Verify: `npx markdownlint-cli2` on the changed files exits 0.
- [x] 2.2 Full gates: `make build test vet lint`, `mise exec -- openspec validate --all --strict`, then archive.
