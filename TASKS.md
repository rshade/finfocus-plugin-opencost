# finfocus-plugin-opencost v0.1.0 Plan

<!-- markdownlint-disable MD013 MD060 -->

**Status**: Planning. Rewritten 2026-10-03 and supersedes the 2026-09-30 kubecost plan. Phase 9 added 2026-10-04.
**Target version**: v0.1.0
**Go**: 1.27.1. **finfocus-spec**: v0.7.1 minimum.
**Process**: Superpowers for the run, then OpenSpec changes per task group (see section 6). Run prompt: `superpowers-prompt.md`.

## 1. Scope decision

The owner's decision on 2026-10-03: **all 21 open issues (the owner said "20"; #48 makes 21) are in scope for v0.1.0**. The plugin is the consolidated Kubecost and OpenCost plugin, named `opencost`, with one client and two endpoint profiles.

Evidence strength differs by profile and the report must say so:

| Profile | Backend in tests | Evidence |
| --- | --- | --- |
| `opencost` | The real OpenCost Helm chart on a kind cluster, with Prometheus and fixed custom pricing | Real responses and an independent price oracle (section 4) |
| `kubecost` | None. No free local backend is known | Contract fixtures written from Kubecost's documentation. Tag every such test `contract-fixture`; never claim it is verified against live Kubecost |

A task that only the `kubecost` profile can serve is delivered against fixtures and listed in the "Not delivered" register as "not verified against a live backend".

## 2. Current state (read 2026-10-03)

- The GitHub repo is `rshade/finfocus-plugin-opencost` (renamed). Its description still says "Kubecost Plugin for Pulumi".
- Code: about 1,000 non-test lines. `internal/allocation` is the client, `internal/server` the gRPC server, `cmd/finfocus-plugin-opencost` the entry point. There is **no finfocus-spec import**: the server uses local stub types and `RegisterService` is a no-op.
- The Go module path and manifest still use the old names.
- The client builds Kubecost's `/model/allocation` URL with Kubecost's `idle` and `accumulate` parameters. OpenCost's documented parameters are `includeIdle` and `shareIdle`, with no `accumulate`, and its path is `/allocation`.
- `GetProjectedCost` passes an empty resource id, so it averages the whole cluster.
- Filter parameters are built from a map, so the query string order changes between runs.
- OpenCost's documentation does not give the `/allocation` response envelope. **Do not code against an assumed shape.** Use the recorded real responses in `testdata/opencost-real/`.

## 3. Rules for this run

All rules in `superpowers-prompt.md` apply. The ones that shape this plan:

1. Phases 1 to 8: one conventional commit per task on the run branch, header ending in the task id. That run did not push. Phase 9 follows `.claude/commands/pick-issue.md`: one issue, one branch from `main`, one pull request. The owner merges.
2. `testdata/opencost-real/` (responses, `expected.json`, provenance) is owner-owned. You may not create, edit or regenerate them.
3. No hand-written fixture stands in for a real OpenCost response. Contract fixtures for Kubecost are allowed only under `testdata/kubecost-contract/` and are labelled.
4. Every task has a `Verify:` command and a break check.
5. Anything not delivered goes in the "Not delivered" register with the reason. Do not mark a task DONE for a stub.

## 4. Real backend and oracle (owner-owned, prepared before the run)

- **Backend**: kind cluster, `prometheus` chart with the OpenCost extra scrape config, `opencost` chart with `opencost.customPricing` set to fixed rates, UI disabled. The values file is `hack/kind/opencost-values.yaml`.
- **Workload**: namespace `oc-test`, a Deployment of 2 replicas, each requesting 500m CPU and 512Mi memory and idle (a `sleep`), so allocation is driven by requests.
- **Oracle**: expected cost = requests x fixed rate x window hours, computed from the manifests without calling OpenCost. Where OpenCost's allocation differs from this (it bills the larger of request and usage), the oracle file says why. Confidence: Medium until the break check below is run.
- **Break check**: change one rate in the values file; the oracle comparison must fail.
- **Recorded responses**: `testdata/opencost-real/*.json` are genuine `/allocation` responses captured from that cluster, with the capture command in `testdata/opencost-real/README.md`.

The oracle shares nothing with the plugin, but it does not prove that OpenCost's pricing is what a cloud bills. That claim is out of scope; custom pricing is a configured rate.

## 4a. Keeping the structs current (drift)

The plugin keeps its own small structs and does not import the OpenCost module. That module pulls in about 150 requirements including the Azure, AWS and GCS SDKs, and `govulncheck` reports 10 findings in it (none called). Three checks stand in for the import:

| Layer | Where | What it catches | Evidence |
| --- | --- | --- | --- |
| Field manifest | `ConsumedFields` and `IgnoredFields` in the plugin (OC-3.2) | A key added to or removed from the recorded response, a field used without being declared | Unit test, every run |
| Upstream schema | `test/drift` module (OC-3.9) | OpenCost changes its `Allocation` type: new, renamed or removed keys. Renovate bumps the pinned upstream version, the test runs on that pull request and fails | Spike on 2026-10-03: marshalling an upstream `opencost.Allocation` (core v1.121.3) gives exactly the 52 keys the real server sent, none extra on either side |
| Live server | Scheduled job on the latest chart (OC-5.6) | Behaviour the Go type does not show: envelope, error format, a new chart default | Compares key sets against the recorded fixtures |

What none of them catches: a changed meaning of an existing field, or a changed unit. The oracle comparison (section 4) covers cost arithmetic. When a drift check fails, the owner re-captures `testdata/opencost-real/`; the agent does not.

## 5. Tasks

Status values: `TODO`, `IN-PROGRESS`, `DONE`, `BLOCKED`, `BLOCKED-ON-INPUT`, `NOT-DELIVERED`, `SKIPPED`.

### Phase 0: Owner actions before the run

| Id | Task | Status |
| --- | --- | --- |
| OC-0.1 | Resolve the toolchain-baseline merge in the checkout and commit it on a branch from `origin/main` | TODO (owner) |
| OC-0.2 | Rewrite the body of issue #48 for the plugin repo (it still describes a core `plugins/opencost` module), rename the repo description | TODO (owner) |
| OC-0.3 | Decide the open design question in OC-3.6 (allocator or cost source) if you want it fixed before the run | TODO (owner, optional) |
| OC-0.4 | Place the owner-owned ground truth in `testdata/opencost-real/` (done 2026-10-03), then take baseline score `baseline` | TODO (PM) |

