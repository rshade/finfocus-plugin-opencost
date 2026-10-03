# projected-cost Specification

## Purpose

Project one Kubernetes object's trailing allocation to a 730-hour month.

## Requirements

### Requirement: Project the requested resource

`GetProjectedCost` SHALL query allocation window `30d` for the resource in the
descriptor and SHALL NOT average the cluster. A namespace descriptor id is the
namespace name. Monthly cost SHALL be that object's `totalCost` divided by its
hours, times 730. `billing_detail` SHALL say the figure is a 30-day trailing
average projected to a 730-hour month. A non-empty `cost_breakdown` SHALL sum
to `cost_per_month`.

#### Scenario: Two namespaces project to different months

- **WHEN** `GetProjectedCost` is called twice for two namespaces in `allocation-namespace-60m.json` whose totals differ
- **THEN** each `cost_per_month` equals that namespace's `totalCost / (minutes / 60) * 730`
- **AND** the two monthly costs differ
- **AND** each `billing_detail` contains `30-day` and `730`
- **AND** each `cost_breakdown` sums to `cost_per_month`
