# allocation-query Specification

## Purpose

Build a stable allocation URL for the OpenCost and Kubecost endpoint profiles.

## Requirements

### Requirement: OpenCost allocation profile

The client SHALL request `GET /allocation` when the profile is `opencost` or unset.
The query SHALL include `includeIdle` and `shareIdle` and SHALL NOT include `accumulate`.
The request SHALL NOT require a bearer token.

#### Scenario: OpenCost URL

- **WHEN** the profile is `opencost` and the query window, aggregate, and filters are fixed
- **THEN** the URL path is `/allocation`
- **AND** the query string contains `includeIdle` and `shareIdle`
- **AND** the query string does not contain `accumulate`

#### Scenario: OpenCost omits the token

- **WHEN** the profile is `opencost` and an API token is configured
- **THEN** the allocation request does not send an `Authorization` header

### Requirement: Kubecost allocation profile

The client SHALL request `GET /model/allocation` when the profile is `kubecost`.
The query SHALL include `idle`, `accumulate`, and `shareIdle`.
The request SHALL send `Authorization: Bearer <token>` when a token is configured.

#### Scenario: Kubecost URL

- **WHEN** the profile is `kubecost` and the query window, aggregate, and filters are fixed
- **THEN** the URL path is `/model/allocation`
- **AND** the query string contains `idle`, `accumulate`, and `shareIdle`

#### Scenario: Kubecost sends the token

- **WHEN** the profile is `kubecost` and an API token is configured
- **THEN** the allocation request sends `Authorization: Bearer` plus that token

### Requirement: Stable query strings

The client SHALL build the identical URL for the same profile and query on every call.
Filter keys SHALL be sorted before they are joined.

#### Scenario: Repeated build

- **WHEN** `BuildAllocationURL` is called twice with the same profile and query
- **THEN** both results are equal
