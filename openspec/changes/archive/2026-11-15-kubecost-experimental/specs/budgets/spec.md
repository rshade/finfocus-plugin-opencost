# Spec Delta

## ADDED Requirements

### Requirement: Budgets say they are unverified

Every budget `GetBudgets` returns SHALL carry metadata `verified` = `false`, because profile `kubecost` is verified only against contract fixtures.

#### Scenario: Contract fixture budgets

- **WHEN** `GetBudgets` returns the budgets from the contract fixture
- **THEN** each budget has metadata `verified` set to `false`

Tested by: TestBudgetsCarryVerifiedFalse
