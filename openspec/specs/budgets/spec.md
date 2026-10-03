# budgets Specification

## Purpose

List namespace budget rules from the Kubecost budget API.

## Requirements

### Requirement: Namespace budgets come from the list endpoint

When the profile is `kubecost`, `GetBudgets` SHALL send `GET /model/budgets`.
It SHALL return rules whose `values.namespace` is set and SHALL omit cluster
rules. `amount.limit` SHALL be `spendLimit`. `amount.currency` SHALL come from
pricing config and SHALL NOT assume USD. `period` SHALL be monthly or weekly
from `interval`. A namespace tag SHALL name the namespace. An action
`percentage` SHALL become an actual threshold. `include_status` SHALL add
`currentSpend`. The contract is not verified against live Kubecost.

#### Scenario: Namespace rule beside a cluster rule

- **WHEN** profile `kubecost` lists the budget contract fixture with `include_status` set
- **THEN** the request is `GET /model/budgets`
- **AND** the only budget is the namespace rule
- **AND** `amount.limit` equals that rule's `spendLimit`
- **AND** `currency` is the configured currency
- **AND** `status.current_spend` equals that rule's `currentSpend`

#### Scenario: Status omitted

- **WHEN** profile `kubecost` lists the same fixture without `include_status`
- **THEN** the namespace budget has no status

#### Scenario: Namespace filter

- **WHEN** profile `kubecost` lists two namespace rules and one cluster rule, and the request tag `namespace` names the weekly rule
- **THEN** the only budget is that weekly rule
- **AND** `period` is weekly
- **AND** metadata `intervalDay` and `kind` match the rule
- **AND** a request for an unknown namespace returns no budgets
- **AND** a request without a namespace tag returns both namespace rules and not the cluster rule

#### Scenario: No configured currency

- **WHEN** profile `kubecost` lists budgets and pricing config has no currency
- **THEN** the RPC is `FailedPrecondition`
- **AND** the message does not contain `USD`
