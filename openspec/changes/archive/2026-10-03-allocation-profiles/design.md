## Context

`BuildAllocationURL` always sets the path to `/model/allocation` and joins filter
map keys in iteration order. Recorded OpenCost calls use `/allocation` with
`includeIdle`. Kubecost calls use `/model/allocation` with `idle`, `accumulate`,
and `shareIdle`, plus a bearer token.

## Goals / Non-Goals

**Goals:**

- One builder, two profiles, selected by config.
- Identical query strings across calls.
- OpenCost requests succeed without a token. Kubecost requests send the token.

**Non-Goals:**

- Decoding the allocation body. That is a later task.
- Moving the token to environment-only. That is a later task.

## Decisions

- Empty profile means `opencost`, because the kind oracle is OpenCost.
- `url.Values.Encode` sorts parameter names. Filter keys are sorted before the
  filter value is built, because that value is one string.
- OpenCost sends `includeIdle` and `shareIdle`. Kubecost sends `idle`,
  `accumulate`, and `shareIdle`. Defaults stay false, matching the recorded
  namespace query.
- An unknown profile is an error.

## Risks / Trade-offs

- Callers that relied on the Kubecost path without setting a profile now hit
  `/allocation`. Tests that want the old path set `profile: kubecost`.

## Migration Plan

- Set `OPENCOST_PROFILE=kubecost` or `profile: kubecost` for a Kubecost backend.
- Leave the profile unset for OpenCost.

## Open Questions

- None.
