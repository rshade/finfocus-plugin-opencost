# Design

## Context

See proposal.md for why these four behaviours are missing from the spec set.
The code and tests already implement them. This change records that behaviour.

## Goals / Non-Goals

**Goals:**

- Add `plugin-info`, `cost-errors`, `contract-sources`, and `kind-suite`.
- Leave the existing cost, query, budget, and health requirements unchanged.

**Non-Goals:**

- No new RPC behaviour and no production code change.
- Do not implement AllocatorService.
- Do not point the Kubecost profile at the live chart. `GET /model/allocation`
  on chart `kubecost` 3.3.0 was HTTP 404.
- Do not recreate kind cluster `oc-e2e` while recording the spec.

## Decisions

- Keep the four behaviours as new capabilities. Folding them into `actual-cost`
  or `budgets` would hide identity, errors, the source list, and the oracle.
- Cite the existing tests as the proof. The kind oracle stays behind `OC_E2E`
  and is not a default unit test.
- The contract source list stays a file list. Kubecost has no upstream module
  this plugin can marshal.

## Risks / Trade-offs

- [Kind cluster is down] → The spec describes the oracle. It does not require
  this commit to bring `oc-e2e` back.
- [Contract fixtures can drift from a live Kubecost API] → The source list says
  the bodies are not verified against live Kubecost. That is weaker than
  `make drift`.

## Migration Plan

Archive the change into `openspec/specs/`. No deploy or rollback step.
