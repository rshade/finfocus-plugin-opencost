# Proposal

## Why

A malformed config file and an unparseable `KUBECOST_TIMEOUT` are silently
dropped today (`_ = yaml.Unmarshal`, a duration helper that falls back to
the default). A missing or mistyped `baseUrl` surfaces only at the first
cost call as `unsupported protocol scheme`. The plugin should refuse to
start with a one-line reason that names the file or variable.

## What Changes

- A YAML syntax or type error in the config file is an error that names
  the file path. Strict decoding (`KnownFields`) stays off.
- An unparseable `KUBECOST_TIMEOUT` or a `KUBECOST_TLS_SKIP_VERIFY` value
  other than `true` or `false` is an error that names the variable.
- **BREAKING**: `Config.Validate` runs before `pluginsdk.Serve`. An empty
  or non-`http(s)` `baseUrl`, a profile outside `opencost` and `kubecost`,
  a set currency that is not three upper-case letters, or a negative
  `timeout`, `requestsPerSecond`, or `rateBurst` exits the process
  non-zero with a one-line reason.
- A missing config file stays tolerated, including a mistyped
  `OPENCOST_CONFIG` path.
- No error or log line ever contains the API token.
- The README states the real precedence: defaults, then environment, then
  the file for keys it sets, then `OPENCOST_PROFILE`, `OPENCOST_CURRENCY`,
  and `KUBECOST_API_TOKEN`, which always win.

## Capabilities

### New Capabilities

- `config-loading`: Loading configuration from environment and file,
  reporting parse errors with the file path or variable name, validating
  the result before the plugin serves, and keeping the token out of
  errors and logs.

### Modified Capabilities

- None.

## Impact

- `internal/allocation/config.go`: parse errors return, new `Validate`.
- `cmd/finfocus-plugin-opencost/main.go`: validate before `Serve`.
- `README.md`: precedence text.
- Existing valid configs are unaffected; invalid ones that were silently
  ignored now stop startup.
