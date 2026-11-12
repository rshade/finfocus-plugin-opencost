# Proposal

## Why

The plugin logger set the field `plugin` (`{"level":"warn","plugin":"opencost",...}`).
finfocus-spec defines the plugin identity field as `plugin_name`
(`pluginsdk.FieldPluginName`), and `pluginsdk.NewPluginLogger` sets it with
`plugin_version`. Core's `finfocus logs` and web Logs view read `plugin_name`
to name a log line's source. Core writes plugin stderr to its log file with no
tag, so every opencost line, including the `allocation data covers only part
of the requested window` warning, showed as `[core]`, and
`finfocus logs --source opencost` found nothing.

## What Changes

- `main.go` builds its logger with `pluginsdk.NewPluginLogger("opencost",
  version.Version, zerolog.TraceLevel, os.Stderr)`. Lines carry `plugin_name`
  and `plugin_version` and no `plugin` field.
- The level stays trace and the writer stays stderr, as before.

## Non-goals

- No change to finfocus-spec or core.
- No change to log messages or other fields.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `plugin-health`: log lines name the plugin with the SDK field.

## Impact

- `cmd/finfocus-plugin-opencost/main.go` and its startup test.
- `README.md` and `CLAUDE.md`.
