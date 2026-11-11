# Spec Delta

## ADDED Requirements

### Requirement: Actual results state the period their data covers

Each `GetActualCost` result SHALL set `focus_record.charge_period_start` and
`focus_record.charge_period_end` to the matched row's `start` and `end` when
both parse as RFC3339. The cost SHALL stay the row's `totalCost` and SHALL NOT
be extrapolated.

#### Scenario: Charge period from the recorded row

- **WHEN** the resource id is `namespace/oc-test` and the body is `allocation-namespace-60m.json`
- **THEN** the result charge period is `2026-10-03T10:55:00Z` to `2026-10-03T10:57:48Z`

### Requirement: A backend data gap logs a warning

When the charge periods leave more of the window uncovered than the
tolerance, `GetActualCost` SHALL read the cluster-wide span (aggregate
`cluster`, no filter, same window). When that span is also short, the plugin
SHALL log the `warn` event `allocation data covers only part of the requested
window` with the window, `data_start`, `data_end`, and `coverage`.

#### Scenario: Retention shorter than the window

- **WHEN** the window is 2026-10-12 to 2026-11-11 and both the `oc-test` row and the cluster row run from `2026-10-27T06:05:00Z` to the window end
- **THEN** the cost is the row's `totalCost`
- **AND** the plugin logs the warning with `data_start` `2026-10-27T06:05:00Z` and `coverage` 21235/43200

#### Scenario: Object created inside a retained window

- **WHEN** the `oc-test` row starts on 2026-10-27 and the cluster row covers the whole window
- **THEN** no warning is logged

#### Scenario: Cluster-wide query fails

- **WHEN** the row is short and the cluster-wide query returns HTTP 500 or no rows
- **THEN** the call succeeds with the row's cost and no warning is logged

### Requirement: Coverage tolerates alignment and lag

A span SHALL be short only when the uncovered time before and after it exceeds
one hour or one percent of the window, whichever is longer. The window SHALL
end at the earlier of its end and the query time.

#### Scenario: Step alignment

- **WHEN** the row starts five minutes after the window start and ends five minutes before the window end
- **THEN** the cluster-wide span is not read and no warning is logged

#### Scenario: Window ending after the query

- **WHEN** the window ends tomorrow and the row ends two minutes before the query
- **THEN** no warning is logged
