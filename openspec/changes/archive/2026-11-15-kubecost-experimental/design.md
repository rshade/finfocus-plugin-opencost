# Design

## Context

`Info(cfg)` builds the plugin record with `pluginsdk.NewPluginInfo`; the SDK has `WithMetadata`. `GetBudgets` already writes rule metadata (`namespaces`, window, skipped rules). `main` calls `cfg.WarnIfTLSSkipped(logger)` after validation.

## Goals / Non-Goals

**Goals:** say "experimental, not verified" wherever a `kubecost` result or record can carry it, without inventing response fields.

**Non-Goals:** no change to what the profile sends or returns, no live capture, no spec change.

## Decisions

1. **Metadata keys** `kubecost_profile=experimental` and `kubecost_verified=false` come from the issue text, so a host can read them without parsing prose. They are added only on profile `kubecost`, so an `opencost` record is unchanged.
2. **`verified=false` on every budget**, next to the existing rule metadata. Budgets are the only `kubecost` result with a metadata map.
3. **Warning at start-up, not first call.** A `Config.WarnIfExperimental(logger)` method next to `WarnIfTLSSkipped` gives one line per process with no `sync.Once` in the server, and the binary-level start-up test pattern already exists.
4. **The decision lives in the spike document's header**, which the README banner links. The spike results stay below it unchanged.

## Risks / Trade-offs

- [A host ignores metadata] → the README banner and the WARN line carry the same statement.
