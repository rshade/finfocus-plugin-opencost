## Context

`ActualCostResult` has no metadata map. Core reads `cost`, `source`,
`expires_at`, impact metrics, and focus currency. It does not read tags.
`finfocus-spec` `proto/finfocus/v1/focus.proto` defines `tags` as
user-defined key-value pairs and `extended_columns` as provider-specific
extensions.

Recorded pod rows in `testdata/opencost-real/allocation-pod-60m.json` have
`properties.labels` and `properties.controllerKind`. No recorded file has
`properties.annotations`.

## Goals / Non-Goals

**Goals:**

- The result for a recorded pod carries that row's labels and controller kind.
- An annotation on the same row is carried without changing the labels.
- A row with none of the three leaves `focus_record` unset.

**Non-Goals:**

- A complete FOCUS record. Currency stays unset until OC-3.8.
- Budget namespace globs. Issue #44's budget criteria are not this change.
- Editing `testdata/opencost-real/`.

## Decisions

- Labels use `focus_record.tags` because they are the row's user tags.
- Controller kind uses extended column `controllerKind`.
- Annotations use extended column `annotation.<key>` so they do not collide
  with `controllerKind` or with label keys.
- The focus record is partial. `ValidateActualCostResponse` does not require
  a valid FOCUS record. Setting a currency here would assume one.

## Risks / Trade-offs

- A host that runs `ValidateFocusRecord` on every actual-cost result will
  reject the partial record. Core's actual-cost adapter does not do that.
