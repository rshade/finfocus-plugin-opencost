## MODIFIED Requirements

### Requirement: Project the requested resource

`GetProjectedCost` SHALL query allocation window `30d` for the resource in the
descriptor and SHALL NOT average the cluster. A namespace descriptor id is the
namespace name. Monthly cost SHALL be that object's `totalCost` divided by its
hours, times 730. `billing_detail` SHALL say the figure is a 30-day trailing
average projected to a 730-hour month. A non-empty `cost_breakdown` SHALL sum
to `cost_per_month`. `unit_price` SHALL be that object's observed hourly rate.
`currency` SHALL be the single currency named by the allocation body when
present, otherwise the pricing config currency. The plugin SHALL NOT assume
USD. Two different currency values in one body SHALL be `FailedPrecondition`.
No samples SHALL stay `NotFound` and SHALL NOT require a currency. A body and
config that both omit a currency SHALL be `FailedPrecondition`.

#### Scenario: Two namespaces project to different months

- **WHEN** `GetProjectedCost` is called twice for two namespaces in `allocation-namespace-60m.json` whose totals differ
- **THEN** each `cost_per_month` equals that namespace's `totalCost / (minutes / 60) * 730`
- **AND** the two monthly costs differ
- **AND** each `billing_detail` contains `30-day` and `730`
- **AND** each `cost_breakdown` sums to `cost_per_month`

#### Scenario: Non-USD config fills an empty response currency

- **WHEN** `GetProjectedCost` reads `allocation-namespace-60m.json` and the config currency is `EUR`
- **THEN** `currency` is `EUR`

#### Scenario: Response currency overrides config

- **WHEN** the body currency is `GBP` and the config currency is `EUR`
- **THEN** `currency` is `GBP`

#### Scenario: Missing currency

- **WHEN** the body names no currency and the config currency is empty
- **THEN** the RPC is `FailedPrecondition`
- **AND** the status message does not contain `USD`
