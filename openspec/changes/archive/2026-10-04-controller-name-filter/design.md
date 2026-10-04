# Design

## Filter key

`resourceRef.aggregate` keeps the kind `controller`. `resourceRef.filter` uses
a separate key, `controllerName`, for that kind. Namespace, pod, and node keys
stay equal to the kind.

Filter keys the plugin may send are `namespace`, `pod`, `node`,
`controllerName`, and `cluster`. A key named `controller` fails the grammar
test. `cluster` is allowed because the profile tests already send it. This
change does not add a cluster filter.

## Captured bodies

`testdata/opencost-real-controller/error-controller-filter.json` is the HTTP
500 for `controller:"fixed"`. `allocation-controllername-60m.json` is the HTTP
200 row for `controllerName:"fixed"`. Tests read those files. The directory
`testdata/opencost-real/` stays unchanged.

## Errors

`DecodeAllocationBody` already returns an error that includes an HTTP 500
body. `mapBackendError` maps that error to `Unavailable`. The new test serves
the captured body and requires the rejected field name in the message.

## Kind oracle

Deployment `fixed` is the only workload in namespace `oc-test`. The controller
check uses the same rate formula as the namespace oracle. Minutes come from a
direct allocation query filtered by `controllerName`. The plugin cost is
`GetActualCost` for `controller/oc-test/fixed`. The CPU `9.0` break applies to
that controller total as well.

## Kubecost

One filter builder serves both profiles. No Kubecost recording shows the
controller field name, so this change does not claim the kubecost profile
accepts `controllerName`.
