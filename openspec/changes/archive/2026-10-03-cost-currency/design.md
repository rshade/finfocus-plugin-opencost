## Context

`GetProjectedCostResponse.currency` and `EstimateCostResponse.currency` are
proto fields. `ActualCostResult` has no currency field. Core reads actual
currency from `FocusCostRecord.billing_currency`, then pricing currency, and
otherwise defaults to USD. Recorded allocations have no `currency` key.
`provenance/oc-values.yaml` sets custom pricing rates and no currency code.

## Goals / Non-Goals

**Goals:**

- One currency for a response: the body when it names a single value, else
  pricing config.
- `OPENCOST_CURRENCY` overrides a YAML `currency`, the same way
  `OPENCOST_PROFILE` overrides a YAML profile.
- Missing currency is `FailedPrecondition` and the message does not say USD.
- An empty match stays `NotFound` and does not ask for a currency.
- A batch item with rows and no currency is a per-item error. The RPC succeeds.
- A batch item with no rows stays `NotFound` even when currency is unset.

**Non-Goals:**

- Adding `currency` to `ConsumedFields` or `IgnoredFields`. The key is absent
  from the recordings, so the wire map must not list it.
- A currency catalog in `GetPricingSpec`. That RPC stays unimplemented.
- Editing `testdata/opencost-real/` or `provenance/oc-values.yaml`.
- Assuming USD when the operator did not set it.

## Decisions

- Envelope `currency` and entry `currency` must agree. Two values is
  `FailedPrecondition`.
- Actual rows always get `billing_currency`, including a row whose labels,
  annotations, and controller kind are empty. That creates a focus record.
- Currency is resolved only after the RPC has a non-empty match, so empty
  data stays `NotFound`.
- `unit_price` stays the observed hourly rate. The recordings do not carry a
  separate unit string.

## Risks / Trade-offs

- A host that still expects an empty currency string now gets `FailedPrecondition`
  until config or the body sets one. That is the point of refusing an assumed USD.
