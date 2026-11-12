# `finfocus-plugin-opencost` Context and Boundaries

## Core Architectural Identity

`finfocus-plugin-opencost` is a gRPC `CostSource` plugin named `opencost` for
[FinFocus](https://github.com/rshade/finfocus), the Pulumi cost CLI. FinFocus
starts the plugin and dials it on `127.0.0.1`. The plugin reads allocation
costs for Kubernetes objects from the OpenCost HTTP API and serves them as
projected cost, actual cost, a pricing spec, an estimate, batch cost, and
budgets.

It reports those costs for Kubernetes objects only: namespace, controller,
pod, and node. It translates between two vocabularies and does not own either
one. FinFocus owns the resource model and the protocol (`finfocus-spec`).
OpenCost owns the prices and the allocation arithmetic.

## Technical Boundaries ("Hard No's")

- **No `AllocatorService` and no `ALLOCATION` capability.** OpenCost totals are
  not a split of host priced node costs, so they fail the conservation
  invariant in `finfocus-spec`. See `docs/allocator-decision.md`.
- **No gRPC authentication.** FinFocus dials the plugin on loopback, so
  authentication is a non-goal.
- **No cloud price catalog.** The plugin does not publish an AWS or other
  provider price. An unsupported type, including `ec2`, is `InvalidArgument`.
- **No persisted state.** The only state is an in-memory allocation cache with
  a TTL (default 30 seconds) and a size cap of 256 live bodies.
- **No import of the OpenCost Go module in the plugin module.** The plugin
  talks to OpenCost over HTTP and keeps its own small structs for the JSON it
  reads, so `go.mod` has no `github.com/opencost/opencost` line. Importing it
  would pull in about 150 requirements, including the Azure, AWS, and GCS SDKs.
  The separate `test/drift` module is the only place that imports it. It checks
  those structs against the upstream `Allocation` type.
- **No assumed currency.** The plugin never falls back to USD. A missing
  currency is `FailedPrecondition`.
- **No invented cost.** A zero cost is only a real zero. A missing row is
  `NotFound` on projected cost, pricing spec, and estimate, and the SDK no-data
  response on actual cost.
- **No hand-written stand-in for an OpenCost response.** `testdata/opencost-real/`
  holds genuine captures and is owner-owned and read-only. Contract fixtures for
  profile `kubecost` live under `testdata/kubecost-contract/` and are labelled.
- **No metrics, tracing, or alerting service.** The health check is the SDK
  `Check` over gRPC. The process mounts no HTTP `/healthz`.
- **No secrets in files.** The API token comes only from `KUBECOST_API_TOKEN`.
  A YAML `apiToken` is ignored, and logs omit the token.
- **No published image from CI.** Release uploads archives and `checksums.txt`
  only.

## Data Source of Truth

OpenCost's allocation API is the source of truth for cost: `GET /allocation`
for profile `opencost`. Profile `kubecost` uses `GET /model/allocation`,
`GET /model/budgets`, and `POST /model/prediction/speccost`, and is verified
against contract fixtures only, not a live backend.

The kind oracle is the source of truth for cost arithmetic: expected cost is
requests times the fixed rate times the window hours, computed from the
manifests without calling OpenCost.

## Interaction Model

- **Inbound:** gRPC from FinFocus on port 50051, with reflection enabled.
- **Outbound:** HTTP to the allocation base URL, rate limited to 10 requests
  per second with burst 20, with TLS verification on unless `tlsSkipVerify` is
  set.
- **Configuration:** environment variables and an optional YAML file. The
  environment wins. A missing file is tolerated and a malformed one is an
  error.
- **Change process:** OpenSpec for behaviour changes, direct commits for tests,
  docs, CI, and fixes with no surface change. See `CLAUDE.md`.

## Verification

Before accepting a feature or an idea, check each of these questions:

1. Does it persist state, call a provider API, or publish a price? If yes, it
   crosses a preceding boundary. Delegate it to FinFocus core or another plugin.
2. Does it need a new `finfocus-spec` message or capability? Ask
   whether the need is specific to this plugin. Usually the answer is the plugin,
   core, or docs.
3. Does it change a cost number? Then `make e2e-kind` must show the oracle
   match, and the oracle must still fail when a rate is broken.
4. Does it change an RPC, the config surface, a profile, the resource mapping,
   or the cost arithmetic? Then it needs an OpenSpec change.