### Phase 1: Foundation

| Id | Task | Issues | Verify | Status |
| --- | --- | --- | --- | --- |
| OC-1.1 | Rename the Go module to `github.com/rshade/finfocus-plugin-opencost`, the cmd directory and binary to `finfocus-plugin-opencost`, and update the Makefile, goreleaser, manifest (`name: opencost`) and `internal/server` file names. The Kubecost client package becomes `internal/allocation` | #5 | `go build ./... && go vet ./...` and `rg for the retired plugin prefix` prints nothing; break check: reintroduce one old import path and the build fails | DONE (`go build ./... && go vet ./...` exit 0; retired-prefix search empty; break: old module import fails with "no required module provides package") |
| OC-1.2 | Initialise OpenSpec: pin `npm:@fission-ai/openspec` in `mise.toml`, run `mise exec -- openspec init`, write `openspec/config.yaml` project context | | `mise exec -- openspec validate --all --strict` | DONE (`mise exec -- openspec validate --all --strict` exit 0, "No items found"; break: the same command before the pin exits 1, "No version is set for shim: openspec") |
| OC-1.3 | Add the `.git/info/exclude` entries for `superpowers-prompt.md`, `superpowers-run-report.md` and `.superpowers/` if missing, and create `.superpowers/ledger.md` | | `git status --short` shows none of them | DONE (exclude entries already present; `git check-ignore` matches all three; `git status --short` lists none of them; break: a status line appears if `.superpowers/` is removed from exclude) |
| OC-1.4 | Confirm CI from the baseline: lint, test, commitlint, prose and release-please workflows parse; add the `RELEASE_PLEASE_TOKEN` note to the report as an owner action | #7 | run `actionlint` if installed; `gh workflow list` is read-only | DONE (`actionlint .github/workflows/*.yml` exit 0; `gh workflow list` exit 0 shows Commitlint, Prose, Release Please, GoReleaser, Test. Owner action: set `RELEASE_PLEASE_TOKEN` if release-please must trigger other workflows; the workflow falls back to `GITHUB_TOKEN`. Break: actionlint fails on a workflow with a bad `runs-on`) |

### Phase 2: Spec integration

| Id | Task | Issues | Verify | Status |
| --- | --- | --- | --- | --- |
| OC-2.1 | Add the `finfocus-spec` v0.7.1 dependency and the SDK; delete the local stub types; register the server through the SDK | #8 | `go build ./...` with no local stub types; a search for the old stub marker prints nothing | DONE (`go build ./...` and `go test -count=1 ./internal/server -run TestServerNameAndSDKRegistration` exit 0; stub-marker search empty outside this plan. Break: `Name` returning kubecost fails TestServerNameAndSDKRegistration) |
| OC-2.2 | Implement `Name`, `GetPluginInfo` (version from `pkg/version`, spec version `v`-prefixed) and declare capabilities with `WithCapabilities` | #5 | unit tests; `GetPluginInfo` returns a `v`-prefixed `spec_version`; break check: drop the prefix and the test fails | DONE (`go test -count=1 ./internal/server -run TestGetPluginInfoReportsSpecAndCapabilities` exit 0. Break: spec version `0.7.1` fails validation and the test) |
| OC-2.3 | Implement `Supports` for the Kubernetes resource types core sends (see OC-2.4), declining everything else with a reason | #3 | table-driven test over every supported and one unsupported type | DONE (`go test -count=1 ./internal/server -run TestSupportsKubernetesTypesCoreSends` exit 0. Break: renaming the pod type makes that subtest fail) |
| OC-2.4 | Read-only investigation: what `ResourceDescriptor` does core send for Kubernetes resources, and what does the `plugins/kubernetes` plugin send? Write `docs/resource-mapping.md` with the answer, citing file and line in the sibling `finfocus` checkout | | the document exists and every claim cites a file; `markdownlint` | DONE (`markdownlint docs/resource-mapping.md` exit 0. Each claim cites a sibling file and line. Break: not a code path; the document is the deliverable) |
| OC-2.5 | Structured logging with zerolog and `trace_id` from gRPC metadata (the key is defined in the spec SDK; do not invent a name) | #36, #9 | test that a request with a trace id logs it and one without does not panic | DONE (`go test -count=1 ./internal/server -run TestSupportsLogsTraceIDFromContext` exit 0. Break: skipping WithTrace, the log has no trace id and the test fails) |
| OC-2.6 | Error mapping: backend errors, timeouts and empty windows to the right gRPC codes and `NO_COST_DATA` style notes, never a silent zero | #9 | test per error path; break check: map a timeout to `OK` and the test fails | DONE (`go test -count=1 ./internal/server -run TestCostRPCErrorMapping` exit 0. Break: a timeout mapped to OK makes the timeout subtest fail) |

### Phase 3: Client and cost RPCs

