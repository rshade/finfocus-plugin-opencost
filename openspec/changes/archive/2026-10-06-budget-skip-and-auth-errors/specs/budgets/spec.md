# Spec Delta

## ADDED Requirements

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
