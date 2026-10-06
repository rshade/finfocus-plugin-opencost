# Tasks

## 1. Descriptor resolution

- [x] 1.1 When `GetActualCostRequest.resource` is set, resolve with `refForDescriptor`. Attributes replace tags. An unset descriptor keeps the resource id and tag path. A missing row names the descriptor id. Verify: `go test -count=1 ./internal/server/ ./test/conformance/ -run 'Plugin|GetActualCost|RPCCorrectness_GetActualCostWithResource'`. Break: ignoring `resource` fails the descriptor cases.

## 2. Close out

- [x] 2.1 `mise exec -- openspec validate actual-cost-resource --strict` exits 0, then archive the change in this commit. `go list -m github.com/rshade/finfocus-spec` stays on v0.7.5.
