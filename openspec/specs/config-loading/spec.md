# config-loading Specification

## Purpose

Load configuration from the environment and an optional YAML file, report
parse errors with the file path or variable name, and validate the result
before the plugin serves so a bad config stops startup with a one-line
reason.

## Requirements

### Requirement: Configuration file parse errors name the file

A config file that fails to parse SHALL be an error naming the file path.
A missing config file SHALL be tolerated and leaves the environment and
default values in place. Unknown keys in the file SHALL NOT be an error.

#### Scenario: Malformed YAML

- **WHEN** the config file contains YAML that does not parse
- **THEN** loading fails with an error containing the file path
- **AND** the plugin does not serve

#### Scenario: Wrong YAML type

- **WHEN** a config file key holds a value of the wrong type, such as a
  list where `timeout` expects a duration
- **THEN** loading fails with an error containing the file path

#### Scenario: Missing file tolerated

- **WHEN** the config path does not exist, including a mistyped
  `OPENCOST_CONFIG` path
- **THEN** loading succeeds with the environment and default values

#### Scenario: Unknown keys ignored

- **WHEN** the config file contains a key the plugin does not know
- **THEN** loading succeeds and the key is ignored

### Requirement: Environment parse errors name the variable

A set `KUBECOST_TIMEOUT` that does not parse as a duration SHALL be an
error naming `KUBECOST_TIMEOUT`. A set `KUBECOST_TLS_SKIP_VERIFY` other
than `true` or `false` SHALL be an error naming
`KUBECOST_TLS_SKIP_VERIFY`. An unset or empty variable keeps the default.

#### Scenario: Timeout without a unit

- **WHEN** `KUBECOST_TIMEOUT` is `30` with no unit
- **THEN** loading fails with an error containing `KUBECOST_TIMEOUT`

#### Scenario: Non-boolean TLS skip verify

- **WHEN** `KUBECOST_TLS_SKIP_VERIFY` is `yes`
- **THEN** loading fails with an error containing
  `KUBECOST_TLS_SKIP_VERIFY`

#### Scenario: Default on empty

- **WHEN** `KUBECOST_TIMEOUT` and `KUBECOST_TLS_SKIP_VERIFY` are unset
- **THEN** loading succeeds with a 15 second timeout and certificate
  verification on

### Requirement: Startup validation

The plugin SHALL validate the loaded config before it serves and SHALL
exit non-zero with a one-line reason when validation fails.
`baseUrl` MUST be an `http` or `https` URL with a host. `profile` MUST
be empty, `opencost`, or `kubecost`. A set `currency` MUST be three
upper-case letters. `timeout`, `requestsPerSecond`, and `rateBurst` MUST
NOT be negative.

#### Scenario: Empty base URL

- **WHEN** `baseUrl` is empty
- **THEN** validation fails with a reason naming `baseUrl` and the
  process exits non-zero before serving

#### Scenario: Non-HTTP base URL

- **WHEN** `baseUrl` is `ftp://example.com` or has no host
- **THEN** validation fails with a reason naming `baseUrl`

#### Scenario: Unknown profile

- **WHEN** `profile` is `opensost`
- **THEN** validation fails with a reason naming the profile value

#### Scenario: Lowercase currency

- **WHEN** `currency` is `eur`
- **THEN** validation fails with a reason naming `currency`

#### Scenario: Negative rate

- **WHEN** `requestsPerSecond` is `-1`
- **THEN** validation fails with a reason naming `requestsPerSecond`

#### Scenario: Valid config serves

- **WHEN** the config is valid
- **THEN** validation passes and the plugin serves

### Requirement: The token never appears in an error or log

No config error or startup log line SHALL contain the API token. Error
messages name the file path, variable name, or config key, not the
offending secret value.

#### Scenario: Invalid config with a token set

- **WHEN** `KUBECOST_API_TOKEN` is set and any config value is invalid
- **THEN** the error message does not contain the token
