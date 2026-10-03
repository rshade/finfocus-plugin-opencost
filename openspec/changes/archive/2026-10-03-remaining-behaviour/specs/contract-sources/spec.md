# Spec Delta

## Purpose

Point each Kubecost contract fixture at the page or commit it was copied from, because Kubecost has no upstream module to marshal.

## ADDED Requirements

### Requirement: Every contract fixture has a source row

`testdata/kubecost-contract/SOURCES.md` SHALL name every JSON fixture in that
directory. The row SHALL repeat that fixture's citation source URL or
kubectl-cost commit and the checked date. The list SHALL say there is no
upstream module. The fixtures are not verified against live Kubecost.

#### Scenario: Each fixture is listed

- **WHEN** the contract directory contains a JSON fixture and its citation
- **THEN** `SOURCES.md` has one row that names the file, the citation source URL or commit, and the checked date
- **AND** the list says there is no upstream module

#### Scenario: A removed row fails

- **WHEN** the `budgets-health.json` row is removed
- **THEN** the source-list check fails
