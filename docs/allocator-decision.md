# AllocatorService decision

Recommendation: do not implement `AllocatorService`. Status for OC-3.6 is NOT-DELIVERED. This plugin stays a CostSource. It does not report `PLUGIN_CAPABILITY_ALLOCATION`. Issue #48 is closed.

`openspec list` shows no active change. This task does not change plugin behavior, so no OpenSpec change is archived with it.

## The conservation invariant

`AllocatorService` is defined in `finfocus-spec` v0.7.1 at `proto/finfocus/v1/allocation.proto`. Hosts price nodes through a CostSource, then call `Allocate`. The allocator does the split. The host does not (`allocation.proto:24-28`).

Every response must satisfy the invariants at `allocation.proto:30-44`. Hosts reject a violation. The Go SDK exposes the checks as `pluginsdk.CheckConservation` (`sdk/go/pluginsdk/allocator.go:46-53`) and `pluginsdk.ValidateAllocateResponse` (`sdk/go/pluginsdk/allocator.go:56-63`).

Conservation, `allocation.proto:33-35`: the sum of `rows.total_cost` equals the sum of `priced.cost` where `priced=true`, within a relative tolerance of `1e-6` and an absolute floor of `1e-9`. The SDK names those `DefaultConservationEpsilon` (`sdk/go/testing/allocation.go:34`) and `ConservationAbsoluteFloor` (`sdk/go/testing/allocation.go:37`). The allowed difference is `max(1e-6 * |expected|, 1e-9)`.

The same comment requires more than the sum:

- Every row has `total_cost = cpu_cost + mem_cost` within that tolerance, except subject kind `__cluster__` (`allocation.proto:36-37`, repeated at `:167-168`).
- No row carries a negative `cpu_cost`, `mem_cost`, or `total_cost` (`allocation.proto:38`).
- Every successfully priced node (`priced=true` and `resource.tags.kind="node"`) has exactly one `__idle__` row whose `node` subject equals that node's `resource.id`, even when idle cost is zero (`allocation.proto:39-41`).
- Every row carries the resolved currency (`allocation.proto:42` and `:171`).

`PricedResource.currency` (`allocation.proto:74-77`) resolves one currency across `priced=true` entries: the single distinct non-empty value, or `"USD"` when all of those currencies are empty. That `"USD"` fallback applies only inside this contract. A `priced=false` entry counts 0 toward conservation (`allocation.proto:80`).

OpenCost does not accept `AllocateRequest.priced`. Its allocation totals are its own prices.

## Recorded rows fail the invariant

`testdata/opencost-real/allocation-namespace-60m.json` is the namespace aggregate used by the cost RPCs. Those five namespace totals are:

| Namespace | totalCost | cpuCost + ramCost | Other positive component |
| --- | --- | --- | --- |
| kube-system | 0.1177 | 0.1177 | none |
| local-path-storage | 0.00049 | 0.00048 | none |
| oc-test | 0.1406 | 0.1406 | none |
| opencost | 0.0035 | 0.00351 | none |
| prometheus-system | 0.09958 | 0.00565 | pvCost 0.09392 |

The namespace totals sum to 0.36187. Against one host priced node of cost 1.0, the difference is 0.63813. The tolerance is `max(1e-6 * 1.0, 1e-9) = 1e-6`. `CheckConservation` fails. Any other host price that is not 0.36187 fails the same way. OpenCost does not take that host price as an input, so the plugin cannot make the sums match by querying harder.

Two row-shape rules fail on the same file:

- `local-path-storage` has `totalCost` 0.00049 and `cpuCost + ramCost` 0.00048. The difference 0.00001 is above the absolute floor `1e-9`. `opencost` is 0.0035 against 0.00351, the same gap.
- `prometheus-system` carries `pvCost` 0.09392. A workload row may hold only `cpu_cost` and `mem_cost`. Putting persistent-volume cost on a workload row violates `allocation.proto:36-37`. A `__cluster__` row may hold cost outside those two fields, and it still has to conserve against `priced.cost`.

No file under `testdata/opencost-real/` contains a `currency` key. Allocation rows must carry a resolved currency (`allocation.proto:171`).

The only node name in those recordings is `oc-spike-control-plane`. The OpenCost profile sets `includeIdle=false` (`internal/allocation/allocation.go:100`). The Kubecost profile sets `idle=false` (`internal/allocation/allocation.go:105`). Neither cost query returns the `__idle__` row `allocation.proto:39-41` requires for every priced node.

## Why a rescaled allocator is not this issue

Scaling OpenCost's shares onto `priced.cost` could make the numeric sum match. That is a different product from issue #48. The issue asks for pre-allocated OpenCost rows. Those rows are OpenCost's prices, not a split of a host node price. Rescaling would also drop or hide `pvCost` and the other components a workload row cannot carry (`gpu`, network, load balancer, shared, external). The plugin would be inventing a split the backend does not compute.

## What this plugin ships

`GetActualCost`, `GetProjectedCost`, `EstimateCost`, and `BatchCost` stay cost-source methods. There is no `Allocate` method and no `PLUGIN_CAPABILITY_ALLOCATION` in `internal/server/info.go`. Issue #48 is closed. A separate allocator that takes priced node costs would be a new issue.
