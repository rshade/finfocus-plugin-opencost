## ADDED Requirements

### Requirement: A namespace row says it includes its workloads

For a namespace descriptor, `billing_detail` SHALL keep the 30-day and 730-hour
wording and SHALL also say the figure includes every workload in the namespace.
The text for any other kind SHALL NOT claim this.

#### Scenario: Namespace row

- **WHEN** `GetProjectedCost` is called for a namespace
- **THEN** `billing_detail` contains `30-day` and `730`
- **AND** `billing_detail` contains `includes every workload in the namespace`

#### Scenario: Workload row

- **WHEN** `GetProjectedCost` is called for a controller
- **THEN** `billing_detail` does not contain `includes every workload`

Tested by: TestProjectedBillingDetailNamespaceIncludesWorkloads
