## Context

Recorded OpenCost envelopes list many objects. `GetActualCost` receives a
resource id such as `namespace/oc-test`. The OpenCost profile queries
`GET /allocation`. There is no node-aggregated recording, so a node request
is checked against rows whose `properties.node` matches.

## Goals / Non-Goals

**Goals:**

- One result per matching allocation, in step order, including a real zero.
- Filter strings with sorted keys, and aggregates that name the object kind.

**Non-Goals:**

- Projection, estimates, currency, and metadata. Those are later tasks.
- Editing `testdata/opencost-real/`.
- Importing `github.com/opencost/opencost`.

## Decisions

- `namespace/<name>` filters `namespace:"<name>"` and aggregates `namespace`.
  A row matches `properties.namespace`, or the map key when that property is
  empty.
- `controller/<namespace>/<name>` filters `controller` and `namespace` and
  aggregates `namespace,controller`. The controller name is `properties.controller`,
  not the `deployment:` prefix in the allocation key.
- `pod/<namespace>/<name>` filters `namespace` and `pod` and aggregates
  `namespace,pod`.
- `node/<name>` filters `node:"<name>"` and aggregates `node`. It matches
  `properties.node`.
- Cost is the typed `totalCost` of each matching row.

## Risks / Trade-offs

- A broad envelope can contain extra rows. The server still selects matches
  after the backend filter.

## Migration Plan

- None. `GetActualCost` replaces the unfiltered enhanced allocation path.

## Open Questions

- None.
