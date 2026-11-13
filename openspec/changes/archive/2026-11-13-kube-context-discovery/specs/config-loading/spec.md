# Spec Delta

## MODIFIED Requirements

### Requirement: Startup validation

The plugin SHALL validate the loaded config before it serves and SHALL
exit non-zero with a one-line reason when validation fails.
Without a `kube` block, `baseUrl` MUST be `https` with a host, or loopback
`http`, unless `allowInsecureHttp` is set. `profile` MUST be empty,
`opencost`, or `kubecost`. A set `currency` MUST be three upper-case
letters. `timeout`, `requestsPerSecond`, and `rateBurst` MUST NOT be
negative.

#### Scenario: Empty base URL

- **WHEN** `baseUrl` is empty and no `kube` block is set
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

## ADDED Requirements

### Requirement: A kube target excludes the direct URL settings

A set `kube.service` SHALL fail validation, naming the key, when `baseUrl`,
`tlsSkipVerify`, `caCertFile`, or `allowInsecureHttp` is also set. The
kubeconfig owns the address and the trust for that connection, so a setting
that would be ignored is an error. `kube.port`, when set, MUST be 1 to 65535.

#### Scenario: Both an address and a kube target

- **WHEN** `baseUrl` and `kube.service` are both set
- **THEN** validation fails with a reason naming `baseUrl` and `kube.service`

#### Scenario: Ignored trust setting

- **WHEN** `kube.service` and `caCertFile` are both set
- **THEN** validation fails with a reason naming `caCertFile`

#### Scenario: Port out of range

- **WHEN** `kube.port` is `70000`
- **THEN** validation fails with a reason naming `kube.port`

Tested by: TestKubeConfigValidation

### Requirement: Kube mode requires profile opencost

A set `kube.service` with profile `kubecost` SHALL fail validation naming
`profile`. The profile `kubecost` bearer token would collide with the
kubeconfig credentials. An empty profile is `opencost` and stays valid.

#### Scenario: Kubecost with a kube target

- **WHEN** `profile` is `kubecost` and `kube.service` is set
- **THEN** validation fails with a reason naming `profile`

#### Scenario: Default profile

- **WHEN** `profile` is empty and `kube.service` is set
- **THEN** validation passes

Tested by: TestKubeConfigValidation

### Requirement: Kube settings load from the environment

`OPENCOST_KUBE_CONTEXT`, `OPENCOST_KUBE_NAMESPACE`, `OPENCOST_KUBE_SERVICE`,
and `OPENCOST_KUBE_PORT` SHALL replace the same file keys. A set
`OPENCOST_KUBE_PORT` that is not an integer SHALL be an error naming that
variable. Unset or empty variables keep the file value.

#### Scenario: Environment wins

- **WHEN** the file sets `kube.service` to `a` and
  `OPENCOST_KUBE_SERVICE` is `b`
- **THEN** the loaded service is `b`

#### Scenario: Port that is not a number

- **WHEN** `OPENCOST_KUBE_PORT` is `nine`
- **THEN** loading fails with an error containing `OPENCOST_KUBE_PORT`

Tested by: TestKubeEnvOverrides
