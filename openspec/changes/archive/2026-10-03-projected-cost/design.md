## Context

Core calls `GetProjectedCost` with a `ResourceDescriptor`. The Pulumi type is
copied through, and `id` is the provider resource id. A namespace id is the
namespace name. A pod or controller id is `namespace/name`.

Recorded envelopes have no currency. OC-3.8 decides currency.

## Goals / Non-Goals

**Goals:**

- Two resources return two monthly costs.
- Monthly cost is `totalCost / (minutes / 60) * 730`.
- `billing_detail` says this is a 30-day trailing average projected to a
  730-hour month.
- `cost_breakdown` sums to `cost_per_month`.

**Non-Goals:**

- Assuming USD.
- Dry-run responses.
- Editing `testdata/opencost-real/`.

## Decisions

- The query window is `30d`. The month uses the minutes in the response, so a
  shorter captured sample still projects from its own rate.
- Component costs are scaled so they sum to the projected total. Recorded
  component sums are not always equal to `totalCost`.
- An empty resource id is `InvalidArgument`. The plugin does not average the
  cluster.

## Risks / Trade-offs

- A 60-minute recording is not 30 days of data. The live oracle in OC-5.3
  checks actual cost. This unit test checks the projection of recorded rows.

## Migration Plan

- None. `GetProjectedCost` replaces the unimplemented stub.

## Open Questions

- None.
