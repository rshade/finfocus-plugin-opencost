# Tasks

## 1. Config

- [x] 1.1 Write `TestIncludeUsageConfig` (default false, YAML true, env overrides file both ways, invalid env names the variable), then add `Config.IncludeUsage` and the `OPENCOST_INCLUDE_USAGE` override; document the key in the README configuration reference and `config.example.yaml`. Verify: `go test -count=1 ./internal/allocation/ -run 'IncludeUsage|Readme'` exits 0. Break check: drop the env override and the env cases fail.

## 2. Decode

- [x] 2.1 Decode the eight keys into typed `Entry` fields and move them from `IgnoredFields` to `ConsumedFields`. Verify: `go test -count=1 ./internal/allocation/ -run WireFields` and `make drift` exit 0. Break check: leave one key in both lists and `TestWireFieldsCoverRecordedKeys` fails.

## 3. Columns

- [x] 3.1 Write `TestActualCostUsageColumnsOffByDefault`, `TestActualCostUsageColumnsFromRecordedRow`, and `TestUsageColumnsSkipIdleAndNeedAGPU` against `testdata/opencost-real/allocation-namespace-60m.json`, then pass the flag into `resultsFor` and add the columns. Document them in the README. Verify: `go test -count=1 ./internal/server/ -run 'UsageColumns'` exits 0. Break check: default the flag to true in `resultsFor` callers and the off-by-default test fails.

## 4. Integration

- [x] 4.1 Run `go test ./...`, `go vet ./...`, `make lint`, `make test-race`, `make drift`, `npx markdownlint-cli2 "**/*.md"`, and `mise exec -- openspec validate --all --strict`; each exits 0.

## Workflow follow-up

- Archive the change in the same commit as the last code task.