| Id | Task | Issues | Verify | Status |
| --- | --- | --- | --- | --- |
| OC-3.1 | Endpoint profiles: `opencost` (`/allocation`, `includeIdle`, no token required) and `kubecost` (`/model/allocation`, Kubecost parameters, bearer token). One query builder, deterministic parameter order, profile chosen by config | | URL golden test per profile; run twice and assert identical strings | DONE (`go test -count=1 ./internal/allocation -run 'TestAllocationURLProfilesAreStable\|TestProfileSendsTokenOnlyForKubecost'` exit 0. Break: reversing filter key order fails the golden URL. OpenSpec `allocation-query` archived) |
| OC-3.2 | Decode the real OpenCost `/allocation` response from `testdata/opencost-real/` into typed structs (no `map[string]interface{}` for cost fields). Declare the wire fields in `internal/allocation/wirefields.go`: `ConsumedFields` (keys the plugin reads) and `IgnoredFields` (every other key OpenCost sends, each with a one-line reason). The two maps together must equal the keys in the recorded response | | test decodes every recorded file and asserts named totals; a second test asserts `ConsumedFields` plus `IgnoredFields` equals the key set of every recorded allocation; break check: delete one key from `IgnoredFields` and the test fails | DONE (`go test -count=1 ./internal/allocation -run 'TestDecodeRecordedAllocations\|TestWireFieldsCoverRecordedKeys'` exit 0. Recorded objects have 53 keys, and the idle file adds two. Break: deleting `window` from `IgnoredFields` fails the key test. OpenSpec `allocation-decode` archived) |
| OC-3.3 | `GetActualCost`: map the requested resource to a filter (namespace, controller, pod, node), query the window, return per-day results with the real totals | | oracle comparison for the `oc-test` namespace within the oracle tolerance (kind suite, OC-5.3) and a unit test per resource type | DONE (`go test -count=1 ./internal/server -run TestGetActualCostFiltersRecordedAllocations` exit 0. Break: returning every row makes the namespace subtest expect 1 and see 5. OpenSpec `actual-cost` archived. Kind oracle stays in OC-5.3) |
| OC-3.4 | `GetProjectedCost`: 30-day trailing average projected to a 730-hour month for the resource requested (not the whole cluster); say so in `billing_detail` | #4 | unit test with two different resources returning different values; break check: ignore the resource and the test fails | DONE (`go test -count=1 ./internal/server -run TestGetProjectedCostUsesRequestedResource` exit 0. Break: summing every row returns 222.46 instead of the requested namespace 72.69. OpenSpec `projected-cost` archived. Closes #4) |
| OC-3.5 | `EstimateCost` and `BatchCost`: one allocation query for a batch, results in request order, partial failure reported per item | | order and partial-failure tests | DONE (`go test -count=1 ./internal/server -run 'TestBatchCostKeepsOrderAndPartialFailure\|TestEstimateCostUsesRecordedNamespace\|TestBatchActualCostKeepsOrder'` exit 0. Break: reversing results expects prometheus-system and returns kube-system. OpenSpec `batch-cost` archived) |
| OC-3.6 | **Design decision**: should the plugin also implement `AllocatorService` (issue #48's pre-allocated rows)? The spec's conservation rule needs priced node costs that OpenCost does not take. Run `openspec explore`, write the finding and a recommendation in `docs/allocator-decision.md`. Implement only if the invariant can be met; otherwise record it as NOT-DELIVERED with the reason | #48 | the decision document names the spec invariant and cites `allocation.proto` | NOT-DELIVERED (`rg -n allocation.proto docs/allocator-decision.md` and `rg -n 1e-6 docs/allocator-decision.md` exit 0. Recorded namespace totals sum to 0.36187 against a priced node of 1.0, difference 0.63813, tolerance 1e-6. AllocatorService is not implemented. Issue #48 stays open. Break: removing the allocation.proto citation makes rg fail) |
| OC-3.7 | Namespace filtering and Kubecost metadata (labels, annotations, controller kind) carried through to results | #44 | unit test over recorded responses | DONE (`go test -count=1 ./internal/server -run TestGetActualCostCarriesRecordedMetadata` exit 0. Break: a nil focus record fails the test. OpenSpec `actual-cost` updated. Issue #44 stays open because its budget criteria are not this task) |
| OC-3.8 | Currency and cost-unit handling: units from the response, no assumed USD unless the pricing config says so | | test with a non-USD config value | DONE (`go test -count=1 ./internal/server -run 'TestCostCurrencyUsesNonUSDConfig\|TestCostCurrencyMissingIsAnError\|TestAllocationCurrencyOverridesConfig\|TestAllocationMixedCurrencyIsAnError\|TestBatchMissingObjectStaysNotFoundWithoutCurrency'` exit 0. Break: clearing the projected currency makes the non-USD test expect EUR and get an empty string. OpenSpec `actual-cost`, `projected-cost`, and `batch-cost` updated. The plugin does not assume USD) |
| OC-3.9 | **Drift check, offline** (section 4a): a separate test-only module `test/drift/` (own `go.mod`, not imported by the plugin) pins `github.com/opencost/opencost/core` at the version matching the chart under test. The test marshals an upstream `opencost.Allocation` and requires its key set to equal `ConsumedFields` plus `IgnoredFields`, and to equal the key set of the recorded real response. Add `make drift`. The plugin itself must not import the upstream module | #14 | `cd test/drift && go test -count=1 ./...`; `go list -m all` in the root module shows no `opencost/opencost`; break check: add a field to the root wire list that upstream lacks, and a key to `IgnoredFields` that upstream has, and each makes the test fail | DONE (`make drift` exit 0. Root `go list -m all` has no `github.com/opencost/opencost`. Break: `notARealField` in IgnoredFields fails because upstream does not send it, and adding `totalCost` to IgnoredFields fails because that key is already consumed. Recorded objects have 53 keys plus two idle-only keys. No OpenSpec change because this task does not change RPC behavior) |

### Phase 4: Hardening

| Id | Task | Issues | Verify | Status |
| --- | --- | --- | --- | --- |
| OC-4.1 | Security: HTTPS and `tlsSkipVerify` off by default, token read from env only and never logged, input validation on filter values (no injection into the filter grammar) | #13 | tests for a hostile namespace value and for a token absent from logs | DONE (`go test -count=1 ./internal/allocation ./internal/server -run 'TestTLSVerifyIsOnUnlessSkipped\|TestTokenIsEnvOnlyAndAbsentFromLogs\|TestHostileNamespaceIsRejected'` exit 0. Break: logging the Authorization header puts the token in the log, and accepting every filter value returns NotFound instead of InvalidArgument. OpenSpec `allocation-query` updated. Issue #13 stays open because rate limits and gRPC auth are not this task) |
| OC-4.2 | Performance: connection-pooled HTTP client with timeouts, a short TTL cache for repeated windows honouring `expires_at`, a request rate limit | #12 | benchmark and a test that two identical calls make one backend request | DONE (`go test -count=1 -bench=BenchmarkCachedAllocation -run 'TestRepeatedActualCostUsesOneRequest\|TestAllocationCacheExpires\|TestRateLimitRejectsTheNextDistinctQuery\|TestPooledClientReusesOneConnection' ./internal/allocation ./internal/server` exit 0. Break: skipping the cache returns two backend calls, and disabling the limiter returns NotFound instead of ResourceExhausted. OpenSpec `allocation-query` updated. Issue #12 stays open because LRU, metrics, and incoming gRPC limits are not this task) |
| OC-4.3 | Observability: `HealthCheck` that probes the backend, request counters and latency, trace propagation | #11 | health test against a stub server returning 200 then 500 | DONE (`go test -count=1 ./internal/server -run TestHealthCheckFollowsBackend` exit 0. Break: ignoring the backend status makes the 500 call return HTTP 200. OpenSpec `plugin-health` archived. Issue #11 stays open because Prometheus, OpenTelemetry, and alerting are not this task) |

### Phase 5: Real-backend suite

| Id | Task | Issues | Verify | Status |
| --- | --- | --- | --- | --- |
| OC-5.1 | `hack/kind/`: kind config, `opencost-values.yaml`, workload manifest, and `make e2e-kind-up` / `e2e-kind-down` scripts using only documented commands | #14 | `make e2e-kind-up` ends with the OpenCost API answering `/allocation/compute?window=60m` | DONE (`make e2e-kind-up` exit 0. HTTP 200 from `/allocation/compute?window=60m`. Charts prometheus 29.35.0 and opencost 2.5.32. Break: `/allocation/compute-missing` returns HTTP 404, so the script's HTTP 200 check fails. Issue #14 stays open for the later kind tasks) |
| OC-5.2 | Wait for data: poll `/allocation` until the `oc-test` namespace appears, with a documented timeout | #14 | the poll returns within the timeout on a fresh cluster | DONE (`make e2e-kind-wait` exit 0. Namespace oc-test had minutes 2.54454 inside the 900s timeout. Break: `OC_WAIT_TIMEOUT=12 hack/kind/e2e-wait.sh oc-missing` exits 1. Issue #14 stays open) |
| OC-5.3 | `make e2e-kind`: run the plugin binary against the cluster and compare `GetActualCost` for `oc-test` with the oracle | #14 | passes; break check: change one fixed rate and it fails | DONE (`make e2e-kind` exit 0. Namespace oc-test relative error 3.75e-5, minutes 1.86667, currency EUR. Break: CPU 9.0 returns relative 2.334 against the 2.0 oracle. Restored CPU 2.0 relative 0. Issue 14 stays open) |
| OC-5.4 | Plugin conformance: run the SDK conformance suite against the plugin over a real gRPC connection | #14 | suite passes | DONE (`OC_E2E=1 go test -tags conformance -count=1 -v ./test/conformance` exit 0. Passed 10, failed 0, skipped 2. GetPricingSpec returns per_hour at the observed hourly rate. Live unit price 3.749997 EUR. The suite aws/ec2 probe is rewritten to namespace oc-test. A direct ec2 call stays InvalidArgument. Break: a zero rate fails TestGetPricingSpecUsesObservedHourlyRate. OpenSpec pricing-spec archived. Issue 14 stays open) |
| OC-5.5 | GitHub Actions job on `ubuntu-latest` using `helm/kind-action` with `install_only: true` (the `finfocus` core `E2E (kind)` job in `ci.yml` does exactly this and passes on `ubuntu-latest`; the make target creates the cluster), label-gated or nightly, with cluster logs uploaded on failure | #14, #7 | `actionlint`; the first CI run is an owner action | DONE (`actionlint` exit 0. `.github/workflows/kind.yml` job E2E (kind) on ubuntu-latest uses helm/kind-action with install_only true, then `make e2e-kind`. It runs nightly, on workflow_dispatch, or when a pull request is labeled e2e-kind. Failure uploads kind export logs for oc-e2e. The first CI run is an owner action. Issues 14 and 7 stay open) |
| OC-5.6 | **Drift check, live** (section 4a): a scheduled CI job installs the **latest** published `opencost` chart (not the pinned one) on kind, queries `/allocation`, and compares the response key set and the error behaviour (HTTP 400, plain text) with the recorded fixtures. On a difference it fails and prints the added and removed keys. It never rewrites fixtures | #14 | `actionlint`; run the comparison locally against the pinned chart and see it pass, then against a copy of the fixture with one key removed and see it fail | DONE (`actionlint` exit 0. `.github/workflows/live-drift.yml` installs the latest opencost chart on cluster oc-drift. Against pinned chart 2.5.32, `go run ./hack/drift/livekeyset` exited 0: added none, removed none, 55 keys, and the HTTP 400 bodies matched the recorded plain text. A fixture copy with `cpuEfficiency` removed exited 1 and printed `removed: cpuEfficiency`. Fixtures were not rewritten. The first CI run is an owner action. Issue 14 stays open) |

### Phase 6: Packaging and docs

| Id | Task | Issues | Verify | Status |
| --- | --- | --- | --- | --- |
| OC-6.1 | Dockerfile for the plugin binary, built in CI, smoke-run in the kind suite | #10 | `docker build` and `docker run ... --help` | DONE (`docker build -t finfocus-plugin-opencost:dev .` exit 0. `docker run --rm finfocus-plugin-opencost:dev --help` prints `-version` and `-port` and exits 0. Break: `ENTRYPOINT ["/missing"]` fails `TestDockerfileBuildsPluginBinary` and `docker run` exits 127 with `exec: "/missing"`. Restored entrypoint, help exits 0 again. `test.yml` builds the image. `kind.yml` builds it and smoke-runs `--help` before e2e, accepting exit 0 or 2. GoReleaser uses `Dockerfile.goreleaser` for the prebuilt binary. `goreleaser check` still exits 2 on deprecations that already fail on the previous config. actionlint exit 0. Issue 10 stays open) |
| OC-6.2 | End-to-end example: a Pulumi stack with Kubernetes resources and the commands to see costs through finfocus | #6 | the example's `pulumi preview --json` is run for real and the output is stored as a fixture | DONE (`pulumi preview --non-interactive --stack dev --json` exit 0 on Pulumi 3.264.0 with a file backend. `examples/kubernetes/preview.json` stores that output. Steps create Namespace `oc-example`, Deployment `fixed`, Service `fixed`, and DaemonSet `node-agent`. The diagnostic home path is `~`. Break: clearing the DaemonSet `newState.type` fails `TestKubernetesExamplePreviewFixture`. Restored preview passes. Issue 6 stays open) |
| OC-6.3 | Documentation: README, configuration reference, troubleshooting, deployment guide, API notes for both profiles | #15 | `markdownlint` and `vale` pass; every config key appears in the reference | DONE (`TestReadmeDocumentsEveryConfigKey` checks every `Config` YAML key in the README and in `config.example.yaml`. `markdownlint README.md` exit 0. `vale README.md` exit 0. The reference covers profiles `opencost` (`GET /allocation`) and `kubecost` (`GET /model/allocation`), deployment, and troubleshooting. Break: renaming `` `cacheTTL` `` fails that test. Restored key passes. Issue 15 stays open) |
| OC-6.4 | Update `CLAUDE.md`, `ROADMAP.md` (if present) and `plugin.manifest.json` for the new name and capabilities | | `rg -i kubecost` finds only the profile name and history | DONE (`rg -i kubecost CLAUDE.md plugin.manifest.json` matches only profile `kubecost` and `KUBECOST_` names. No ROADMAP.md. Manifest capabilities are PROJECTED_COSTS, ACTUAL_COSTS, PRICING_SPEC, ESTIMATE_COST, and BATCH_COST. Break: "Kubecost plugin" fails TestDocsLimitKubecostToProfileAndHistory. Restored text passes) |

### Phase 7: Kubecost profile

| Id | Task | Issues | Verify | Status |
| --- | --- | --- | --- | --- |
| OC-7.1 | Cost prediction (`/model/prediction/speccost`) behind the `kubecost` profile, exposed through `EstimateCost`. Read `kubecost/kubectl-cost` `pkg/query/prediction_speccost.go` first (Apache-2.0) and cite the file and commit in each fixture | #21 | contract-fixture test; label `contract-fixture`; report as "not verified against live Kubecost" | DONE (`TestEstimateCostKubecostUsesSpecCostContract` label contract-fixture. Profile kubecost posts attributes to `/model/prediction/speccost` and `cost_monthly` is `costAfter.totalMonthlyRate`. Fixture cites `pkg/query/prediction_speccost.go` commit `1f45d3085b2ffa84758bfa8131ea8b7784cd8ed1`. Break: returning `costBefore` differs by 15.5. Restored `costAfter` passes. Not verified against live Kubecost. Issue 21 stays open) |
| OC-7.2 | `GetBudgets` from Kubecost namespace budgets | #38 | contract-fixture test | DONE (`TestGetBudgetsUsesNamespaceBudgetContract` label contract-fixture. Profile kubecost sends GET `/model/budgets` and returns the namespace rule. `spendLimit` is `amount.limit`, currency is pricing config EUR, `currentSpend` is status when requested. The cluster rule is omitted. Fixture cites the Budget API page checked 2026-10-03. Break: returning the cluster rule fails the length check. Restored namespace filter passes. Not verified against live Kubecost. Issue 38 stays open) |
| OC-7.3 | Budget mapping and namespace filtering tests | #42 | tests pass over fixtures | DONE (`TestGetBudgetsFiltersNamespaceAndMapsInterval` over `budgets-namespaces.json`. A namespace tag returns that weekly rule with period, intervalDay, kind, windowStart, and percentage taken from the fixture. An unknown namespace returns none. No tag returns both namespace rules and not the cluster rule. Break: ignoring the tag returns 2 budgets. Restored filter passes. Not verified against live Kubecost. Issue 42 stays open) |
| OC-7.4 | Budget health aggregation and multi-provider summary tests | #43 | tests pass over fixtures | DONE (`TestGetBudgetsAggregatesHealthAndSummary` over `budgets-health.json`. `include_status` sets health from `currentSpend`, `spendLimit`, and the lowest action percentage: over the limit is exceeded, equal to the limit is critical, at or above that percentage is warning, and below it is ok. The summary counts those four states for the filtered namespace budgets, and the four counts equal total. Critical stays its own count. A namespace tag counts only that rule. An unknown namespace counts zero. Without `include_status` the summary is absent. The cluster rule is omitted. Break: zeroing the summary expects total 4 and gets 0. Restored counts pass. Fixture cites the Budget API page checked 2026-10-03. Not verified against live Kubecost. Issue 43 stays open) |
| OC-7.5 | Optional spike: can the Kubecost chart run on kind without a paid token? Record the answer with evidence; do not build on it unless it works | | a documented yes or no with the command output | DONE (`docs/kubecost-kind-spike.md`, answer yes. Cluster `oc-kubecost`, not `oc-e2e`. Chart `cost-analyzer` 2.9.7 template exits 1: Missing global federated-store. Chart `kubecost` 3.3.0 `helm upgrade --install` exit 0 with only `global.clusterId` set. Nine pods `1/1 Running`. Frontend HTTP 200. `GET /model/allocation` HTTP 404. Cluster deleted afterward. The plugin stays on the contract fixtures. Break: a second `answer: no` line fails `TestKubecostKindSpikeRecordsVerdict`) |
| OC-7.6 | Kubecost drift note: Kubecost has no upstream module we can marshal, so its guard is weaker. Add a `testdata/kubecost-contract/SOURCES.md` listing each fixture's source URL or `kubectl-cost` commit and the date checked, and a test that fails when a fixture lacks an entry | #21 | the test fails when an entry is removed | DONE (`TestKubecostContractSourcesListsEveryFixture` reads every `testdata/kubecost-contract/*.json` and its `.citation`. `SOURCES.md` must name that file and repeat the citation source URL or commit and the checked date. Break: removing the `budgets-health.json` row fails with no entry. Restored list passes. Not verified against live Kubecost. Issue 21 stays open) |

### Phase 8: Closeout

| Id | Task | Status |
| --- | --- | --- |
| OC-8.1 | Disposition of #18 (Dependency Dashboard): not code. Record "obsolete, Renovate handles it; owner closes" in the register | DONE (`TestDependencyDashboardIsOwnerClose` reads section 8. The register says obsolete, Renovate handles it; owner closes. `renovate.json` extends `config:recommended` and leaves the dashboard on. Break: removing that sentence fails the test. Issue 18 stays open) |
| OC-8.2 | Convert any remaining Phase 1 to 7 behaviour not yet captured into `openspec/specs/` capabilities and archive the changes | DONE (OpenSpec change `remaining-behaviour` archived as `2026-10-03-remaining-behaviour`. New capabilities: `plugin-info`, `cost-errors`, `contract-sources`, and `kind-suite`. Earlier specs already cover allocation, cost, budgets, health, and pricing. Packaging and the Pulumi example stay outside the spec set. No production code change. Break: replacing `no upstream module` in the contract-sources spec fails `TestRemainingBehaviourSpecsExist`. `mise exec -- openspec validate --all --strict` exit 0) |
| OC-8.3 | Write `superpowers-run-report.md` with the issue-by-issue table, the "Not delivered" register and the spec-gap log | DONE (`superpowers-run-report.md` is git-excluded and lists every task, the 21 issues, and the not-delivered register. Section 7 commands in the acceptance log exit 0, including `make govulncheck` and `make drift`. `make e2e-kind` evidence is the session log `E2E_EXIT:0` on cluster `oc-e2e`, charts prometheus 29.35.0 and opencost 2.5.32. Two live rounds of `Supports` and `GetActualCost` for namespace `oc-test` returned cost 0.49333 EUR, and `Supports` for `kubernetes:core/v1:Service` is false. The spec-gap log has no filed issue. Break: not a code path; the report is the deliverable) |

### Phase 9: Review follow-ups

Added 2026-10-04. These rows are the queue. `.claude/commands/pick-issue.md` is the procedure.

A row is eligible when its status is exactly `TODO` and every task id in Depends is `DONE` or `IN-PROGRESS` on the open stack. An empty Depends is eligible. `BLOCKED` and `BLOCKED-ON-INPUT` are not eligible. Phase 0 statuses `TODO (owner)` are not `TODO`.

Take the first eligible row, one issue, one pull request, then stop. The owner merges. When the previous pull request is still open, branch from it and set the new pull request's base to that branch. A Depends id that is `IN-PROGRESS` on that stack is eligible. Release pull request #76 stays open until OC-9.1 and OC-9.2 have landed and Release Please shows version 0.1.0.

On a row that says to stop and ask, stop. Do not invent the missing input. After asking, set that row to `BLOCKED-ON-INPUT`. The owner sets it back to `TODO` once the input is in the tree. For #66, #67, and #68, finish the agent slice, stop before secrets and branch protection, leave the GitHub issue open, and record the owner step in the ledger.

`testdata/opencost-real/` stays read-only. A captured controller body goes in `testdata/opencost-real-controller/` after the owner captures it.

Launch prompt:

```text
Follow .claude/commands/pick-issue.md. The plan of record is TASKS.md Phase 9.

Take the first row whose status is exactly TODO and whose Depends task ids are all DONE. One issue, one small pull request, then stop.

Start from a clean main that matches origin/main, or from the previous open issue branch when you are stacking. Push only the issue branch. Set the new pull request base to that branch. The owner merges. Leave release pull request #76 open.

Do not close #18. Do not take #63 unless this prompt names it. Close an issue yourself only when main already satisfies it, and cite the commit and the command.

On #64 (OC-9.3), use the captured responses in `testdata/opencost-real-controller/`. `testdata/opencost-real/` is read-only. Do not hand-write a response body. If that controller directory is missing, stop and ask.

For #66, #67, and #68, do the agent slice and stop before secrets and branch protection. Leave those issues open.

Skip BLOCKED and BLOCKED-ON-INPUT rows.
```

| Id | Issue | Task | Depends | Verify | Status |
| --- | --- | --- | --- | --- | --- |
| OC-9.1 | [#77](https://github.com/rshade/finfocus-plugin-opencost/issues/77) | Direct. Add `"initial-version": "0.1.0"` on the `.` package in `release-please-config.json`. After Release Please rebuilds, the release PR title and manifest say `0.1.0`. If they still say `1.0.0`, stop and ask the owner to land a `Release-As: 0.1.0` commit. Leave #76 unmerged. | | `jq -r '.packages["."]["initial-version"]' release-please-config.json` prints `0.1.0`. `gh pr view 76 --json title` shows `0.1.0` after the rebuild. Break: delete the key and jq prints `null`. | DONE (`initial-version` is `0.1.0` on main at `88784d1`. Release PR #76 titles `0.1.0`, its manifest is `0.1.0`, and `CHANGELOG.md` has a `0.1.0` heading. No tags exist. Leave #76 unmerged. #77 stays open until the owner cuts that release.) |
| OC-9.2 | [#78](https://github.com/rshade/finfocus-plugin-opencost/issues/78) | Direct. Add `CHANGELOG.md` to the `ignores` list in `.markdownlint-cli2.jsonc`. Leave `MD012` on for every other file. Add one `CLAUDE.md` line that `CHANGELOG.md` is generated and is not hand-edited. A later #63 ignore list keeps this entry and the narrowed OpenSpec globs. | | Check out the release branch's `CHANGELOG.md` and run `npx markdownlint-cli2 "**/*.md"`; exit 0. Break: two consecutive blank lines in `README.md` still report `MD012`. | IN-PROGRESS (`CHANGELOG.md` is in the markdownlint ignores. `MD012` still reports on `README.md`. Pull request #81 targets main and its checks passed.) |
| OC-9.3 | [#64](https://github.com/rshade/finfocus-plugin-opencost/issues/64) | OpenSpec. Controller queries filter on `controllerName`; aggregate stays `controller`. First action: ask the owner to capture the HTTP 500 body for `controller:"fixed"` and the row for `controllerName:"fixed"` into `testdata/opencost-real-controller/`, with date, chart versions, and the command in a README. Set this row to `BLOCKED-ON-INPUT` and stop. The owner sets it back to `TODO` once that directory is in the tree. Do not change the filter and do not write a body before then. | OC-9.1, OC-9.2 | After the capture: `go test -count=1 ./internal/server/ -run 'TestGetActualCost\|TestIdentity'` and `mise exec -- openspec validate --all --strict`. Break: a filter key `controller` fails the grammar test. Then `make e2e-kind` for Deployment `fixed`. | DONE (Filter key is `controllerName`. Aggregate stays `namespace,controller`. Restoring the key `controller` fails `TestAllocationFilterKeysAreAcceptedFields`. Kind oracle matched Deployment `fixed` at CPU 2.0, diverged at 9.0, and matched again after restore. `mise exec -- openspec validate --all --strict` exit 0.) |
| OC-9.4 | [#73](https://github.com/rshade/finfocus-plugin-opencost/issues/73) | OpenSpec. Return YAML and environment parse errors with the file path or variable name. `Config.Validate` runs before `Serve`: `baseUrl` is an http(s) URL with a host, profile is empty or `opencost` or `kubecost`, and a set currency is three upper-case letters. A missing config file stays tolerated. Do not enable `KnownFields` unless the invocation says so. No token in an error or a log. README states the real precedence. | OC-9.3 | `go test -count=1 ./internal/allocation/ ./cmd/finfocus-plugin-opencost/ -run 'Config\|Validate'`. The binary with an empty `KUBECOST_BASE_URL` exits 1. Break: restore `_ = yaml.Unmarshal` and the syntax test fails. | TODO |
| OC-9.5 | [#70](https://github.com/rshade/finfocus-plugin-opencost/issues/70) | OpenSpec for the gRPC behaviour. No new proto field. Skip a budget rule that fails validation and return the rest, with one WARN naming the rule id and the reason. HTTP 401 is `Unauthenticated` and HTTP 403 is `PermissionDenied`, through a typed status error. The message names the check for `KUBECOST_API_TOKEN` and does not contain the token. | OC-9.4 | `go test -count=1 ./internal/server/ ./internal/allocation/ -run 'GetBudgets\|Auth\|Backend\|Token'`. Break: a `daily` interval fails the whole call again. | TODO |
| OC-9.6 | [#74](https://github.com/rshade/finfocus-plugin-opencost/issues/74) | Direct. Replace the empty `main` tests with a built-binary check of `-version` and `-version-full`. Delete dead `Client.Allocation`, `EnhancedAllocation`, `ConvertToSimpleResponse`, `Point`, and `Response`, and delete unreferenced `testdata/sample_*.json`. Keep `TestKubecostContractSourcesListsEveryFixture`. Keep or remove each other doc-content test with a one-line reason. `internal/server` statement coverage stays at or above 84.2%. | OC-9.5 | `go test -count=1 ./cmd/finfocus-plugin-opencost/ ./internal/allocation/ ./internal/server/`. `rg 'EnhancedAllocation\|ConvertToSimpleResponse\|allocation\.Point' --glob '*.go'` prints nothing. Break: `-version` printing nothing fails the binary test. | TODO |
| OC-9.7 | [#72](https://github.com/rshade/finfocus-plugin-opencost/issues/72) | OpenSpec. Reject a non-loopback `http` base URL unless `allowInsecureHttp` is set. `http://localhost` still starts. Log one WARN per startup when `tlsSkipVerify` is on, and none when it is off. Add `caCertFile` for a private CA. Document the new keys in the README and `config.example.yaml`. gRPC authentication stays a non-goal. | OC-9.6 | `go test -count=1 ./internal/allocation/ -run 'TestReadmeDocumentsEveryConfigKey\|TestTLS\|TestInsecure\|TestCA'`. Break: accepting `http://example.com` fails the reject test. | TODO |
| OC-9.8 | [#65](https://github.com/rshade/finfocus-plugin-opencost/issues/65) | OpenSpec once core sends `GetActualCostRequest.resource`. Bump `finfocus-spec` to v0.7.4 and, when the descriptor is set, resolve actual cost from it. Attributes win over tags. An unset descriptor keeps today's tag path. Core at the time of the issue does not populate the field. | OC-9.7, core sends `resource` on actual-cost calls | `go test -count=1 ./internal/server/ ./test/conformance/ -run 'Plugin\|GetActualCost\|RPCCorrectness_GetActualCostWithResource'`. Break: ignoring `resource` fails the descriptor cases. | BLOCKED |
| OC-9.9 | [#66](https://github.com/rshade/finfocus-plugin-opencost/issues/66) | Direct. The owner already replaced the token; release PR #76 opened. Add the finfocus-spec style probe so an invalid `RELEASE_PLEASE_TOKEN` warns and falls back, and document permissions and rotation. Do not read or set the secret. Leave the issue open until `release.yml` has run after the first release. | OC-9.7 | `actionlint .github/workflows/release-please.yml`. A branch run with the secret name blanked shows the warning. Break: the job fails with `Bad credentials` again. | TODO |
| OC-9.10 | [#67](https://github.com/rshade/finfocus-plugin-opencost/issues/67) | Direct for the workflow edit and one dispatch of `kind.yml` and `live-drift.yml` when the token can start them. Record both run URLs on the issue. Run `E2E (kind)` on pull requests that touch `internal/**`, `test/e2e/**`, or `hack/kind/**`. A live key-set difference opens or updates one issue. Stop before adding a required check. A break branch that edits a fixture is not merged. | OC-9.9 | `actionlint .github/workflows/kind.yml .github/workflows/live-drift.yml`. `gh run list --workflow kind.yml --limit 1` shows a completed run. Break: one removed fixture key on a throwaway branch opens the drift issue. | TODO |
| OC-9.11 | [#68](https://github.com/rshade/finfocus-plugin-opencost/issues/68) | Direct. Run `go test -race -covermode=atomic -coverprofile=coverage.out ./...` in `test.yml`, write the total to the step summary, and upload `coverage.out`. No Go matrix. Badges only for workflows that are green on `main`. Stop before the branch-protection ruleset. That ruleset is the owner's, and `Prose` waits until OC-9.2 has landed. | OC-9.10 | `actionlint .github/workflows/test.yml` and `make test-race` exit 0. Break: a `matrix` key in `test.yml` fails the empty-matrix check. | TODO |
| OC-9.12 | [#69](https://github.com/rshade/finfocus-plugin-opencost/issues/69) | Direct once the owner has reviewed the expectation inputs. Extend the kind oracle to controller, pod, and node. Discover the pod name at run time. Do not derive a number from the plugin. Do not edit `testdata/opencost-real/` until the owner has reviewed the values. The controller filter is OC-9.3. | OC-9.3, owner-reviewed expectations | `make e2e-kind` logs an oracle match for each kind. Break: CPU `9.0` diverges for every kind, then restore `2.0`. | BLOCKED-ON-INPUT |
| OC-9.13 | [#71](https://github.com/rshade/finfocus-plugin-opencost/issues/71) | The kubecost profile needs a Kubecost the agent can call. The kind spike's `GET /model/allocation` was HTTP 404. A live capture uses owner-redacted responses. Label the profile experimental only when the owner chooses that in the invocation. | OC-9.7, a reachable Kubecost | The decision is written in `docs/kubecost-kind-spike.md`. Contract tests still pass. Break: removing the decision line fails that test. | BLOCKED-ON-INPUT |
| OC-9.14 | [#75](https://github.com/rshade/finfocus-plugin-opencost/issues/75) | Replace `google.golang.org/grpc v1.86.0-dev` when a stable tag contains the GO-2026-6443 fix. Until that tag exists, leave the require line as it is. Land the pin note with the bump, then delete the note. | a stable grpc tag at or above the fix | `govulncheck ./...` prints `No vulnerabilities found.` and `go list -m google.golang.org/grpc` shows the stable tag. Break: `v1.84.0` reports GO-2026-6443, then restore `go.mod` and `go.sum`. | BLOCKED |

Outside this phase: [#63](https://github.com/rshade/finfocus-plugin-opencost/issues/63) (OpenSpec process and its CI gaps) stays open and is not a row. [#76](https://github.com/rshade/finfocus-plugin-opencost/pull/76) is the release pull request, not an issue to implement. [#18](https://github.com/rshade/finfocus-plugin-opencost/issues/18) stays in section 8.

## 6. OpenSpec conversion

OpenSpec describes behaviour that exists. At the start there is none, so:

- Phase 1 and Phase 2 are plain tasks (a rename and a dependency do not need a spec).
- From Phase 3 on, each behaviour-changing task or task group is one OpenSpec change. Target capabilities: `allocation-client`, `cost-rpcs`, `endpoint-profiles`, `security`, `kind-suite`, `kubecost-profile`.
- Each change is proposed, applied, verified and archived. The archive lands in the same commit as the code. After the run, `openspec/specs/` is the description of current behaviour and `TASKS.md` is history.

## 7. Issue index (21 open on 2026-10-03)

| Issue | Title | Task |
| --- | --- | --- |
| [#3](https://github.com/rshade/finfocus-plugin-opencost/issues/3) | Supports() implementation | OC-2.3 |
| [#4](https://github.com/rshade/finfocus-plugin-opencost/issues/4) | Projected cost, 30-day average | OC-3.4 |
| [#5](https://github.com/rshade/finfocus-plugin-opencost/issues/5) | Plugin manifest and install path | OC-1.1, OC-2.2 |
| [#6](https://github.com/rshade/finfocus-plugin-opencost/issues/6) | End-to-end example Pulumi stack | OC-6.2 |
| [#7](https://github.com/rshade/finfocus-plugin-opencost/issues/7) | CI/CD pipeline | OC-1.4, OC-5.5 |
| [#8](https://github.com/rshade/finfocus-plugin-opencost/issues/8) | Protocol buffer integration | OC-2.1 |
| [#9](https://github.com/rshade/finfocus-plugin-opencost/issues/9) | Error handling and structured logging | OC-2.5, OC-2.6 |
| [#10](https://github.com/rshade/finfocus-plugin-opencost/issues/10) | Docker containerization | OC-6.1 |
| [#11](https://github.com/rshade/finfocus-plugin-opencost/issues/11) | Monitoring and observability | OC-4.3 |
| [#12](https://github.com/rshade/finfocus-plugin-opencost/issues/12) | Performance optimisation | OC-4.2 |
| [#13](https://github.com/rshade/finfocus-plugin-opencost/issues/13) | Security enhancements | OC-4.1 |
| [#14](https://github.com/rshade/finfocus-plugin-opencost/issues/14) | Integration testing suite | OC-5.1 to OC-5.5 |
| [#15](https://github.com/rshade/finfocus-plugin-opencost/issues/15) | Documentation | OC-6.3 |
| [#18](https://github.com/rshade/finfocus-plugin-opencost/issues/18) | Dependency Dashboard | OC-8.1 |
| [#21](https://github.com/rshade/finfocus-plugin-opencost/issues/21) | Cost Prediction API | OC-7.1 |
| [#36](https://github.com/rshade/finfocus-plugin-opencost/issues/36) | zerolog and trace propagation | OC-2.5 |
| [#38](https://github.com/rshade/finfocus-plugin-opencost/issues/38) | GetBudgets | OC-7.2 |
| [#42](https://github.com/rshade/finfocus-plugin-opencost/issues/42) | Budget mapping tests | OC-7.3 |
| [#43](https://github.com/rshade/finfocus-plugin-opencost/issues/43) | Budget health tests | OC-7.4 |
| [#44](https://github.com/rshade/finfocus-plugin-opencost/issues/44) | Namespace filtering and metadata | OC-3.7 |
| [#48](https://github.com/rshade/finfocus-plugin-opencost/issues/48) | OpenCost plugin returning pre-allocated rows | OC-3.6 and the whole plan |

## 8. Not delivered (the agent fills this in)

| Item | Reason | Owner action |
| --- | --- | --- |
| [#48](https://github.com/rshade/finfocus-plugin-opencost/issues/48) AllocatorService | OpenCost totals are not a split of host priced node costs, so they fail conservation in `allocation.proto`. See `docs/allocator-decision.md`. | Leave the issue open. Do not claim `ALLOCATION`. |
| [#18](https://github.com/rshade/finfocus-plugin-opencost/issues/18) Dependency Dashboard | obsolete, Renovate handles it; owner closes. `renovate.json` extends `config:recommended` and leaves the dashboard on. | Owner closes #18. |

## 9. Spec-gap log (the agent fills this in)

| Gap | Genericity argument | Issue filed |
| --- | --- | --- |
| | | |
