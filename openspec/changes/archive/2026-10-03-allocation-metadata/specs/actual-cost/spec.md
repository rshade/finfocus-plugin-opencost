## ADDED Requirements

### Requirement: Allocation metadata is carried on the result

`GetActualCost` SHALL copy labels, controller kind, and annotations from the
matched allocation onto `ActualCostResult.focus_record`. Labels SHALL be the
focus `tags`. A non-empty controller kind SHALL be extended column
`controllerKind`. Each annotation SHALL be extended column `annotation.<key>`.
A row with none of the three SHALL leave `focus_record` unset. The record
SHALL NOT set a currency.

#### Scenario: Recorded pod labels and controller kind

- **WHEN** the resource id is `pod/oc-test/fixed-7996d494d-fbz99` and the body is `allocation-pod-60m.json`
- **THEN** the result tags equal that row's `properties.labels`
- **AND** extended column `controllerKind` equals that row's `properties.controllerKind`
- **AND** no extended column key starts with `annotation.`

#### Scenario: Annotation on the recorded pod

- **WHEN** that same row also has annotation `team` equal to `platform`
- **THEN** extended column `annotation.team` is `platform`
- **AND** the labels and controller kind stay equal to the recorded row
