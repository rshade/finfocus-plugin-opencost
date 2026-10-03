## ADDED Requirements

### Requirement: Typed allocation totals

The decoder SHALL read `totalCost`, `cpuCost`, and `ramCost` from typed fields
on each allocation object. It SHALL NOT take those totals from `rawAllocationOnly`.

#### Scenario: Recorded namespace totals

- **WHEN** `allocation-namespace-60m.json` is decoded
- **THEN** the `oc-test` entry reports total cost 0.1406, CPU cost 0.09373, and RAM cost 0.04687

### Requirement: Plain-text backend errors

The decoder SHALL return the response body when the HTTP status is 400 and the
body is plain text.

#### Scenario: Illegal window

- **WHEN** the body is the recorded `error-bad-window.json` text and the status is 400
- **THEN** the error contains `illegal window: notawindow`
- **AND** the error does not report a JSON syntax failure

### Requirement: Complete wire-key set

`ConsumedFields` and `IgnoredFields` SHALL be disjoint, and their union SHALL
equal the set of keys that appear on recorded allocation objects.

#### Scenario: Every recorded key is declared

- **WHEN** every `allocation-*.json` file under `testdata/opencost-real/` is scanned
- **THEN** each allocation key is in exactly one of the two maps
- **AND** every declared key appears in at least one recorded allocation
