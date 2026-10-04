# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Building and Testing

```bash
# Build the plugin binary
make build

# Run tests
make test

# Run linting (requires golangci-lint)
make lint

# Install to local plugin directory
make install

# Build the container image from the source tree
docker build -t finfocus-plugin-opencost:dev .

# Print -version and -port. The process exits 0.
docker run --rm finfocus-plugin-opencost:dev --help
```

`make` goals are `.PHONY`. A directory named `test` would otherwise make `make test` report that it is up to date and skip the recipe.

## Project Architecture

This is a gRPC plugin that implements the CostSource service from `finfocus-spec`. The plugin name is `opencost`. `GetPluginInfo` reports `PROJECTED_COSTS`, `ACTUAL_COSTS`, `PRICING_SPEC`, `ESTIMATE_COST`, `BATCH_COST`, and `BUDGETS`. It does not report `ALLOCATION`. Key components:

- **gRPC Server**: Listens on port 50051, implements the CostSource service methods
- **Allocation client**: HTTP client in `internal/allocation`. Profile `opencost` queries the OpenCost allocation API. Profile `kubecost` uses the model allocation path.
- **Configuration**: Supports both environment variables and YAML config files

## Key Implementation Details

### Resource ID Mapping

`ResourceDescriptor.id` is an opaque correlation token. A Pulumi URN is not split on `/`. When the id is opaque, the OpenCost name comes from `metadata.name` and `metadata.namespace`. On a descriptor, those values in `attributes` replace the same flattened tags. `GetActualCost` has no attributes field and still reads tags. It accepts these filters:

- `namespace/<name>` → filter by namespace
- `pod/<namespace>/<name>` → filter by namespace and pod
- `controller/<namespace>/<name>` → filter by namespace and controller
- `node/<name>` → filter by node

A bare cloud id such as `oc-example` is a namespace or node when `resource_type` says so. `oc-example/fixed` is a pod or controller. A kind-prefixed id that disagrees with the resource type, including a `resource_type` tag on `GetActualCost`, is `InvalidArgument`. An empty id stays `InvalidArgument` even when `metadata.name` is set.

### Cost Projections

`GetProjectedCost` queries window `30d` for the requested object, not the cluster. Monthly cost is `totalCost / (minutes / 60) * 730`. `billing_detail` states that 30-day trailing average. `cost_breakdown` sums to `cost_per_month`. A namespace name is `metadata.name` or a bare id with no slash. `unit_price` is the observed hourly rate. Currency comes from the allocation body or pricing config (`OPENCOST_CURRENCY`). The plugin does not assume USD.

### Cost prediction

Profile `opencost` `EstimateCost` reads `metadata.name` as a namespace and queries the allocation API for window `30d`. An empty `metadata.namespace` on that profile uses the namespace `default`. Profile `kubecost` `EstimateCost` posts the resource attributes unchanged to `POST /model/prediction/speccost` and returns `costAfter.totalMonthlyRate`. An empty `metadata.namespace` on that profile matches the row in `defaultNamespace`. Query names are `clusterID`, `defaultNamespace`, `windowAvgUsage` (`predictionWindow`, default `2d`), `windowResourceCost` (`7d offset 48h`), and `noUsage=false`. Currency comes from pricing config. The response fixture is a contract example.

### Error Handling

- HTTP client includes timeout support via context
- TLS certificate verification can be disabled for development
- All errors are propagated with context

## Dependencies

The plugin depends on:
- `github.com/rshade/finfocus-spec/sdk/go/proto` - Protocol buffer definitions
- `google.golang.org/grpc` v1.86.0-dev. `govulncheck` reports GO-2026-6443 for v1.84.0, and no stable tag newer than v1.84.0 is published. The plugin server does not call `xds.NewGRPCServer`.
- `gopkg.in/yaml.v3` for configuration parsing

## Testing Approach

- Unit tests for individual components (client, config, server methods)
- Integration tests can use the testdata/ JSON files
- For live testing, set `KUBECOST_BASE_URL` environment variable

## Common Development Tasks

### Adding New Resource Types

1. Update `Supports()` in `internal/server/supports.go`
2. Add mapping logic in `GetActualCost()` for the new resource ID format
3. Update `plugin.manifest.json` with the new resource type

### Legacy environment names

These names are still read. Profile `opencost` does not send them. Profile `kubecost` `EstimateCost` sends `clusterID`, `defaultNamespace`, and `windowAvgUsage` from `predictionWindow`.

