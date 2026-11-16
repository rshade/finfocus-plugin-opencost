# Proposal

## Why

Profile `kubecost` has never talked to a real backend (issue #71). Its coverage is contract fixtures, and the kind spike (`docs/kubecost-kind-spike.md`) reached no endpoint: `GET /model/allocation` returned HTTP 404. No backend is reachable for a capture. The owner chose option (b): label the profile experimental, so the code, docs, and metadata say what is and is not verified.

## What Changes

- On profile `kubecost`, `GetPluginInfo` metadata carries `kubecost_profile=experimental` and `kubecost_verified=false`. Profile `opencost` sets neither.
- Each budget from `GetBudgets` carries metadata `verified=false`.
- Start-up logs one WARN line on profile `kubecost` saying the profile is experimental. The issue asked for a warning at the first call; the start-up line reaches the same log, follows the existing `tlsSkipVerify` warning, and is emitted exactly once without per-request state.
- The README `kubecost` profile section opens with an experimental banner that links the decision, recorded at the top of `docs/kubecost-kind-spike.md`.
- `EstimateCostResponse` has no notes field, so the estimate itself carries no marker; no response field is invented.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `plugin-info`: experimental metadata on profile `kubecost`.
- `budgets`: each budget says it is unverified.
- `config-loading`: the start-up warning.

## Impact

- `internal/server/info.go`, `internal/server/budgets.go`, `internal/allocation/client.go`, `cmd/finfocus-plugin-opencost/main.go`.
- README, `docs/kubecost-kind-spike.md`.
- Not delivered here: a live capture (option a), and confirming the prediction request body against a live backend. #71 stays open for those.
