# Design

## Context

`projectedResponse` in `internal/server/project.go` writes one constant `billing_detail` for every kind. Namespace aggregation sums its workloads, so only the namespace row overlaps with other rows of the same stack. The plugin sees one object per call and cannot know what else the stack prices.

## Goals / Non-Goals

**Goals:**
- Tell the user, in the response itself, that a namespace figure contains its workloads.
- Make the shipped example give a correct total.

**Non-Goals:**
- No de-duplication across rows, which needs a parent field in `finfocus-spec` and core.
- No change to `Supports`, so a namespace-only stack can still be priced.

## Decisions

- Pass the resolved `resourceRef` into `projectedResponse` and choose the detail string from `ref.kind`. Both callers, `GetProjectedCost` and the projected `BatchCost` item, already hold the ref or can get it. Alternative: append the sentence in each caller. Rejected: two copies drift.
- The example names the namespace as a literal string and has no Namespace resource, which matches a real stack that deploys into an existing namespace.

## Risks / Trade-offs

- Pulumi YAML needs the namespace to exist at apply time, but the example only renders manifests, so nothing is applied.
- The regenerated `preview.json` comes from the installed Pulumi, not the 3.264.0 named in the README; the README records the version used.