```bash
export KUBECOST_CLUSTER_ID="production-cluster"
export KUBECOST_DEFAULT_NAMESPACE="default"
export KUBECOST_PREDICTION_WINDOW="7d"
```

### Debugging the gRPC Server

The server includes reflection support, so you can use tools like `grpcurl`:

```bash
grpcurl -plaintext localhost:50051 list
grpcurl -plaintext localhost:50051 describe CostSource
```

### Modifying allocation API calls

The HTTP client in `internal/allocation/client.go` queries the allocation API. Exported allocation types are `Entry`, `Properties`, `Window`, `Query`, `Point`, and `Response`. `DetailedAllocationResponse` keeps its name. To add new endpoints:

1. Add new methods to the Client struct
2. Define request/response types
3. Handle the API call with proper error handling and timeout

## Go-Specific Development Patterns

### Struct Field Consistency

- **Critical**: Field names must be consistent across related structs
- Example issue: `PVCost` vs `PVCCost` caused compilation failures
- Always verify field names when copying/adapting struct definitions

### URL Parameter Handling

- Map iteration order is random in Go, affecting URL parameter order
- Use flexible testing patterns that accept multiple valid orders:

  ```go
  if !contains(url, "order1") && !contains(url, "order2") {
      t.Errorf("Expected URL to contain parameters in either order")
  }
  ```

### Error Handling Best Practices

- Use `errors.Is(err, os.ErrNotExist)` for file existence checks
- Graceful degradation: missing config files should not fail hard
- Always check for unused variables in strict builds (`_ = variable`)

### Protobuf and Timestamp Handling

- Use `timestamppb.New(time)` instead of manual timestamp construction
- Avoid duplicate helper functions when standard library provides them

## Project-Specific Architecture Insights

### Allocation API Methods

The project has two allocation methods with different capabilities:
- **Basic `Allocation`**: Simple method, was incomplete (fixed in recent update)
- **Enhanced `EnhancedAllocation`**: Full-featured method using `GetDetailedAllocation` + `ConvertToSimpleResponse`
- **Recommendation**: Use `EnhancedAllocation` for new implementations

### Testing Architecture

- Uses mock HTTP servers (`httptest.NewServer`) for integration testing
- Mock protobuf types defined locally due to missing `finfocus-spec` dependency
- Integration tests validate end-to-end functionality including URL building and response parsing

### Configuration Handling

- Supports both environment variables and YAML files
- Environment variables take precedence
- Config loading gracefully handles missing files with `os.ErrNotExist`

## Common Development Issues & Solutions

### Dependency Management

- Issue: Missing `go.sum` entries cause build failures
- Solution: Run `go mod tidy` when adding new imports or after git operations
- Symptom: "missing go.sum entry for module" errors

### Linting Configuration

- Issue: `embeddedstructfieldcheck` linter requires golangci-lint v2.3+
- Solution: Disable incompatible linters in `.golangci.yml` for older versions
- Check version compatibility before enabling new linters

### URL Building and Testing

- Issue: Go's map iteration randomness affects URL parameter order
- Solution: Test for multiple valid parameter orders or use URL parsing
- Debug tip: Create temporary debug files to inspect generated URLs

### Parallel Process Conflicts

- Issue: "parallel golangci-lint is running" errors
- Solution: Wait between runs or use direct `golangci-lint run` command
- Alternative: Use `--timeout` flag to prevent hanging processes

## Testing Strategies

### Integration Testing with Mock Servers

```go
// Effective pattern for testing HTTP API clients
mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // Validate request parameters
    // Return mock response
}))
defer mockServer.Close()

// Point client to mock server
client := NewClient(Config{BaseURL: mockServer.URL})
```

### URL Parameter Testing

- Test both parameter presence and proper encoding
- Account for random map iteration order in Go
- Use helper functions to reduce test code duplication

### Error Scenario Testing

- Test missing configuration files
- Test network timeouts and failures  
- Test malformed API responses
- Test invalid parameter combinations

## Tool Usage and Workflow Optimizations

### Go Testing Commands

```bash
# Test specific package with verbose output
go test ./internal/allocation -v

# Test specific function
go test ./internal/allocation -v -run TestFunctionName

# Test with race detection
go test -race ./...
```

### Debugging Techniques

