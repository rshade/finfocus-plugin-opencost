# Proposal

## Why

`GetPluginInfo` reports `BUDGETS` on every configuration, but `GetBudgets` returns `Unimplemented` on profile `opencost` ("budgets require profile kubecost"). Core gates RPCs on the reported list, and the `finfocus-spec` v0.7.12 conformance suite fails a plugin that declares `BUDGETS` and answers `Unimplemented`. `go test -tags conformance` against the kind cluster now reports 10 passed, 1 failed. Issue #133.

## What Changes

- `Info` takes the resolved allocation config and returns the capability list for that profile.
- Profile `opencost` reports `PROJECTED_COSTS`, `ACTUAL_COSTS`, `PRICING_SPEC`, `ESTIMATE_COST`, `BATCH_COST`. Profile `kubecost` adds `BUDGETS`.
- `GetPluginInfo` and `main` pass the client config to `Info`.
- `plugin.manifest.json` keeps the superset the binary can serve. Its pin in `naming_test.go` is unchanged.
- The README capability list states which profile reports which capability.
- `GetBudgets` behaviour is unchanged.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `plugin-info`: the reported capability set follows the profile instead of being fixed.

## Impact

- `internal/server/info.go`, `internal/server/plugininfo_test.go`, `cmd/finfocus-plugin-opencost/main.go`, README.
- A user on profile `opencost` no longer sees `BUDGETS`. That is the intended correction, not a new restriction: the RPC already failed.
- Open question from the issue: the `rshade/finfocus` registry entry for `opencost` may list `budgets`. That is a separate pull request in that repo.
