# Tasks

## 1. Typed backend status error

- [x] 1.1 Add `StatusError` and `HTTPStatus` in `internal/allocation`; return it from the budget, allocation, prediction, and health client paths with the message text unchanged. Verify: `go test -count=1 ./internal/allocation/ -run TestBackendStatusErrorIsTyped` fails before, passes after. Break check: return the plain `fmt.Errorf` at one site and its subtest fails.

## 2. Auth mapping

- [x] 2.1 Map HTTP 401 to `Unauthenticated` and 403 to `PermissionDenied` in `mapBackendError`, with a message naming the `KUBECOST_API_TOKEN` check and never the token; route the health check error through the mapper. Verify: `go test -count=1 ./internal/server/ -run 'TestBackendAuthMaps\|TestBackendAuthErrorOmitsToken\|TestHealthCheckFollowsBackend'`. Break check: revert the new branch and the 401 subtests fail with `Unavailable`.

## 3. Skip invalid budget rules

- [x] 3.1 `namespaceBudgets` skips a rule `budgetFromRule` rejects, logs one WARN per skipped rule with id and reason, and records `skippedRules` and `skippedRuleReasons` in the first returned budget's metadata; an all-invalid list returns empty. Verify: `go test -count=1 ./internal/server/ -run TestGetBudgetsSkips`. Break check: restore the early return and a `daily` interval fails the whole call again.

## 4. Docs and close out

- [x] 4.1 README troubleshooting: 401 returns `Unauthenticated`; add the 403 line. Verify: `markdownlint README.md` and `vale README.md` exit 0.
- [x] 4.2 Full gates: `go test -count=1 ./internal/server/ ./internal/allocation/ -run 'GetBudgets\|Auth\|Backend\|Token'`, `make build test lint`, `mise exec -- openspec validate --all --strict`, then archive.
