# Tasks

## 1. Charge period and coverage warning

- [x] 1.1 Actual results carry the row span as the FOCUS charge period. Verify: `go test -count=1 ./internal/server/ -run TestActualCostChargePeriodIsTheRowSpan` fails before, passes after.
- [x] 1.2 A short cluster-wide span logs the warning on `GetActualCost` and actual `BatchCost`. Verify: `go test -count=1 ./internal/server/ -run 'StoreCoversPartOfWindow'`. Break check: return the object span instead of reading the store and the `object created`, `store span unknown`, and `store query fails` subtests fail.
- [x] 1.3 Tolerance and the query-time clamp. Verify: `go test -count=1 ./internal/server/ -run TestActualCostDoesNotWarnWithoutAStoreGap`. Break check: set the tolerance to zero and drop the clamp, and the test fails.

## 2. Docs and close out

- [x] 2.1 README API notes and CLAUDE.md. Verify: `npx markdownlint-cli2` on the changed files exits 0.
- [x] 2.2 Full gates: `make build test vet lint`, `mise exec -- openspec validate --all --strict`, then archive.
