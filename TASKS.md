# finfocus-plugin-opencost v0.1.0 Plan

<!-- markdownlint-disable MD013 MD060 -->

**Status**: Planning. Rewritten 2026-10-03 and supersedes the 2026-09-30 kubecost plan.
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

1. One conventional commit per task on the run branch, header ending in the task id. Never push. GitHub is read-only.
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
| OC-3.9 | **Drift check, offline** (section 4a): a separate test-only module `test/drift/` (own `go.mod`, not imported by the plugin) pins `github.com/opencost/opencost/core` at the version matching the chart under test. The test marshals an upstream `opencost.Allocation` and requires its key set to equal `ConsumedFields` plus `IgnoredFields`, and to equal the key set of the recorded real response. Add `make drift`. The plugin itself must not import the upstream module | #14 | `cd test/drift && go test -count=1 ./...`; `go list -m all` in the root module shows no `opencost/opencost`; break check: add a field to the root wire list that upstream lacks, and a key to `IgnoredFields` that upstream has, and each makes the test fail | TODO |

### Phase 4: Hardening

| Id | Task | Issues | Verify | Status |
| --- | --- | --- | --- | --- |
| OC-4.1 | Security: HTTPS and `tlsSkipVerify` off by default, token read from env only and never logged, input validation on filter values (no injection into the filter grammar) | #13 | tests for a hostile namespace value and for a token absent from logs | TODO |
| OC-4.2 | Performance: connection-pooled HTTP client with timeouts, a short TTL cache for repeated windows honouring `expires_at`, a request rate limit | #12 | benchmark and a test that two identical calls make one backend request | TODO |
| OC-4.3 | Observability: `HealthCheck` that probes the backend, request counters and latency, trace propagation | #11 | health test against a stub server returning 200 then 500 | TODO |

### Phase 5: Real-backend suite

| Id | Task | Issues | Verify | Status |
| --- | --- | --- | --- | --- |
| OC-5.1 | `hack/kind/`: kind config, `opencost-values.yaml`, workload manifest, and `make e2e-kind-up` / `e2e-kind-down` scripts using only documented commands | #14 | `make e2e-kind-up` ends with the OpenCost API answering `/allocation/compute?window=60m` | TODO |
| OC-5.2 | Wait for data: poll `/allocation` until the `oc-test` namespace appears, with a documented timeout | #14 | the poll returns within the timeout on a fresh cluster | TODO |
| OC-5.3 | `make e2e-kind`: run the plugin binary against the cluster and compare `GetActualCost` for `oc-test` with the oracle | #14 | passes; break check: change one fixed rate and it fails | TODO |
| OC-5.4 | Plugin conformance: run the SDK conformance suite against the plugin over a real gRPC connection | #14 | suite passes | TODO |
| OC-5.5 | GitHub Actions job on `ubuntu-latest` using `helm/kind-action` with `install_only: true` (the `finfocus` core `E2E (kind)` job in `ci.yml` does exactly this and passes on `ubuntu-latest`; the make target creates the cluster), label-gated or nightly, with cluster logs uploaded on failure | #14, #7 | `actionlint`; the first CI run is an owner action | TODO |
| OC-5.6 | **Drift check, live** (section 4a): a scheduled CI job installs the **latest** published `opencost` chart (not the pinned one) on kind, queries `/allocation`, and compares the response key set and the error behaviour (HTTP 400, plain text) with the recorded fixtures. On a difference it fails and prints the added and removed keys. It never rewrites fixtures | #14 | `actionlint`; run the comparison locally against the pinned chart and see it pass, then against a copy of the fixture with one key removed and see it fail | TODO |

### Phase 6: Packaging and docs

| Id | Task | Issues | Verify | Status |
| --- | --- | --- | --- | --- |
| OC-6.1 | Dockerfile for the plugin binary, built in CI, smoke-run in the kind suite | #10 | `docker build` and `docker run ... --help` | TODO |
| OC-6.2 | End-to-end example: a Pulumi stack with Kubernetes resources and the commands to see costs through finfocus | #6 | the example's `pulumi preview --json` is run for real and the output is stored as a fixture | TODO |
| OC-6.3 | Documentation: README, configuration reference, troubleshooting, deployment guide, API notes for both profiles | #15 | `markdownlint` and `vale` pass; every config key appears in the reference | TODO |
| OC-6.4 | Update `CLAUDE.md`, `ROADMAP.md` (if present) and `plugin.manifest.json` for the new name and capabilities | | `rg -i kubecost` finds only the profile name and history | TODO |

### Phase 7: Kubecost profile

| Id | Task | Issues | Verify | Status |
| --- | --- | --- | --- | --- |
| OC-7.1 | Cost prediction (`/model/prediction/speccost`) behind the `kubecost` profile, exposed through `EstimateCost`. Read `kubecost/kubectl-cost` `pkg/query/prediction_speccost.go` first (Apache-2.0) and cite the file and commit in each fixture | #21 | contract-fixture test; label `contract-fixture`; report as "not verified against live Kubecost" | TODO |
| OC-7.2 | `GetBudgets` from Kubecost namespace budgets | #38 | contract-fixture test | TODO |
| OC-7.3 | Budget mapping and namespace filtering tests | #42 | tests pass over fixtures | TODO |
| OC-7.4 | Budget health aggregation and multi-provider summary tests | #43 | tests pass over fixtures | TODO |
| OC-7.5 | Optional spike: can the Kubecost chart run on kind without a paid token? Record the answer with evidence; do not build on it unless it works | | a documented yes or no with the command output | TODO |
| OC-7.6 | Kubecost drift note: Kubecost has no upstream module we can marshal, so its guard is weaker. Add a `testdata/kubecost-contract/SOURCES.md` listing each fixture's source URL or `kubectl-cost` commit and the date checked, and a test that fails when a fixture lacks an entry | #21 | the test fails when an entry is removed | TODO |

### Phase 8: Closeout

| Id | Task | Status |
| --- | --- | --- |
| OC-8.1 | Disposition of #18 (Dependency Dashboard): not code. Record "obsolete, Renovate handles it; owner closes" in the register | TODO |
| OC-8.2 | Convert any remaining Phase 1 to 7 behaviour not yet captured into `openspec/specs/` capabilities and archive the changes | TODO |
| OC-8.3 | Write `superpowers-run-report.md` with the issue-by-issue table, the "Not delivered" register and the spec-gap log | TODO |

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

## 9. Spec-gap log (the agent fills this in)

| Gap | Genericity argument | Issue filed |
| --- | --- | --- |
| | | |