- Create temporary debug files for URL inspection
- Use `fmt.Printf` debugging in tests (remove before commit)
- Check HTTP request/response details in mock server handlers

### Build and Validation Workflow

```bash
# Complete validation sequence
go mod tidy
make build
make test
make lint
```

### golangci-lint Best Practices

- Check version compatibility before updating config
- Use `--timeout=60s` flag for slow systems
- Disable strict linters during development, enable for production

## Allocation profiles and recorded responses

- Basic SDK conformance is `OC_E2E=1 go test -tags conformance ./test/conformance`. It dials the plugin binary over TCP. The adapter rewrites only the suite's `aws`/`ec2` probe to namespace `oc-test` before that call. `Supports` for aws stays false. A direct `ec2` call stays `InvalidArgument`. `go test ./...` does not build that tag. The untagged package still tests the rewrite. On the live cluster the suite passed 10, failed 0, skipped 2, with projected unit price 3.749997 EUR.
- `.github/workflows/kind.yml` runs `make e2e-kind` on `ubuntu-latest`. `helm/kind-action` uses `install_only: true`, so the make target creates cluster `oc-e2e`. The job runs nightly, on `workflow_dispatch`, or when a pull request is labeled `e2e-kind`. A failure uploads `kind export logs` for that cluster. The first CI run is an owner action. Issues #14 and #7 stay open.
- `make e2e-kind-up` creates kind cluster `oc-e2e` with kubeconfig `hack/kind/kubeconfig` and context `kind-oc-e2e`. It installs Prometheus chart 29.35.0 and OpenCost chart 2.5.32, applies `hack/kind/workload.yaml`, and waits until `/allocation/compute?window=60m` returns HTTP 200. `make e2e-kind-wait` polls `/allocation` until namespace `oc-test` has positive `minutes`, for up to 900 seconds. `make e2e-kind` sets `KUBECONFIG` to `hack/kind/kubeconfig` and reuses `oc-e2e` when that cluster exists, then runs `go test -tags e2e ./test/e2e`. `OC_KEEP_FAILED_CLUSTER=1` leaves a failed cluster in place so CI can export its logs. The kind and live-drift workflows set that variable. That test compares `GetActualCost` for `namespace/oc-test` with `testdata/opencost-real/expected.json` on a two-minute window, because OpenCost prices the window from `node_cpu_hourly_cost`. It sets CPU to 9.0, requires the oracle comparison to fail, and restores 2.0. `make e2e-kind-up` deletes `oc-e2e` before creating it. `make e2e-kind-down` deletes only that cluster. kubectl and helm in these scripts pass that context and kubeconfig.
- `Check` implements the SDK health checker. It sends `GET /healthz` to the allocation base URL. A status below 400 is healthy. A status of 400 or higher is an error that includes the status code. The probe does not use the allocation cache or the rate limit, and it uses the same profile authentication as an allocation request. Each health check and cost RPC logs `trace_id`, `requests`, and `latency`. Outbound requests copy the SDK trace id when the context has one. The process stays on gRPC, so it does not mount its own HTTP `/healthz`. `main` calls `plugin.SetLogger` so those lines reach stderr. `pluginsdk.Serve` on this path does not call `Check`. Issue #11 is closed. Prometheus, OpenTelemetry, and alerting are not this check. zerolog `Info` is a pointer method, so assign `requestLogger` to a local before calling it.
- The client package is `internal/allocation`. Profile `opencost` (the default) calls `GET /allocation` with `includeIdle` and `shareIdle` and does not send a token. Profile `kubecost` calls `GET /model/allocation` with `idle`, `accumulate`, and `shareIdle`, and sends a bearer token. Set `OPENCOST_PROFILE` or `profile` in the config. Filter keys are sorted. TLS verification is on unless `tlsSkipVerify` is set. The API token comes only from `KUBECOST_API_TOKEN`; a YAML `apiToken` is ignored, and request logs omit the token. Filter values that contain a quote, plus, backslash, whitespace, or parenthesis are `InvalidArgument` and are not sent. A successful allocation URL is cached for `cacheTTL` (default 30 seconds). A negative `cacheTTL` disables it. Insert deletes entries whose deadline has passed, and it evicts the least recently used body once more than 256 live bodies are stored. A read inside the TTL counts as use. Actual, projected, and estimate results set `expires_at` to that deadline. Outbound requests default to 10 per second with burst 20 (`requestsPerSecond`, `rateBurst`). Over the limit is `ResourceExhausted` and does not call the backend. The HTTP transport pools 10 idle connections per host and sets dial, handshake, idle, and header timeouts. The cache key is the URL, so it does not store the token.
- `DecodeAllocationBody` reads typed cost fields. `ConsumedFields` and `IgnoredFields` together cover every key of the recorded allocations. Those objects have 53 keys. `allocation-namespace-idle.json` also has `proportionalAssetResourceCosts` and `sharedCostBreakdown`. Do not edit `testdata/opencost-real/`.
- HTTP 400 bodies in that directory are plain text, not JSON.
- Cost RPCs use `finfocus-spec` v0.7.3. `GetPluginInfo` reports `pluginsdk.SpecVersion`. `Supports` accepts the Pulumi tokens documented in `docs/resource-mapping.md`. A descriptor's `attributes` win over flattened `metadata.name` and `metadata.namespace` tags. `GetActualCost` still uses tags only.
- `GetActualCost` maps `namespace/<name>`, `controller/<namespace>/<name>`, `pod/<namespace>/<name>`, and `node/<name>` to a sorted OpenCost filter. A cloud id is accepted when `resource_type` names a supported object: a bare name for a namespace or node, and `namespace/name` for a pod or controller. It returns typed `totalCost` for matching rows only, including a real zero. A node request matches `properties.node` because there is no node-aggregated recording. Labels from that row are `focus_record.tags`. A namespace row uses `namespaceLabels` when `labels` is empty, and it does not copy `controllerKind`. A pod or controller row keeps `controllerKind` as extended column `controllerKind`. The result timestamp is `window.start`, then `start`. An unparsed time is `FailedPrecondition`. Annotations are extended columns `annotation.<key>`. `focus_record.billing_currency` comes from the allocation body when it names one currency, otherwise from pricing config. Empty is `FailedPrecondition`, not USD. A missing row keeps the caller's resource id in `NO_COST_DATA`.
- `GetProjectedCost` keeps the descriptor id as a correlation token. A Pulumi URN is not parsed as a Kubernetes name. The OpenCost name comes from `metadata.name` and `metadata.namespace` tags, or from a bare name or `namespace/name` when the id is not opaque. It queries `window=30d` and projects `totalCost / (minutes / 60) * 730`. An empty id is `InvalidArgument`. The response `currency` uses the same rule. `unit_price` stays the observed hourly rate. `NO_COST_DATA` names the original descriptor id.
- `GetPricingSpec` uses that same 30-day query. `provider` is `kubernetes`, `billing_mode` is `per_hour`, and `rate_per_unit` is the observed hourly rate. Its assumption says that hourly rate. The 730-hour sentence stays on projected cost. Currency uses the same rule. An unsupported type, including `ec2`, is `InvalidArgument`. The plugin does not publish an AWS price. A zero rate is only a real zero.
- `EstimateCost` reads `metadata.name` for a namespace. `BatchCost` issues one allocation query and returns results in request order. A missing object is a per-item `NotFound`, not an RPC error, even when currency is unset. A present row with no currency is a per-item `FailedPrecondition`. The plugin reports `BATCH_COST`. Pricing config currency is `currency` in YAML, and `OPENCOST_CURRENCY` overrides it.
- This plugin does not implement `AllocatorService` and does not report `ALLOCATION`. OpenCost totals are not a split of host priced node costs, so they fail conservation in `finfocus-spec` `proto/finfocus/v1/allocation.proto`. See `docs/allocator-decision.md`. Issue #48 stays open.
- `.github/workflows/live-drift.yml` runs nightly or on `workflow_dispatch`. It creates kind cluster `oc-drift` (not `oc-e2e`), installs Prometheus chart 29.35.0, and installs the latest published `opencost` chart with no version pin. `go run ./hack/drift/livekeyset` compares the live `/allocation` key set and the HTTP 400 plain-text bodies with `testdata/opencost-real`. A difference prints `added:` and `removed:` and exits 1. The command only reads the fixtures. On the pinned chart 2.5.32 the key set matched (55 keys). Issue #14 stays open. The first CI run is an owner action.
- `test/drift` is a separate module. Its `go.mod` replaces this module with `../..`, so `go mod tidy` there follows the plugin's `finfocus-spec` require. `make drift` marshals `opencost.Allocation` from `github.com/opencost/opencost/core` v1.121.3 and requires those keys to equal `ConsumedFields` plus `IgnoredFields` and the recorded allocation objects. The plugin module does not import OpenCost. Empty `proportionalAssetResourceCosts` and `sharedCostBreakdown` maps are omitted by upstream JSON, so the drift value sets them. Do not add either key to the wire lists just to quiet a failure.
- The README configuration reference and `config.example.yaml` list every YAML key on `allocation.Config`. `OPENCOST_CONFIG` wins over `KUBECOST_CONFIG`. `OPENCOST_PROFILE`, `OPENCOST_CURRENCY`, and `KUBECOST_API_TOKEN` replace the file. A YAML `apiToken` is ignored. `defaultWindow`, `clusterId`, `defaultNamespace`, and `predictionWindow` are loaded. Profile `opencost` does not send the last three. Profile `kubecost` `EstimateCost` sends them. Profile `opencost` `EstimateCost` uses the 30-day allocation query. Profile `kubecost` `EstimateCost` posts to `POST /model/prediction/speccost`. Vale `IgnoredScopes` skips inline code. Issue #15 stays open.
- `examples/kubernetes` is a Pulumi YAML stack. `pulumi preview --json` renders a namespace, deployment, service, and daemonset through `renderYamlToDirectory` and does not need a cluster. `examples/kubernetes/preview.json` is that output from Pulumi 3.264.0 on a file backend. FinFocus reads `steps`, and the resource type is on `newState.type`. The plugin prices the namespace, deployment, and daemonset. Service stays unsupported. A node id is `node/<node name>`. Issue #6 stays open.
- `Dockerfile` builds `./cmd/finfocus-plugin-opencost` with `CGO_ENABLED=0` on `golang:1.27.1` and copies that binary into `gcr.io/distroless/static-debian12:nonroot`. `docker run --rm finfocus-plugin-opencost:dev --help` prints `-version` and `-port` and exits 0. `.github/workflows/test.yml` builds tag `finfocus-plugin-opencost:dev`. `.github/workflows/kind.yml` builds the same tag and smoke-runs `--help` before `make e2e-kind`. The smoke step accepts exit 0 or 2 and requires both flags. GoReleaser's docker context has the prebuilt binary plus `config.example.yaml` and `plugin.manifest.json`, so `.goreleaser.yaml` sets `dockerfile: Dockerfile.goreleaser`. That file copies `$TARGETPLATFORM/finfocus-plugin-opencost` and does not run `go build`. `goreleaser check` exits 0. Archives use `formats`, snapshot uses `version_template`, images use `dockers_v2`, and the formula publish uses `homebrew_casks`. Issue #10 stays open. These jobs do not push the image.
- `CLAUDE.md` and `plugin.manifest.json` may mention `kubecost` only as profile `kubecost` or as a `KUBECOST_` environment name. There is no `ROADMAP.md`. The manifest `capabilities` list matches `GetPluginInfo`: `PROJECTED_COSTS`, `ACTUAL_COSTS`, `PRICING_SPEC`, `ESTIMATE_COST`, `BATCH_COST`, and `BUDGETS`. Profile `kubecost` `GetBudgets` calls `GET /model/budgets` and keeps rules whose `values.namespace` is set. A rule with several namespaces is one budget: `spendLimit` and `currentSpend` stay the rule amounts, and metadata `namespaces` lists every name. A request tag `namespace` keeps that namespace, and a shared rule's id becomes `<rule id>/<namespace>`. An unfiltered summary counts the rule once. Currency comes from pricing config. Cluster rules are omitted. When `include_status` is set, spend over `spendLimit` is exceeded, spend equal to `spendLimit` is critical, spend at or above the lowest action percentage is warning, and spend below that percentage is ok. The summary counts those four states for the budgets the namespace filter returns. Critical stays its own count. Without `include_status` the summary is absent. The fixture is a contract example. Chart kubecost 3.3.0 ran on kind cluster oc-kubecost with no product key: Helm exit 0, nine pods ready, frontend HTTP 200. Chart cost-analyzer 2.9.7 did not render without a federated store. GET /model/allocation on that frontend was HTTP 404. The run did not create or delete oc-e2e. See docs/kubecost-kind-spike.md. The plugin stays on the contract fixtures. `testdata/kubecost-contract/SOURCES.md` lists each JSON fixture with its source URL or kubectl-cost commit and the date checked. There is no upstream module to marshal, so that guard is weaker than the OpenCost drift check.
