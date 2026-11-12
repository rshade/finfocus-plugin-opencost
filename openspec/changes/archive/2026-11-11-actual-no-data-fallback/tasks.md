# Tasks

## 1. Actual cost no-data response

- [x] 1.1 `GetActualCost` returns no results and `FALLBACK_HINT_RECOMMENDED` when no row matches, before the currency check. Verify: `go test -count=1 ./internal/server/ -run 'TestCostRPCErrorMapping|TestAllocationFilterKeysAreAcceptedFields'` fails before, passes after. Break check: restore `noCostData` and the `no match` subtests fail with `NotFound`.
- [x] 1.2 An actual `BatchCost` item with no row is `ActualCostData` with no results and `FALLBACK_HINT_RECOMMENDED`. Verify: `go test -count=1 ./internal/server/ -run TestBatchActualMissingObjectRecommendsFallback`. Break check: restore the per-item error and the test fails on a non-nil error.
- [x] 1.3 Projected cost keeps `NotFound` with `NO_COST_DATA`. Verify: subtest `projected no match stays NotFound`.

## 2. Docs and close out

- [x] 2.1 README API notes, batch notes, and troubleshooting; CLAUDE.md implementation notes. Verify: `npx markdownlint-cli2` on the changed files exits 0.
- [x] 2.2 Full gates: `make build test vet lint`, `mise exec -- openspec validate --all --strict`, then archive.
