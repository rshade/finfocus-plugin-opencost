# Proposal

## Why

OpenCost reports each controller's request and usage averages over a window, and #136 now decodes them. A controller that requests far more than it uses is a cost a team can cut by lowering its requests. Issue #138 asks for a right-sizing `GetRecommendations` behind a flag. The owner chose a first slice: profile `opencost` only, with the headroom above usage as a required config value, because no default headroom has a documented source.

## What Changes

- New config keys `recommendations` (bool, default `false`) and `recommendationHeadroom` (fraction above usage, no default), with env overrides `OPENCOST_RECOMMENDATIONS` and `OPENCOST_RECOMMENDATION_HEADROOM`. Start-up validation rejects `recommendations` without a headroom above zero, and with profile `kubecost`.
- `GetPluginInfo` reports `RECOMMENDATIONS` only when `recommendations` is on. `plugin.manifest.json` lists it in the superset.
- `GetRecommendations` (stateless) queries the 30-day allocation by namespace and controller. For each controller whose CPU or memory request average is above usage × (1 + headroom), it returns one `ADJUST_REQUESTS` recommendation. Savings use the same row's observed hourly CPU and memory rates.
- The request honours `excluded_recommendation_ids`, `filter`, `target_resources` (controllers only), `projection_period`, and paging. With the flag off, the RPC returns the SDK's empty answer.
- No `DismissRecommendation` and no `GetRecommendationHistory`: both need persisted state.

## Capabilities

### New Capabilities

- `recommendations`: stateless right-sizing recommendations from OpenCost request and usage averages.

### Modified Capabilities

- `plugin-info`: `RECOMMENDATIONS` joins the capability set when enabled, and joins the manifest superset.
- `config-loading`: the two new keys, their env overrides, and their start-up validation.

## Impact

- New `internal/server/recommendations.go`; `internal/server/info.go`; `internal/allocation/config.go`; `internal/allocation/wirefields.go` (`cpuCoreHours` and `ramByteHours` become consumed, for the per-unit rate).
- README, `config.example.yaml`, `plugin.manifest.json`, `naming_test.go`, `plugininfo_test.go`.
- Profile `kubecost` (`/model/savings/requestSizingV2`) is not built; it stays open on #138.
