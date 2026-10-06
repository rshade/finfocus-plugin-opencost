# Design

## Context

`internal/server/budgets.go` `namespaceBudgets` returns the first error
from `budgetFromRule`, so one bad rule fails the whole call. The
allocation clients keep the backend HTTP status only inside an error
string, so `mapBackendError` cannot tell 401 from 500 and everything
falls through to `Unavailable`.

## Goals / Non-Goals

Goals: typed status error on every client path; 401/403 mapped to
`Unauthenticated`/`PermissionDenied` on every backend call; invalid
budget rules skipped with one WARN each.

Non-goals: new proto fields or a `warnings` field on
`GetBudgetsResponse`, retry or token refresh, changes to rate-limit or
timeout mapping, live Kubecost verification.

## Decisions

- **`StatusError{Code, Msg}` in `internal/allocation`, with
  `HTTPStatus(err) (int, bool)` using `errors.As`.** The message text
  of every existing error is preserved; only the type changes. A
  helper beats exported struct assembly at each call site.
- **Skip visibility goes in `metadata` of the first returned budget.**
  The issue's open question offered metadata or logs-only. The issue's
  acceptance criteria require the skip to be visible to the caller, and
  the response has no warnings field (inventing one is a non-goal).
  When every rule is skipped, the list is empty and only the log
  records it. Cost if wrong: metadata on one budget is easy to miss —
  the owner can ask for logs-only in review and the change is small.
- **Health errors also pass through `mapBackendError`.** Today `Check`
  returns the probe error raw. The plugin-health spec still holds: a
  401 failure includes `401` (in the `HTTP 401` message), and a 500
  failure still includes `500` under the `Unavailable` wrap.

## Risks / Trade-offs

- [A caller that treated `Unavailable` as retryable now sees
  `Unauthenticated`] → That is the point of the issue; credentials
  failures are not transient.
- [WARN per skipped rule could spam on a fully broken list] → One line
  per rule per call, same order as the rule count; no aggregation
  needed at this scale.

## Migration Plan

None. Valid budget lists and healthy backends behave exactly as before.
