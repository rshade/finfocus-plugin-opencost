# `finfocus-plugin-opencost` Strategic Roadmap

## Vision

Serve accurate, explainable Kubernetes cost to FinFocus from OpenCost, and
prove it against a real backend on every change. The boundaries are in
[CONTEXT.md](CONTEXT.md). The work queue, the task ledger, and the engineering
reference follow the boundary safeguards at the end of this file.

Status on 2026-11-12: v0.1.2 is released with binaries. The cost RPCs, the
kind oracle, the drift checks, and the OpenSpec process are in place. Every
Immediate Focus and #69 and #71 wait on an owner decision or input. The
issues filed on 2026-11-12 (#133 to #140) are code work and are not blocked.

## Immediate Focus (v0.1.x)

- [ ] #68 `ci`: race and coverage reporting, protect main, README badges [S]
  Shipped: `-race` and a coverage summary in `Test`, and the `Test` badge. Left:
  the branch-protection ruleset (owner) and a coverage-threshold decision. The
  current total is 76.6%.
- [ ] #67 `ci(kind)`: run `kind.yml` and `live-drift.yml` once, make E2E (kind) a required check [M]
  Shipped: the workflow edits and the drift-issue reporter. Left: find out why
  the nightly live-drift job failed in each of its last five runs, dispatch both
  workflows once with a token that can start them, and decide whether
  `E2E (kind)` is required. A required check must be skipped at job level, not
  workflow level, or it stays pending forever.

## Near-Term Vision (v0.2)

- [ ] #69 `test(e2e)`: extend the live oracle to controller, pod, and node requests [M]
  Needs the owner to review the expected values first. The pod, node, and
  cluster filter keys are unproven until this lands.
- [ ] #71 `test`: verify profile `kubecost` against a real backend or label it experimental [M]
  Needs a reachable backend, or the owner's decision to label the profile
  experimental. The kind spike's `GET /model/allocation` returned HTTP 404.
- [ ] #133 `feat(server)`: report only the capabilities that are enabled [S]
  `GetPluginInfo` advertises `BUDGETS` on profile `opencost`, where
  `GetBudgets` returns `Unimplemented`. Core gates RPCs on that list.
- [ ] #134 `feat(server)`: implement `DryRun` so `finfocus plugin inspect` works [S]
  Core's `plugin inspect` calls `DryRun`, which the plugin does not implement.
- [ ] #135 `ci(drift)`: run the cost oracle against the latest OpenCost chart nightly [M]
  Adds the chart-version and cluster env vars. Depends on the live-drift fix in
  #131, which has merged but has not yet had a passing run.

## Future Vision (Long-Term)

- [ ] #136 `feat(server)`: efficiency and request/usage on actual cost behind `includeUsage` [M]
  OpenCost returns this data and the plugin ignores it today.
- [ ] #137 `spike`: how OpenCost allocation maps to FOCUS 1.3 split-cost columns
  Timebox 1d. The deliverable is a decision, not code.
- [ ] #138 `feat(server)`: `GetRecommendations` (right-sizing) behind a flag [L]
  Blocked on #133 (capabilities follow config) and #136 (usage columns).
- [ ] #139 `docs`: the spec version in README and CLAUDE.md is stale, and nothing checks it [S]
- [ ] #140 `test(drift)`: cover the live key-set comparison, now 11.6% [S]
  Land it before the first real live-drift run is trusted.
- Profile `kubecost` checked against a live backend in CI, only if #71 finds
  one that can run without a product key.

## Completed Milestones

### 2026-Q4

- [x] #63 `process`: OpenSpec as the change process, with CI gates. Closed 2026-11-12. [M]
- [x] #64 `client`: controller filter on `controllerName`. Closed 2026-11-12. [S]
- [x] #65 `server`: read `GetActualCostRequest.resource`. Closed 2026-11-12. [M]
- [x] #66 `release`: release-token probe with a safe fallback. Closed 2026-11-12. [S]
- [x] #72 `security`: reject public HTTP, CA file, TLS warning. Closed 2026-11-12. [M]
- [x] #75 `deps`: `gRPC` on stable v1.85.0. Closed 2026-11-12. [S]
- [x] #78 `lint`: ignore the generated changelog. Closed 2026-11-12. [S]
- [x] #70 `server`: skip invalid budget rules, and map 401 and 403. Closed 2026-10-06. [M]
- [x] #73 `config`: surface config errors, validate at startup. Closed 2026-10-06. [M]
- [x] #74 `test`: remove vacuous tests and dead client code. Closed 2026-10-06. [S]
- [x] #77 `release`: first release is v0.1.0. Closed 2026-10-06. [S]
- [x] #3 `server`: `Supports` for Pulumi resource tokens. Closed 2026-10-04. [M]
- [x] #4 `server`: 30-day projected monthly cost. Closed 2026-10-04. [M]
- [x] #5 `release`: plugin manifest and install path. Closed 2026-10-04. [M]
- [x] #6 `docs`: example Pulumi stack. Closed 2026-10-04. [M]
- [x] #7 `ci`: GitHub Actions workflows. Closed 2026-10-04. [M]
- [x] #8 `server`: finfocus-spec SDK integration. Closed 2026-10-04. [M]
- [x] #9 `server`: structured errors and logging. Closed 2026-10-04. [M]
- [x] #10 `release`: Dockerfile and image build. Closed 2026-10-04. [M]
- [x] #11 `server`: health check and request logging. Closed 2026-10-04. [M]
- [x] #12 `client`: cache, rate limit, connection pool. Closed 2026-10-04. [M]
- [x] #13 `security`: token handling and input validation. Closed 2026-10-04. [M]
- [x] #14 `test`: kind oracle, drift checks, conformance suite. Closed 2026-10-04. [L]
- [x] #15 `docs`: README and resource mapping. Closed 2026-10-04. [M]
- [x] #21 `server`: estimate cost through spec cost. Closed 2026-10-04. [M]
- [x] #36 `server`: zerolog with trace propagation. Closed 2026-10-04. [S]
- [x] #38 `server`: `GetBudgets`. Closed 2026-10-04. [M]
- [x] #42 `test`: budget mapping and filter tests. Closed 2026-10-04. [S]
- [x] #43 `test`: budget health and summary tests. Closed 2026-10-04. [S]
- [x] #44 `server`: namespace filter and metadata. Closed 2026-10-04. [S]
- [x] #48 `server`: OpenCost plugin; `AllocatorService` not delivered. Closed 2026-10-04. [L]

### 2025-Q3

- [x] #2 `client`: allocation query for a date range. Closed 2025-08-16. [M]
- [x] #1 `deps`: fix the Renovate configuration. Closed 2025-08-14. [S]
- [x] #16 `deps`: fix the Renovate configuration. Closed 2025-08-15. [S]
- [x] #17 `deps`: fix the Renovate configuration. Closed 2025-08-15. [S]

## Boundary Safeguards

From [CONTEXT.md](CONTEXT.md). An item that crosses one is delegated, not
scheduled.

- No `AllocatorService` and no `ALLOCATION` capability.
- No gRPC authentication, no cloud price catalog, no persisted state.
- No import of the OpenCost module into the plugin module.
- No assumed currency and no invented cost.
- No hand-written stand-in for an OpenCost response.
- No metrics, tracing, or alerting service.

## Work Queue (Phase 9)

This is the queue `.claude/commands/pick-issue.md` works from. Status values:
`TODO`, `IN-PROGRESS`, `DONE`, `BLOCKED`, `BLOCKED-ON-INPUT`, `NOT-DELIVERED`,
`SKIPPED`. A row is eligible when its status is exactly `TODO` and every id in
Depends is `DONE` or `IN-PROGRESS` on the open stack. `TODO (owner)`, `BLOCKED`,
and `BLOCKED-ON-INPUT` are not eligible. The owner sets a blocked row back to
`TODO` once the input is in the tree.

**No row is eligible today.** Every row below waits on the owner.

| Id | Issue | Task | Depends | Status |
| --- | --- | --- | --- | --- |
| OC-9.12 | #69 | Extend the kind oracle to controller, pod, and node. Discover the pod name at run time and derive no number from the plugin. `testdata/opencost-real/` stays read-only until the owner reviews the expected values. Break check: CPU `9.0` diverges for every kind, then restore `2.0`. | OC-9.3, owner-reviewed expectations | BLOCKED-ON-INPUT |
| OC-9.13 | #71 | Profile `kubecost` needs a backend the agent can call. The kind spike's `GET /model/allocation` returned HTTP 404 (`docs/kubecost-kind-spike.md`). A live capture uses owner-redacted responses. Label the profile experimental only when the owner chooses that in the invocation. | a reachable backend | BLOCKED-ON-INPUT |
| OC-9.17 | to file | `finfocus plugin inspect opencost <type>` fails with "plugin 'opencost' does not support inspection (capability discovery not implemented)" (seen against v0.1.2). azure-public supports it. The owner confirms the gap is wanted and an issue is filed, then implement what `plugin inspect` needs and document it. Break check: removing the handler fails with the same message. | OC-9.8 | BLOCKED-ON-INPUT |

Agent slices already done, with an owner step left:

- OC-9.10 (#67): the workflow edits and the drift-issue reporter are in. Left:
  dispatch `kind.yml` and `live-drift.yml` once with a token that can start
  them (the earlier dispatch returned HTTP 403), diagnose the nightly drift
  failures, and decide on a required check.
- OC-9.11 (#68): `-race`, the coverage summary, and the `Test` badge are in.
  Left: the branch-protection ruleset and a coverage-threshold decision.

Launch prompt:

```text
Follow .claude/commands/pick-issue.md. The plan of record is the Work Queue in ROADMAP.md.

Take the first row whose status is exactly TODO and whose Depends ids are all DONE or IN-PROGRESS on the open stack. One issue, one small pull request, then stop.

Start from a clean main that matches origin/main, or from the previous open issue branch when you are stacking. Push only the issue branch. Set the new pull request base to that branch. The owner merges.

Do not close #18. Close an issue yourself only when main already satisfies it, and cite the commit and the command.

For #66, #67, and #68, do the agent slice and stop before secrets and branch protection.

Skip BLOCKED and BLOCKED-ON-INPUT rows.
```

## Owner Decisions

These rows were written before the first run. Three look overtaken by later
work. They are the owner's to close out, and no agent marks them done.

| Id | Task | Status |
| --- | --- | --- |
| OC-0.1 | Resolve the toolchain-baseline merge in the checkout and commit it on a branch from `origin/main`. | TODO (owner) |
| OC-0.2 | Rewrite the body of issue #48 for the plugin repo and rename the repo description. #48 is closed. | TODO (owner) |
| OC-0.3 | Decide the allocator-or-cost-source question in OC-3.6. `docs/allocator-decision.md` settled it as no `AllocatorService`. | TODO (owner, optional) |
| OC-0.4 | Take the baseline score after placing the ground truth in `testdata/opencost-real/` (placed 2026-10-03). | TODO (PM) |

## Task Ledger

Finished tasks, kept so the ids in `docs/allocator-decision.md`, the Work Queue,
and the OpenSpec archive still resolve. The command output and break checks
for each row are in the git history of the retired `TASKS.md`.

| Id | What shipped | Issue |
| --- | --- | --- |
| OC-1.1 | Module, command directory, binary, and manifest renamed to `finfocus-plugin-opencost` | #5 |
| OC-1.2 | OpenSpec initialised and pinned in `mise.toml` | |
| OC-1.3 | Local ledger and exclude entries | |
| OC-1.4 | CI workflows confirmed to parse | #7 |
| OC-2.1 | `finfocus-spec` SDK dependency, local stub types deleted | #8 |
| OC-2.2 | `Name`, `GetPluginInfo`, and declared capabilities | #5 |
| OC-2.3 | `Supports` for the Kubernetes types core sends | #3 |
| OC-2.4 | Investigation of the resource descriptor core sends (`docs/resource-mapping.md`) | |
| OC-2.5 | zerolog with `trace_id` from the SDK | #36, #9 |
| OC-2.6 | Backend errors mapped to gRPC codes, never a silent zero | #9 |
| OC-3.1 | Endpoint profiles `opencost` and `kubecost` | |
| OC-3.2 | Typed decode of the recorded `/allocation` responses, with `ConsumedFields` and `IgnoredFields` | |
| OC-3.3 | `GetActualCost` filters for namespace, controller, pod, and node | |
| OC-3.4 | `GetProjectedCost` as a 30-day average projected to a 730-hour month | #4 |
| OC-3.5 | `EstimateCost` and `BatchCost` | |
| OC-3.6 | `AllocatorService` decision: not delivered (see Not Delivered) | #48 |
| OC-3.7 | Namespace filtering and metadata on results | #44 |
| OC-3.8 | Currency and unit handling, never an assumed USD | |
| OC-3.9 | Offline drift check in the `test/drift` module | #14 |
| OC-4.1 | Security: TLS on by default, env-only token, filter-value validation | #13 |
| OC-4.2 | Pooled client, TTL cache, rate limit | #12 |
| OC-4.3 | Health check, request counters, latency, trace propagation | #11 |
| OC-5.1 | `hack/kind/` cluster, values, workload, and up and down scripts | #14 |
| OC-5.2 | Wait for `oc-test` allocation data | #14 |
| OC-5.3 | `make e2e-kind` oracle comparison | #14 |
| OC-5.4 | SDK conformance suite over a real gRPC connection | #14 |
| OC-5.5 | `E2E (kind)` GitHub Actions job | #14, #7 |
| OC-5.6 | Live drift job against the latest published chart | #14 |
| OC-6.1 | Dockerfile, built and smoke-run in CI | #10 |
| OC-6.2 | Example Pulumi stack | #6 |
| OC-6.3 | README, configuration reference, and docs | #15 |
| OC-6.4 | `CLAUDE.md` and manifest naming | |
| OC-7.1 | Cost prediction through `POST /model/prediction/speccost` | #21 |
| OC-7.2 | `GetBudgets` for the `kubecost` profile | #38 |
| OC-7.3 | Budget mapping and namespace filter tests | #42 |
| OC-7.4 | Budget health and summary tests | #43 |
| OC-7.5 | Spike: the kubecost chart on kind (`docs/kubecost-kind-spike.md`) | |
| OC-7.6 | `testdata/kubecost-contract/SOURCES.md` | #21 |
| OC-8.1 | Disposition of #18 recorded | |
| OC-8.2 | Remaining behaviour captured as OpenSpec capabilities | |
| OC-8.3 | Run report | |
| OC-9.1 | `initial-version` `0.1.0` in the release-please config | #77 |
| OC-9.2 | `CHANGELOG.md` ignored by markdownlint | #78 |
| OC-9.3 | Controller filter on `controllerName` | #64 |
| OC-9.4 | Config parse errors name the file or variable, startup validation | #73 |
| OC-9.5 | Budget rule skipping and 401 and 403 mapping | #70 |
| OC-9.6 | Vacuous tests and dead client code removed | #74 |
| OC-9.7 | Plain HTTP rejected, TLS warning, `caCertFile` | #72 |
| OC-9.8 | `GetActualCostRequest.resource` read | #65 |
| OC-9.9 | Release-token probe with a safe fallback | #66 |
| OC-9.10 | Kind and live-drift workflows, drift-issue reporter (agent slice) | #67 |
| OC-9.11 | `-race`, coverage summary, `Test` badge (agent slice) | #68 |
| OC-9.14 | `google.golang.org/grpc` v1.85.0 stable pin | #75 |
| OC-9.15 | `include-component-in-tag: false` and the release rules; v0.1.2 is the first release with assets | |
| OC-9.16 | Controller-filter pull request rebased onto `main` | |

## Not Delivered

| Item | Reason | Owner action |
| --- | --- | --- |
| `AllocatorService` (OC-3.6, #48) | OpenCost totals are not a split of host priced node costs, so they fail conservation in `allocation.proto`. See `docs/allocator-decision.md`. | None. #48 is closed. |
| Profile `kubecost` against a live backend | No reachable backend. It is delivered against labelled contract fixtures. | Decide #71. |
| Oracle coverage for controller, pod, and node (OC-9.12) | The expected values need the owner's review. The `pod`, `node`, and `cluster` filter keys stay unproven until then. | Review the values, then #69. |

The spec-gap log is empty: no gap in `finfocus-spec` has needed an issue.

## Scope and Evidence by Profile

One client and two endpoint profiles. Evidence strength differs, and every
report must say so:

| Profile | Backend in tests | Evidence |
| --- | --- | --- |
| `opencost` | The real OpenCost Helm chart on a kind cluster, with Prometheus and fixed custom pricing | Real responses and an independent price oracle |
| `kubecost` | None. No free local backend is known | Contract fixtures written from the kubecost documentation, labelled `contract-fixture`. Never claim they are verified against a live backend |

A task that only the `kubecost` profile can serve is delivered against fixtures
and listed under Not Delivered as not verified against a live backend.

## Rules for Agent Work

1. One issue, one branch from `main`, one pull request. The owner merges.
   `.claude/commands/pick-issue.md` is the procedure. After a stacked base
   merges (squash), rebase onto `main` with `git rebase --onto origin/main <last
   merged commit>`. A stacked pull request with no `Test` check is not ready to
   merge, because `test.yml` runs only on pull requests into `main`.
2. `testdata/opencost-real/` (responses, `expected.json`, provenance) is
   owner-owned. Never create, edit, or regenerate it.
3. No hand-written fixture stands in for a real OpenCost response. Contract
   fixtures for the `kubecost` profile are allowed only under
   `testdata/kubecost-contract/`, labelled.
4. Every task has a `Verify:` command and a break check.
5. Anything not delivered goes under Not Delivered with the reason. Do not mark
   a task `DONE` for a stub.
6. A fixture the owner captures stays the owner's, even when the agent can
   reach the tool. Stop and ask unless the invocation hands the capture over.
7. Filter or field names that did not come from a real capture (`pod`, `node`,
   and `cluster` today) go in the report as unproven until the kind oracle
   covers them.
8. Update this file in the pull request that changes a status, and again after
   it merges.

## Real Backend and Oracle

Owner-owned and prepared before any agent work.

- **Backend:** a kind cluster, the `prometheus` chart with the OpenCost extra
  scrape config, and the `opencost` chart with `opencost.customPricing` set to
  fixed rates and the UI disabled. The values file is
  `hack/kind/opencost-values.yaml`.
- **Workload:** namespace `oc-test`, a Deployment of 2 replicas, each requesting
  500m CPU and 512Mi memory and idle (a `sleep`), so allocation is driven by
  requests.
- **Oracle:** expected cost is requests times the fixed rate times the window
  hours, computed from the manifests without calling OpenCost. Where OpenCost's
  allocation differs from this (it bills the larger of request and usage), the
  oracle file says why.
- **Break check:** change one rate in the values file, and the oracle
  comparison must fail.
- **Recorded responses:** `testdata/opencost-real/*.json` are genuine
  `/allocation` responses from that cluster, with the capture command in
  `testdata/opencost-real/README.md`.

The oracle shares nothing with the plugin. It does not prove that OpenCost's
pricing is what a cloud bills. Custom pricing is a configured rate.

## Keeping the Structs Current (Drift)

The plugin keeps its own small structs and does not import the OpenCost module.
That module pulls in about 150 requirements including the Azure, AWS, and GCS
SDKs, and `govulncheck` reports 10 findings in it (none called). Three checks
stand in for the import:

| Layer | Where | What it catches |
| --- | --- | --- |
| Field manifest | `ConsumedFields` and `IgnoredFields` in the plugin | A key added to or removed from the recorded response, or a field used without being declared. Unit test, every run |
| Upstream schema | the `test/drift` module (`make drift`) | OpenCost changing its `Allocation` type. Renovate bumps the pinned upstream version and the test fails on that pull request |
| Live server | the scheduled `live-drift.yml` job on the latest chart | Behaviour the Go type does not show: the envelope, the error format, a new chart default. Compares key sets against the recorded fixtures |

None of them catches a changed meaning or unit of an existing field. The oracle
comparison covers cost arithmetic. When a drift check fails, the owner
re-captures `testdata/opencost-real/`. The agent does not.

## Release and Tag Rules

Learned on opencost and azure-public. Follow them and do not rediscover them.

### Release Please settings

Every release-please config needs these on the `.` package, and a test that
fails when any is wrong:

1. `"include-component-in-tag": false`. Without it a config that sets
   `package-name` creates tags like `finfocus-plugin-<name>-v0.1.0`.
   `release.yml` runs on `release: created`, GoReleaser parses the tag as
   semver, fails, and the release has no binaries.
2. `"initial-version": "0.1.0"`. Without it the first release PR can propose
   1.0.0.
3. `.release-please-manifest.json` keeps `"."` at `0.0.0` until the first
   release PR merges.

Use `googleapis/release-please-action@v5.0.0`, not v4.

### GoReleaser settings

The release uploads archives and `checksums.txt` only:

- No `dockers`, `dockers_v2`, `homebrew_casks`, `brews`, or `nfpms` section, and
  no deb or rpm. `release.yml` has no registry login, no `packages: write`, and
  no tap token.
- Use only template fields GoReleaser defines. `{{if .Dirty}}` fails with
  `map has no entry for key "Dirty"`. Use `{{ .GitTreeState }}`.
- Use current keys: `goreleaser check` must exit 0, so use `formats`, not the
  deprecated `archives.format`.
- Names say `finfocus-plugin-<name>`.

Proof before the first tag, on the pinned GoReleaser version:

```bash
GORELEASER_CURRENT_TAG=v0.1.0 goreleaser release --snapshot --clean --skip=publish
goreleaser check
```

Both must succeed, and the build must write archives and `checksums.txt`. A
config that was never built is not a working config.

### Workflow files and the release token

- `release-please.yml`: `googleapis/release-please-action@v5.0.0` with
  `token: ${{ secrets.RELEASE_PLEASE_TOKEN }}` (falling back to
  `GITHUB_TOKEN`), triggers `push` to `main` and `workflow_dispatch`,
  permissions `contents`, `issues`, and `pull-requests` write. A probe step
  warns on an invalid token and falls back.
- `release.yml` (single-binary plugins): triggers `release: types: [created]`
  and `workflow_dispatch` with a required `tag` input, `permissions: contents:
  write`, checkout at `ref: ${{ inputs.tag || github.event.release.tag_name }}`
  with `fetch-depth: 0`, `actions/setup-go@v7` with `go-version-file: go.mod`,
  `goreleaser/goreleaser-action@v7` with `args: release --clean`, and
  `GITHUB_TOKEN` only.
- Not allowed: a `push: tags` trigger (release-please creates the tag through
  the API, so it never fires), `--rm-dist` (removed in GoReleaser v2),
  `actions/checkout` or `goreleaser-action` older than v7, `release-please-action`
  v4, and `google-apis/release-please-action` (the org is `googleapis`).

Why the token matters: events created with `GITHUB_TOKEN` do not trigger other
workflows, so a release PR opened that way never runs `Test` or `Commitlint`, and
a published release never fires `release.yml`. The secret is a fine-grained
token limited to this repo with Contents, Pull requests, and Issues read and
write, and Metadata read. The agent never reads or sets it. An expired token is
a non-empty string, so `||` does not rescue it. The probe covers that case.
Rotation is documented in `CLAUDE.md`.

### Release steps

- Merge the release PR, then check: the tag is plain `vX.Y.Z`, the GoReleaser
  run is green, and `gh release view vX.Y.Z --json assets --jq '.assets |
  length'` is above 0. A merged release PR is not a release.
- Recovery if the tag or the run is wrong: never hand-create a tag. The owner
  deletes the bad release and tag, then re-runs `release.yml` with the existing
  tag (`workflow_dispatch`). A tag points at the release PR's merge commit, so
  fixes merged afterwards ship in the next release.
- Release tooling (goreleaser, release-please, the release workflow) is
  ask-first: change it only when the task says so. The agent never deletes or
  moves a tag or release.

### Registry entry

A pull request to `rshade/finfocus`, modelled on #1697 (azure-public) and #1720
(opencost):

- `internal/registry/registry.json`: one entry with the provider, capabilities
  from the allowed list in `registry_json_test.go`, `asset_prefix`
  `finfocus-plugin-<name>` with `version_prefix: false`, and `min_spec_version`
  equal to the spec in the release's `go.mod`.
- A docs page `docs/src/content/docs/plugins/<name>.md`, a row in the
  compatibility matrix, the FAQ plugin table, the docs table of contents, and
  the `finfocus-install` skill references. Claim only what the plugin README
  and manifest say.
- Validate with a binary built from the branch and a fresh temporary
  `FINFOCUS_HOME`: `plugin list --available` shows the entry, `plugin install
  <name>` prints `Checksum verified (SHA256)`, and `plugin list` shows the
  version. Also run `plugin inspect <name> <type>`. It fails for this plugin
  today, which is OC-9.17.

State on 2026-11-12: v0.1.2 is the first release with assets (v0.1.0 and v0.1.1
have none). The registry entry merged in `rshade/finfocus` PR 1720.
