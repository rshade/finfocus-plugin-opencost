# Design

## Context

`LoadConfigFromEnvOrFile` drops YAML errors (`_ = yaml.Unmarshal`) and
`getenvDuration` falls back to the default on a parse failure.
`KUBECOST_TLS_SKIP_VERIFY` is true only for the exact text `true`.
`main.go` goes from load straight to `pluginsdk.Serve` with no
validation, so a bad `baseUrl` fails only at the first cost call.

## Goals / Non-Goals

Goals: parse errors returned with the file path or variable name;
`Config.Validate` called from `main.go` before `Serve`; precedence
documented.

Non-goals: `KnownFields` strict decoding, probing `baseUrl` over the
network, hot reload, new config keys, failing a missing config file.

## Decisions

- **`KUBECOST_TLS_SKIP_VERIFY` accepts only `true` and `false`.**
  Alternative: `strconv.ParseBool`, which also accepts `1`, `t`, `True`.
  That would turn verify off for a deployment whose `1` silently meant
  verify-on today — a security-relevant change the task does not ask
  for. Rejecting everything but `true`/`false` fails closed and loud.
- **Validate in `main.go`, not in `NewClient`.** Tests and library
  callers build clients with partial configs (no `baseUrl`); making the
  constructor reject them widens the blast radius beyond the startup
  path the issue scopes.
- **Errors name the file path, variable, or key — never a value.**
  Values can carry secrets; names cannot.

## Risks / Trade-offs

- [A deployment running with a silently-broken config now exits at
  startup] → That is the point of the issue; the error names the fix.
- [`yaml.v3` type errors for `timeout: 45s`] → yaml.v3 parses duration
  strings into `time.Duration` already; only real type mismatches error.

## Migration Plan

None. Valid configs behave exactly as before.
