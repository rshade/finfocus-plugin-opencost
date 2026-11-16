# Spec Delta

## ADDED Requirements

### Requirement: Usage columns are off by default

When `includeUsage` is off, `GetActualCost` and the actual `BatchCost` path SHALL add no `efficiency.*`, `request.*`, or `usage.*` extended column to any result.

#### Scenario: Default configuration

- **WHEN** `GetActualCost` returns the recorded `oc-test` row with `includeUsage` unset
- **THEN** no extended column starts with `efficiency.`, `request.`, or `usage.`

Tested by: TestActualCostUsageColumnsOffByDefault

### Requirement: Usage columns carry the recorded row values

When `includeUsage` is on, each actual cost result SHALL add `efficiency.cpu`, `efficiency.ram`, `efficiency.total`, `request.cpu_cores`, `usage.cpu_cores`, `request.ram_bytes`, and `usage.ram_bytes` from the matched row, formatted as the shortest exact decimal. It SHALL add `efficiency.gpu` only when the row has a GPU, and SHALL add none of them to the `__idle__` row. Cost values SHALL NOT change.

#### Scenario: Recorded oc-test row

- **WHEN** `GetActualCost` returns the recorded `oc-test` row with `includeUsage` on
- **THEN** `request.cpu_cores` is `1`, `usage.cpu_cores` is `0`, `efficiency.ram` is `0.00188`, and `efficiency.total` is `0.00063`
- **AND** there is no `efficiency.gpu` column and the cost equals the flag-off cost

#### Scenario: Idle and GPU rows

- **WHEN** a body has an `__idle__` row and a row with `gpuCount` above zero
- **THEN** the idle row has no usage column and the GPU row has `efficiency.gpu`

Tested by: TestActualCostUsageColumnsFromRecordedRow, TestUsageColumnsSkipIdleAndNeedAGPU
