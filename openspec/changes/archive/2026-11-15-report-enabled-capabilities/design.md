# Design

## Context

`Info()` in `internal/server/info.go` lists six capabilities statically through `pluginsdk.WithCapabilities`, which replaces the SDK's inferred set. `GetBudgets` already refuses profile `opencost`, so the static list overstates what the plugin serves. The SDK converts any `Unimplemented` from a plugin into its own "plugin does not support GetBudgets", which the conformance suite accepts only when `BUDGETS` is not declared.

## Goals / Non-Goals

**Goals:**
- The reported capabilities match what the active profile serves.
- A shape that lets later flag-dependent capabilities (usage columns, recommendations) add themselves.

**Non-Goals:**
- No new capability, no manifest schema change, no change to `GetBudgets`.

## Decisions

- `Info(cfg allocation.Config)` derives the list from `cfg.ProfileName()`. Alternative: drop `BUDGETS` and rely on the SDK inferring it from a `BudgetsProvider` interface. Rejected: `Server` always implements `GetBudgets`, so inference would still report it.
- An unresolved profile (`ProfileName` error) reports the base five. Startup validation already rejects an unknown profile, so this branch is defensive, and omitting a capability is the safe direction.
- The manifest stays the superset. Core validates the manifest at install time, before any config exists, so it cannot be profile-specific.

## Risks / Trade-offs

- The registry entry in `rshade/finfocus` may still list `budgets`. It needs its own pull request there.
- `GetPluginInfo` now depends on config, so tests must build a client with a profile.
