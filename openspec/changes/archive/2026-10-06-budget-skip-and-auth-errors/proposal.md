# Proposal

## Why

Two error-handling defects in the Kubecost profile. One malformed budget
rule fails `GetBudgets` for every rule, and a backend HTTP 401 or 403
reaches the caller as `Unavailable`, which reads as an outage instead of
a credential problem.

## What Changes

- A budget rule that fails validation (an unsupported `interval`, a
  non-positive `spendLimit`) is skipped and the remaining rules are
  returned. Each skipped rule logs one WARN naming the rule id and the
  reason. The first returned budget carries `skippedRules` and
  `skippedRuleReasons` in `metadata`; when no rule survives, the list is
  empty and only the log records the skips. The summary counts only
  returned budgets.
- **BREAKING** for callers that relied on `Unavailable`: a backend HTTP
  401 is `Unauthenticated` and a 403 is `PermissionDenied`, on every
  backend call (allocation, budgets, prediction, health). The message
  names the `KUBECOST_API_TOKEN` check and never contains the token.
  Other 4xx and 5xx stay `Unavailable`.
- The allocation client returns a typed error carrying the HTTP status
  so the mapper can tell 401 from 500. Message text is unchanged.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `budgets`: invalid rules are skipped with a WARN instead of failing
  the whole call.
- `cost-errors`: backend 401 and 403 map to `Unauthenticated` and
  `PermissionDenied`.

## Impact

- `internal/allocation`: typed status error on the budget, allocation,
  prediction, and health client paths.
- `internal/server`: `mapBackendError` auth branch, `namespaceBudgets`
  skip logic, health check error mapping.
- `README.md`: troubleshooting lines for 401 and 403.
- Kubecost behaviour stays contract-fixture only.
