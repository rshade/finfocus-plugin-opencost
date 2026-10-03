## Context

Recorded OpenCost responses are envelopes of name-keyed allocation objects.
Most objects have 53 keys. The idle capture adds two keys. Error captures are
plain text. The plan text said 52 keys. The files are the source of truth.

## Goals / Non-Goals

**Goals:**

- Typed totals for the fields the plugin bills.
- A declared reason for every other key.
- Plain-text HTTP errors stay readable.

**Non-Goals:**

- Editing `testdata/opencost-real/`.
- Importing `github.com/opencost/opencost`.

## Decisions

- Cost fields stay on `AllocationEntry`. `rawAllocationOnly` is ignored.
- The key test allows objects to omit keys. The idle-only keys are still
  declared, and every declared key must appear at least once.
- Status 400 returns before JSON decoding.

## Risks / Trade-offs

- A future OpenCost chart can add a key and fail the wire-field test. That is
  the drift signal. Do not delete a declared key to silence it.

## Migration Plan

- None. The decoder replaces the inline JSON decode in `GetDetailedAllocation`.

## Open Questions

- None.
