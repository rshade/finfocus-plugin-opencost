# Proposal

## Why

OpenCost returns efficiency and request-versus-usage averages on every allocation row, and the plugin discards them (issue #136). They are real recorded values, not costs, and right-sizing (#138) needs them. Carrying them behind a flag that is off by default leaves every existing consumer unchanged.

## What Changes

- New config key `includeUsage` (YAML) and env `OPENCOST_INCLUDE_USAGE`, default `false`. The env value overrides the file when set; an unparsable value is a start-up error naming the variable.
- The allocation decoder reads `cpuEfficiency`, `ramEfficiency`, `gpuEfficiency`, `totalEfficiency`, `cpuCoreRequestAverage`, `cpuCoreUsageAverage`, `ramByteRequestAverage`, and `ramByteUsageAverage` as typed fields. They move from `IgnoredFields` to `ConsumedFields`.
- With the flag on, each `GetActualCost` result (and each actual `BatchCost` item) adds extended columns `efficiency.cpu`, `efficiency.ram`, `efficiency.total`, `request.cpu_cores`, `usage.cpu_cores`, `request.ram_bytes`, and `usage.ram_bytes`, plus `efficiency.gpu` only when the row has a GPU. The `__idle__` row carries none of them.
- No cost value changes.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `actual-cost`: actual cost results may carry usage and efficiency extended columns.
- `config-loading`: a new `includeUsage` key and `OPENCOST_INCLUDE_USAGE` override.

## Impact

- `internal/allocation`: `Config.IncludeUsage`, env loading, `Entry` fields, `wirefields.go`.
- `internal/server/actual.go`: `resultsFor` takes the flag and adds the columns.
- README configuration reference and `config.example.yaml` (`TestReadmeDocumentsEveryConfigKey`).
- Profile `kubecost` decodes the same keys if present. That is unverified: there is no `kubecost` allocation fixture.
