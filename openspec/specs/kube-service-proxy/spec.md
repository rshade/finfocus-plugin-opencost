# kube-service-proxy Specification

## Purpose

Reach OpenCost through the Kubernetes API server service proxy using the
caller's kubeconfig or in-cluster credentials, so a CI runner, a workstation,
and a pod need no hand-run port-forward, and report each way that can fail.

## Requirements

### Requirement: A kube target routes requests through the service proxy

When `kube.service` is set, the plugin SHALL send every allocation and
health request to the API server path
`/api/v1/namespaces/<namespace>/services/<service>:<port>/proxy/` followed
by the path and query it would send to `baseUrl`. `namespace` defaults to
`opencost` and `port` to `9003`. The query string SHALL be identical to the
`baseUrl` form.

#### Scenario: Default namespace and port

- **WHEN** `kube.service` is `opencost` and `namespace` and `port` are unset
- **THEN** an allocation request goes to
  `/api/v1/namespaces/opencost/services/opencost:9003/proxy/allocation`

#### Scenario: Explicit namespace and port

- **WHEN** `kube.namespace` is `cost`, `kube.service` is `oc`, and
  `kube.port` is `8080`
- **THEN** the request path starts
  `/api/v1/namespaces/cost/services/oc:8080/proxy/`

#### Scenario: Same query as baseUrl

- **WHEN** the same namespace request is made in kube mode and in `baseUrl`
  mode
- **THEN** the query strings are equal

Tested by: TestKubeTargetBuildsServiceProxyURL, TestKubeDefaultsNamespaceAndPort

### Requirement: Credentials come from the kubeconfig or the pod

The plugin SHALL read the kubeconfig from `kube.kubeconfig`, then the
`KUBECONFIG` environment variable, then the default kubeconfig path. It SHALL
use `kube.context` when set, otherwise the current context. With no
kubeconfig and a pod service account present, it SHALL use in-cluster
configuration. Static certificates, static tokens, and exec plugins SHALL
all authenticate.

#### Scenario: Context selected

- **WHEN** the kubeconfig has two contexts and `kube.context` names the second
- **THEN** requests go to the second context's API server

#### Scenario: Exec plugin credentials

- **WHEN** the selected user authenticates with an exec plugin
- **THEN** the plugin runs it and sends the token it returns

#### Scenario: In-cluster fallback

- **WHEN** no kubeconfig exists, `KUBECONFIG` is unset, and a service
  account token is mounted
- **THEN** requests use the in-cluster API server and token

#### Scenario: KUBECONFIG names no usable kubeconfig

- **WHEN** `KUBECONFIG` is set to a missing file or a file with no
  contexts and a service account token is mounted
- **THEN** startup fails with a reason naming `KUBECONFIG`
- **AND** the pod's service account is not used

Tested by: TestKubeCredentialSources, TestKubeCredentialSourcesInCluster, TestKubeExecPluginCredentials, TestKubeSetKubeconfigNeverFallsBackToPod

### Requirement: A missing context fails at startup

A `kube.context` that is not in the kubeconfig, or a kube target with no
kubeconfig and no pod service account, SHALL fail startup with a one-line
reason. The reason SHALL name the context or say that no kubeconfig was
found, and the plugin SHALL NOT serve.

#### Scenario: Unknown context

- **WHEN** `kube.context` is `prod-east` and the kubeconfig has no such context
- **THEN** startup fails with a reason containing `prod-east`

#### Scenario: No credentials at all

- **WHEN** `kube.service` is set, no kubeconfig exists, and no service
  account is mounted
- **THEN** startup fails with a reason saying no kubeconfig was found

Tested by: TestKubeMissingContextNamesContext

### Requirement: A denied proxy request names the missing grant

An API server `403` on the proxy path SHALL return gRPC `PermissionDenied`.
The message SHALL name the verb and resource `get services/proxy`, the
namespace, and the service. It SHALL NOT contain a credential.

#### Scenario: RBAC forbids the proxy

- **WHEN** the API server answers `403` for the proxy path
- **THEN** the RPC fails with `PermissionDenied`
- **AND** the message contains `services/proxy`, the namespace, and the
  service name

Tested by: TestKubeForbiddenIsPermissionDenied

### Requirement: Rejected credentials are Unauthenticated

An API server `401` on the proxy path SHALL return gRPC `Unauthenticated`.
The message SHALL name the context and SHALL NOT contain a credential.

#### Scenario: API server rejects the credentials

- **WHEN** the API server answers `401` with a `Status` body for the proxy path
- **THEN** the RPC fails with `Unauthenticated`
- **AND** the message names the context

Tested by: TestKubeTransportMapsAPIServerFailures

### Requirement: An unreachable service is distinct from missing cost data

A missing service, a service with no ready endpoints, or an API server that
cannot be reached SHALL return gRPC `Unavailable` and name the context,
namespace, and service. It SHALL NOT return `NotFound` or the no-data
response, because those mean OpenCost answered.

#### Scenario: Service does not exist

- **WHEN** the API server answers `404` with a `Status` body for the service
- **THEN** the RPC fails with `Unavailable` naming the service

#### Scenario: No ready endpoints

- **WHEN** the API server answers `503` because no endpoints are available
- **THEN** the RPC fails with `Unavailable` naming the service

#### Scenario: OpenCost answered with no row

- **WHEN** OpenCost answers `200` with no matching row
- **THEN** the result is the no-data response, not `Unavailable`

Tested by: TestKubeUnreachableServiceIsUnavailable

### Requirement: The health check uses the proxy

`Check` SHALL send `GET /healthz` through the same proxy path. A status
below 400 from OpenCost SHALL be healthy. A `403`, a missing service, or an
unreachable API server SHALL be unhealthy with the same reason a cost RPC
would give.

#### Scenario: Healthy through the proxy

- **WHEN** OpenCost answers `/healthz` with `200` through the proxy
- **THEN** `Check` reports healthy

#### Scenario: Forbidden health probe

- **WHEN** the API server answers `403` for `/healthz`
- **THEN** `Check` is unhealthy and the reason names `services/proxy`

Tested by: TestKubeHealthProbeUsesProxy

### Requirement: Kube credentials never appear in a log or error

No log line or error SHALL contain a bearer token, client key, or exec
plugin output. Startup SHALL log one line naming the context, namespace,
service, and port.

#### Scenario: Token stays out of output

- **WHEN** the selected user holds a static token and any request fails
- **THEN** neither the error nor stderr contains the token

#### Scenario: One startup line

- **WHEN** the plugin starts in kube mode
- **THEN** stderr has one line naming the context, namespace, service,
  and port

Tested by: TestKubeCredentialsNeverLogged
