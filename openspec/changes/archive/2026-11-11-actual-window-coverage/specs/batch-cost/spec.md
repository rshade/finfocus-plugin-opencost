# Spec Delta

## ADDED Requirements

### Requirement: Actual batch items state the period their data covers

Each result in an actual `BatchCost` item SHALL carry the same charge period
as `GetActualCost`. The batch query has no filter, so the cluster-wide span
SHALL be the joined span of every row in that body, and the batch SHALL NOT
send a second allocation query for it. A short span SHALL log the same
warning as `GetActualCost`, with `resource` set to the descriptor id.

#### Scenario: Retention shorter than the batch window

- **WHEN** an actual batch for `oc-test` covers 2026-10-12 to 2026-11-11 and every row in the body starts at `2026-10-27T06:05:00Z`
- **THEN** the item cost is the row's `totalCost` and its charge period starts at `2026-10-27T06:05:00Z`
- **AND** the plugin logs the warning without a cluster query
