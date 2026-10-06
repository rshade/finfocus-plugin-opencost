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

### Requirement: Budget health and summary counts

When `include_status` is true, each returned budget's health SHALL come from
`currentSpend`, `spendLimit`, and the lowest positive action `percentage`.
Spend over `spendLimit` SHALL be exceeded. Spend equal to `spendLimit` SHALL
be critical. Spend at or above that percentage and under the limit SHALL be
warning. Spend below that percentage SHALL be ok. The summary SHALL count ok,
warning, critical, and exceeded for the budgets the filter returns. Those four
counts SHALL equal `total_budgets`. Critical SHALL stay in its own count. A
namespace filter SHALL count only the budgets that filter returns. Without
`include_status`, the summary SHALL be absent. The contract is not verified
against live Kubecost.

#### Scenario: Four health states beside a cluster rule

- **WHEN** profile `kubecost` lists the health contract fixture with `include_status` set
- **THEN** each namespace budget health matches its spend, limit, and lowest action percentage
- **AND** the summary counts equal the returned budgets
- **AND** the cluster rule is absent from the budgets and from the summary total
- **AND** a namespace tag for one rule counts only that rule
- **AND** an unknown namespace returns no budgets and a zero summary
- **AND** a request without `include_status` has no summary

### Requirement: Invalid budget rules are skipped

A namespace budget rule that fails validation SHALL be skipped and the
remaining rules SHALL be returned. Each skipped rule SHALL log one WARN
naming the rule id and the reason. The first returned budget SHALL carry
`skippedRules` with the count and `skippedRuleReasons` with each
`id: reason` pair joined by `;` in `metadata`. When no rule survives,
the response SHALL be an empty list. The summary SHALL count only
returned budgets.

#### Scenario: One bad interval beside two valid rules

- **WHEN** the list has three namespace rules and one has interval `daily`
- **THEN** the call succeeds with two budgets
- **AND** one WARN names the skipped rule id and the interval reason
- **AND** the first budget's metadata names the skipped rule and reason

#### Scenario: Non-positive spend limit

- **WHEN** one namespace rule has `spendLimit` of `0`
- **THEN** that rule is skipped and the other rules are returned

#### Scenario: Every rule invalid

- **WHEN** every namespace rule fails validation
- **THEN** the call succeeds with an empty budget list
- **AND** each skipped rule has one WARN

#### Scenario: Summary ignores skipped rules

- **WHEN** `include_status` is set and one rule is skipped
- **THEN** the summary counts only the returned budgets
