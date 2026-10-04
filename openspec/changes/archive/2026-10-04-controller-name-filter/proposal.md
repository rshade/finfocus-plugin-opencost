# Proposal

## Why

OpenCost 2.5.32 rejects the filter field `controller` with HTTP 500
`expect filter field`. The field it accepts for that object is
`controllerName`. The plugin still sends `controller`, so a controller
cost query fails on a live cluster.

## What Changes

- Controller allocation queries filter on `controllerName`.
- The aggregate stays `namespace,controller`.
- An HTTP 500 whose body names an invalid filter is `Unavailable` and
  includes that body. It is not `NO_COST_DATA`.
- The kind oracle also prices Deployment `fixed` in namespace `oc-test`.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `actual-cost`: The controller scenario filter is `controllerName`.
- `cost-errors`: An invalid-filter HTTP 500 names the rejected field.

## Impact

- `internal/server` filter construction and the actual-cost tests.
- `testdata/opencost-real-controller/` is the owner capture. `testdata/opencost-real/` stays unchanged.
- Kind e2e oracle in `test/e2e`.
- Kubecost has no recording of this field. The shared filter builder uses the OpenCost name. That profile stays unverified against a live Kubecost.
