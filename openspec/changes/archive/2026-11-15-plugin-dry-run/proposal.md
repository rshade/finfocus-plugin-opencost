# Proposal

## Why

`finfocus plugin inspect opencost <type>` fails with "plugin 'opencost' does not support inspection (capability discovery not implemented)" (issue #134). Core's `plugin inspect` calls the `DryRun` RPC, and the plugin leaves it at the SDK default, `Unimplemented`. azure-public answers it. The plugin already knows which types it accepts and which FOCUS columns actual cost fills, so it can say so without a backend call.

## What Changes

- Implement `DryRun`. A supported type returns one field mapping per FOCUS field name from the SDK. `billing_currency` is `SUPPORTED`. `charge_period_start`, `charge_period_end`, `tags`, and `extended_columns` are `CONDITIONAL` with a condition. Every other field is `UNSUPPORTED`.
- An unsupported type returns `resource_type_supported=false`, no field mappings, and no gRPC error. A request with no resource is `InvalidArgument`.
- `configuration_valid` and `configuration_errors` come from the same config validation the binary runs at start-up. `DryRun` sends no request to the allocation backend.
- In kube mode the allocation client keeps the operator's config unchanged and holds the API server origin separately, so the config it reports still passes validation. Today `newClient` writes the origin into `baseUrl`, which validation rejects next to `kube.service`.
- `GetPluginInfo` reports `DRY_RUN` on both profiles. `plugin.manifest.json` lists `DRY_RUN`.

The field names follow `costsource.proto`: a `FieldMapping.field_name` must name a `FocusCostRecord` field. The issue's first task listed input keys (`id`, `metadata.name`, `metadata.namespace`). Those are not FOCUS fields, so they are not mappings.

## Capabilities

### New Capabilities

- `dry-run`: what `DryRun` returns for a resource type: the FOCUS field mappings, type support, and configuration validity, without calling the backend.

### Modified Capabilities

- `plugin-info`: the reported capability set and the manifest capability list add `DRY_RUN`.

## Impact

- `internal/server`: new `dryrun.go` (`HandleDryRun`, which the SDK server calls for the `DryRun` RPC). `info.go` adds `DRY_RUN`.
- `internal/allocation/client.go`: new `WithBaseURL` option and a client-held base URL. The four URL builders read it instead of `cfg.BaseURL`.
- `cmd/finfocus-plugin-opencost/client.go`: passes the origin with `WithBaseURL` instead of writing `cfg.BaseURL`.
- `plugin.manifest.json`, README capability list, `naming_test.go`, and `plugininfo_test.go` pins.
- No new dependency and no `finfocus-spec` bump (v0.7.12 has `DryRunHandler` and the builders).
- Follow-up outside this repo: the `opencost` capability list in the `rshade/finfocus` registry entry, after release.
