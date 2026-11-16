# Design

## Context

`pluginsdk.Server.DryRun` (finfocus-spec v0.7.12) rejects a request with no resource, then calls the plugin's `HandleDryRun` when the plugin implements `pluginsdk.DryRunHandler`, and returns `Unimplemented` otherwise. `main` serves `server.New(cli)` through `pluginsdk.Serve`, so a method on `*server.Server` is all the RPC needs. Core's `plugin inspect` sends only `resource.resource_type` and renders `field_name`, `support_status`, and `condition_description`.

`costsource.proto` says a `FieldMapping.field_name` must name a `FocusCostRecord` field. azure-public builds its mappings from `pluginsdk.FocusFieldNames()` and sets the subset it fills.

Actual cost is the only RPC here that returns a `FocusCostRecord`. It sets `billing_currency` on every result (`applyCurrency`), `charge_period_start` and `charge_period_end` when the row has a start and end (`stampChargePeriod`), `tags` from the row labels, with `namespaceLabels` as the fallback on a namespace row (`focusFor`), and `extended_columns` from annotations plus `controllerKind` on rows that are not namespaces. The amount is `ActualCostResult.cost`, not `FocusCostRecord.billed_cost`.

`main` validates the config and exits on failure before it serves. In kube mode `newClient` then writes the API server origin into `cfg.BaseURL` and hands that config to `allocation.NewClient`. `validateKube` rejects `baseUrl` next to `kube.service`, so `cli.GetConfig().Validate()` fails on every kube deployment today.

## Goals / Non-Goals

**Goals:**

- `DryRun` describes the existing actual-cost record and prices nothing.
- `cli.GetConfig()` returns the config the operator gave, so validating it again gives the start-up answer.

**Non-Goals:**

- No probe of the backend or of `/healthz` from `DryRun`.
- No use of `simulation_parameters`. The mappings do not depend on the descriptor beyond its type.
- No change to what actual cost returns.

## Decisions

1. **Mappings are FOCUS output fields.** Start from `pluginsdk.AllFieldsWithStatus(UNSUPPORTED)` and set five fields with `pluginsdk.SetFieldStatus`. This follows the proto and azure-public. The alternative from the issue text, mappings for the input keys `id`, `metadata.name`, and `metadata.namespace`, would put names in `field_name` that no `FocusCostRecord` has, and core would print them as FOCUS fields.
2. **`billed_cost` stays `UNSUPPORTED`.** The record does not carry it. Reporting `SUPPORTED` would describe a field core never receives. Moving the amount into the record is a separate behaviour change.
3. **Conditions differ only for namespace rows.** A namespace `tags` condition names the `namespaceLabels` fallback, and its `extended_columns` condition names annotations only. Every other supported type names `labels`, `controllerKind`, and annotations. A node row keeps `controllerKind` because `focusFor` clears it only for a namespace.
4. **Unsupported type: no mappings, configuration still reported.** `resource_type_supported=false` with empty mappings matches azure-public and the SDK's batch fallback. Configuration validity does not depend on the type, so it is reported the same way for both.
5. **Configuration errors reuse `Config.Validate`.** `Validate` returns the first error. `configuration_errors` holds that one message. It reads `caCertFile` from disk, which is local and not a backend call.
6. **Kube origin moves out of the config.** Add `allocation.WithBaseURL(origin)`. The client keeps a `baseURL` field that defaults to `cfg.BaseURL`, and the four URL builders (`BuildAllocationURL`, `ListBudgets`, `PredictSpecCost`, `Probe`) read it. `newClient` passes `WithBaseURL(conn.BaseURL)` and stops changing `cfg`. The alternative, clearing `BaseURL` inside `DryRun` when `Kube` is set, would teach the server package how `main` wires kube mode. `Info(cli.GetConfig())` reads only the profile, so it is unaffected.
7. **`HandleDryRun` uses `observeResult`.** It logs `rpc`, `requests`, `latency`, and `trace_id` like every other RPC.

## Risks / Trade-offs

- [`plugin inspect` shows `billed_cost` as `UNSUPPORTED` although actual cost has an amount] → the condition text cannot explain an `UNSUPPORTED` row, so the README `DryRun` paragraph says the amount is `ActualCostResult.cost`.
- [A future change fills another FOCUS field and forgets `DryRun`] → `TestDryRunMapsFocusFieldsForEverySupportedType` pins the exact supported and conditional sets, and the design names `focusFor` as the source to compare.
- [`WithBaseURL` with an empty `cfg.BaseURL` and no option] → unchanged behaviour: URL parsing fails the same way it does today.
