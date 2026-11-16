# Design

## Context

`resultsFor` (`internal/server/actual.go`) builds one `ActualCostResult` per matched row for `GetActualCost` and the actual `BatchCost` path. `focusFor` copies labels, `controllerKind`, and annotations into the FOCUS record; extended columns are the provider-specific extension. `Entry` already decodes `cpuEfficiency`, `ramEfficiency`, and `totalEfficiency`, but the wire manifest lists them as ignored.

## Goals / Non-Goals

**Goals:** carry the eight recorded values as strings, only when the operator asks.

**Non-Goals:** no derived metrics (no "waste" or "savings"), no recommendation logic, no change to `totalCost`, no new `DryRun` status (the columns ride inside `extended_columns`, which `DryRun` already reports as conditional).

## Decisions

1. **Flag on the config, passed to `resultsFor`.** The server reads `s.cli.GetConfig().IncludeUsage` once per request and passes it down. A package-level switch would make tests share state.
2. **Column names use dotted families** (`efficiency.*`, `request.*`, `usage.*`) like the existing `annotation.<key>`, so a consumer can select a family by prefix. Units are in the name (`cpu_cores`, `ram_bytes`).
3. **Formatting is `strconv.FormatFloat(v, 'g', -1, 64)`**, the shortest exact representation, so a recorded `1` is `"1"` and `0.00188` stays `"0.00188"`.
4. **`efficiency.gpu` only with a GPU.** A row with `gpuCount` and `gpuHours` both zero reports `gpuEfficiency` 0, which would read as a wasted GPU.
5. **`__idle__` gets no usage columns.** Idle is unallocated capacity; request and usage averages are not defined for it.
6. **Env override is tri-state.** `OPENCOST_INCLUDE_USAGE` unset or empty leaves the file value; a set value overrides it, matching `OPENCOST_PROFILE` and `OPENCOST_CURRENCY`.

## Risks / Trade-offs

- [The keys are verified only against OpenCost recordings] → profile `kubecost` decodes them if present; the README says that is unverified.
- [More bytes per result when on] → off by default.
