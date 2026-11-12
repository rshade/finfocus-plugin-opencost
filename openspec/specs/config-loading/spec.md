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

Tested by: TestConfigFileParseErrorNamesPath, TestLoadConfigFromEnvOrFile_FileNotFound, TestLoadConfigFromEnvOrFile_WithFile

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

Tested by: TestEnvParseErrorNamesVariable, TestConfigDefaults

### Requirement: Startup validation

The plugin SHALL validate the loaded config before it serves and SHALL
exit non-zero with a one-line reason when validation fails.
`baseUrl` MUST be `https` with a host, or loopback `http`, unless
`allowInsecureHttp` is set. `profile` MUST be empty, `opencost`, or
`kubecost`. A set `currency` MUST be three upper-case letters.
`timeout`, `requestsPerSecond`, and `rateBurst` MUST NOT be negative.

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

Tested by: TestConfigValidate, TestBinaryExitsOnInvalidConfig

### Requirement: The token never appears in an error or log

No config error or startup log line SHALL contain the API token. Error
messages name the file path, variable name, or config key, not the
offending secret value.

#### Scenario: Invalid config with a token set

- **WHEN** `KUBECOST_API_TOKEN` is set and any config value is invalid
- **THEN** the error message does not contain the token

Tested by: TestConfigErrorsNeverContainToken, TestTokenIsEnvOnlyAndAbsentFromLogs

### Requirement: Plain HTTP stays on loopback

An `http` `baseUrl` SHALL fail validation unless the host is loopback
or `allowInsecureHttp` is set. Loopback means `localhost`, an address
in `127.0.0.0/8`, or `::1`. The error SHALL contain
`base URL uses http; use https or set allowInsecureHttp`.
An `https` URL SHALL stay accepted when the flag is unset.

#### Scenario: Public HTTP rejected

- **WHEN** `baseUrl` is `http://example.com` and `allowInsecureHttp`
  is unset
- **THEN** validation fails with that error text

#### Scenario: Loopback HTTP accepted

- **WHEN** `baseUrl` is `http://localhost:9090`,
  `http://127.0.0.1:9090`, or `http://[::1]:9090`
- **THEN** validation passes with `allowInsecureHttp` unset

#### Scenario: Opt-in public HTTP

- **WHEN** `allowInsecureHttp` is set and `baseUrl` is
  `http://example.com`
- **THEN** validation passes

#### Scenario: HTTPS unchanged

- **WHEN** `baseUrl` is `https://example.com` and
  `allowInsecureHttp` is unset
- **THEN** validation passes

Tested by: TestInsecureHTTPRejectsNonLoopback

### Requirement: Skip-verify logs one warning

Startup SHALL log one warning containing
`TLS certificate verification is disabled` when `tlsSkipVerify` is on,
and SHALL log that warning zero times when it is off.
The warning SHALL NOT contain the API token or URL userinfo.

#### Scenario: Warning when verification is skipped

- **WHEN** the process starts with `tlsSkipVerify` on
- **THEN** stderr contains that warning once

#### Scenario: Silent when verification is on

- **WHEN** the process starts with `tlsSkipVerify` off
- **THEN** stderr does not contain that warning

#### Scenario: Secret stays out of the warning

- **WHEN** a token and URL userinfo are set and `tlsSkipVerify` is on
- **THEN** the startup log contains neither secret

Tested by: TestTLSSkipVerifyWarnsAtStartup

### Requirement: Private CA file extends trust

A set `caCertFile` SHALL add its PEM certificates to the trust pool
for the backend. A missing or unparsable file SHALL fail startup with
an error naming `caCertFile`. An empty `caCertFile` SHALL keep the
default roots.

#### Scenario: Private server accepted with the file

- **WHEN** `caCertFile` holds the PEM of a server the default roots
  reject
- **THEN** a request to that server succeeds

#### Scenario: Private server rejected without the file

- **WHEN** `caCertFile` is empty and the server uses that certificate
- **THEN** the request fails certificate verification

#### Scenario: Missing or unparsable file

- **WHEN** `caCertFile` is missing or is not a PEM certificate
- **THEN** startup validation fails with an error naming `caCertFile`

Tested by: TestCACertFileTrustsPrivateCA
