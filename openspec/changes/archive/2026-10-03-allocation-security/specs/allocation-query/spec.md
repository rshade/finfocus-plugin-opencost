## ADDED Requirements

### Requirement: TLS verification is on by default

The allocation client SHALL verify TLS certificates. `tlsSkipVerify` SHALL
default to false. Setting `tlsSkipVerify` SHALL skip certificate verification
for that client.

#### Scenario: Untrusted certificate

- **WHEN** the allocation URL is HTTPS and the server certificate is not trusted
- **THEN** the allocation request fails
- **AND** the error mentions the certificate

#### Scenario: Explicit skip

- **WHEN** `tlsSkipVerify` is true for that same server
- **THEN** the allocation request succeeds

### Requirement: API token comes from the environment

The API token SHALL be read from `KUBECOST_API_TOKEN`. A YAML `apiToken` SHALL
be ignored. Request logs and allocation error strings SHALL NOT contain the
token.

#### Scenario: YAML token is ignored

- **WHEN** the config file sets `apiToken` and `KUBECOST_API_TOKEN` is empty
- **THEN** the loaded token is empty

#### Scenario: Environment token is sent and not logged

- **WHEN** the profile is `kubecost` and `KUBECOST_API_TOKEN` is set
- **THEN** the allocation request sends `Authorization: Bearer` plus that token
- **AND** the request log contains `allocation request`
- **AND** the request log does not contain the token
- **AND** a backend error string does not contain the token

### Requirement: Hostile filter values are rejected

A filter key or value that contains a double quote, plus, backslash,
whitespace, or parenthesis SHALL be rejected before the HTTP request.
`GetActualCost` SHALL return `InvalidArgument`. The status message SHALL NOT
echo the rejected value.

#### Scenario: Hostile namespace

- **WHEN** `GetActualCost` is called for a namespace whose name contains one of those characters
- **THEN** the backend is not called
- **AND** the status is `InvalidArgument`
- **AND** the status message does not contain that namespace name
