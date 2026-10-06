# FinFocus OpenCost plugin

`finfocus-plugin-opencost` is a FinFocus cost source named `opencost`.
It reads Kubernetes allocation costs over HTTP and serves them with gRPC.
The protocol is `finfocus-spec` v0.7.3.

## Capabilities

`GetPluginInfo` reports these capabilities:

- `PROJECTED_COSTS`
- `ACTUAL_COSTS`
- `PRICING_SPEC`
- `ESTIMATE_COST`
- `BATCH_COST`
- `BUDGETS`

The provider list is `kubernetes`.
The plugin does not report `ALLOCATION` and does not implement `AllocatorService`.
Profile `kubecost` `GetBudgets` calls `GET /model/budgets` and returns rules whose `values.namespace` is set.
A request tag `namespace` keeps that namespace.
`amount.currency` comes from pricing config.
When `include_status` is set, spend over the limit is exceeded, spend equal to the limit is critical, and spend that reaches the lowest action percentage is warning.
Spend under that percentage is healthy.
The summary counts OK, warning, critical, and exceeded for the budgets the filter returns.
Those four counts equal the number of budgets.
Critical stays its own count.
A request without `include_status` omits the summary.
A rule with an unsupported interval or no spend limit is skipped with one WARN per rule; the rest are returned.
The first returned budget carries `skippedRules` and `skippedRuleReasons` in metadata.
A namespace filter that removes every returned budget also removes that metadata; the WARN log still names each skipped rule.
That response is a contract fixture and is not verified against live Kubecost.
That decision is in [docs/allocator-decision.md](docs/allocator-decision.md).

## Installation

```bash
make build
make install
```

`make build` writes `bin/finfocus-plugin-opencost`.
`make install` copies that binary to `~/.finfocus/plugins/opencost/0.1.0/`.

## Configuration reference

The process reads the file in `OPENCOST_CONFIG`.
When `OPENCOST_CONFIG` is empty, it reads `KUBECOST_CONFIG`.
A missing file leaves the environment values in place, even when the path is a typo.

Precedence runs defaults first, then the environment, then the file for the keys it sets.
`OPENCOST_PROFILE`, `OPENCOST_CURRENCY`, and `KUBECOST_API_TOKEN` are read again after the file and always win.

A file that does not parse, a `KUBECOST_TIMEOUT` value that is not a duration, or a `KUBECOST_TLS_SKIP_VERIFY` value other than `true` or `false` stops startup with an error that names the file or the variable.
Before serving, the plugin validates the config: `baseUrl` must be an `http` or `https` URL with a host, `profile` must be empty, `opencost`, or `kubecost`, a set `currency` must be three upper-case letters, and `timeout`, `requestsPerSecond`, and `rateBurst` must not be negative.
A failure exits with a one-line reason.
No error or log line contains the API token.

The API token is `KUBECOST_API_TOKEN` only.
A YAML `apiToken` field is ignored.
Request logs do not include the token.

`config.example.yaml` contains every YAML key from the client config.

| Key | Environment variable | Default | Effect |
| --- | --- | --- | --- |
| `baseUrl` | `KUBECOST_BASE_URL` | empty | Origin of the allocation API. A query needs this value. |
| `profile` | `OPENCOST_PROFILE` | `opencost` | `opencost` or `kubecost`. The environment variable wins. |
| `currency` | `OPENCOST_CURRENCY` | empty | ISO 4217 code used when the body does not name one currency. The environment variable wins. The plugin does not assume USD. |
| `defaultWindow` | `KUBECOST_DEFAULT_WINDOW` | `30d` | Stored on the client. Cost methods send their own window. |
| `timeout` | `KUBECOST_TIMEOUT` | `15s` | HTTP client timeout. |
| `tlsSkipVerify` | `KUBECOST_TLS_SKIP_VERIFY` | `false` | Set the variable to `true` to skip certificate checks. Leave it unset on a shared cluster. |
| `cacheTTL` | none | `30s` | How long a successful allocation URL is reused. `0s` selects 30 seconds. A negative value disables the cache. |
| `requestsPerSecond` | none | `10` | Outbound allocation rate. `0` selects 10 per second. Past that limit, the call is `ResourceExhausted` and does not reach the backend. |
| `rateBurst` | none | `20` | How many outbound requests may run together. `0` selects 20. |
| `clusterId` | `KUBECOST_CLUSTER_ID` | empty | Profile `kubecost` `EstimateCost` sends this as `clusterID`. Other cost methods do not. |
| `defaultNamespace` | `KUBECOST_DEFAULT_NAMESPACE` | `default` | Profile `kubecost` `EstimateCost` sends this as `defaultNamespace`. Other cost methods do not. |
| `predictionWindow` | `KUBECOST_PREDICTION_WINDOW` | `2d` | Profile `kubecost` `EstimateCost` sends this as `windowAvgUsage`. Other cost methods do not. |

## Profiles

An empty `profile` selects `opencost`.
Any other value than `opencost` or `kubecost` returns `unknown allocation profile`.

### Profile `opencost`

The client calls `GET /allocation` with `includeIdle=false` and `shareIdle=false`.
It does not send a bearer token.

### Profile `kubecost`

The `kubecost` profile calls `GET /model/allocation` with `idle=false`, `accumulate=false`, and `shareIdle=false`.
It sends `Authorization: Bearer` when `KUBECOST_API_TOKEN` is set.
An empty token sends no `Authorization` header.

Both profiles probe `GET /healthz` on `baseUrl`.
A status below 400 is healthy.
A status of 400 or higher is an error that includes the status code.
The probe skips the allocation cache and the rate limit.
The plugin process stays on gRPC and does not serve HTTP `/healthz` itself.

Filter values that contain a quote, plus sign, backslash, whitespace, or parenthesis are `InvalidArgument`.
The client does not send those values.

## Resource identifiers

`GetActualCost` accepts these resource ids:

- `namespace/<name>`
- `controller/<namespace>/<name>`
- `pod/<namespace>/<name>`
- `node/<name>`

`Supports` accepts these types:

- `kubernetes:core/v1:Namespace` and `k8s-namespace`
- `kubernetes:core/v1:Pod` and `k8s-pod`
- `kubernetes:core/v1:Node` and `k8s-node`
- `kubernetes:apps/v1:Deployment`
- `kubernetes:apps/v1:StatefulSet`
- `kubernetes:apps/v1:DaemonSet`
- `kubernetes:apps/v1:ReplicaSet`
- `kubernetes:batch/v1:Job`
- `kubernetes:batch/v1:CronJob`
- `k8s-controller`

`kubernetes:core/v1:Service` is unsupported.
`Supports` returns false with a reason.
A direct `ec2` request is `InvalidArgument` on projected cost and pricing.

For projected cost and pricing, the descriptor id is the name of a namespace or a node.
A pod or controller id is `namespace/name`.
How core copies Pulumi types is recorded in [docs/resource-mapping.md](docs/resource-mapping.md).

## API notes

`GetActualCost` returns `totalCost` for matching rows, including a real zero.
No matching row is `NotFound`.
Labels from the row are focus tags.
The controller kind is extended column `controllerKind`.
Annotations are extended columns `annotation.<key>`.
The query window is the window on the request.

`GetProjectedCost` queries window `30d`.
Monthly cost is `totalCost / (minutes / 60) * 730`.
`unit_price` is the observed hourly rate.
`billing_detail` states that this is a 30-day trailing average projected to a 730-hour month.

`GetPricingSpec` uses that same 30-day query.
`provider` is `kubernetes`.
`billing_mode` is `per_hour`.
`unit` is `hour`.
`source` is `opencost`.
`rate_per_unit` is the observed hourly rate.
A zero rate is only a real zero total.

On profile `opencost`, `EstimateCost` reads `metadata.name` for a namespace and queries window `30d` on the allocation API.
On profile `kubecost`, `EstimateCost` posts the resource attributes to `POST /model/prediction/speccost`.
`cost_monthly` is `costAfter.totalMonthlyRate` from that response.
The fixture follows `kubectl-cost` `pkg/query/prediction_speccost.go` at commit `1f45d3085b2ffa84758bfa8131ea8b7784cd8ed1`.
It is not verified against live Kubecost.

`BatchCost` issues one allocation query and returns results in request order.
A missing object is a per-item `NotFound`, including when currency is unset.
A present row with no currency is a per-item `FailedPrecondition`.

Currency comes from the allocation body when that body names one currency.
Otherwise it comes from `currency` or `OPENCOST_CURRENCY`.
When both are empty, the RPC is `FailedPrecondition` and the message is `currency is not set in the allocation response or pricing config`.

## Deployment

Build the binary and install it for a local FinFocus host:

```bash
make build
make install
```

Build a container from the source tree.
The image runs the static binary as `nonroot`.

```bash
docker build -t finfocus-plugin-opencost:dev .
docker run --rm finfocus-plugin-opencost:dev --help
```

`--help` prints `-version`, `-version-full`, and `-port`, and the process exits 0.
`-port` overrides `FINFOCUS_PLUGIN_PORT`.

The project kind suite is `make e2e-kind`.
`make e2e-kind-up` deletes a cluster named `oc-e2e` and then creates it.

[examples/kubernetes](examples/kubernetes/README.md) is a Pulumi stack for a namespace, a deployment, a service, and a `DaemonSet`.
`pulumi preview` renders manifests and does not need a cluster.
FinFocus can price the checked-in plan:

```bash
finfocus cost projected --pulumi-json examples/kubernetes/preview.json
finfocus cost actual --pulumi-json examples/kubernetes/preview.json \
  --from 2026-09-01 --to 2026-10-03
```

The service in that plan stays unsupported.

## Troubleshooting

- Message `currency is not set in the allocation response or pricing config`: set `currency` or `OPENCOST_CURRENCY`. Recorded OpenCost bodies omit currency.
- Message `unknown allocation profile`: set `profile` to `opencost` or `kubecost`.
- Message `allocation request rate limit exceeded`: the RPC is `ResourceExhausted`. Raise `requestsPerSecond` or `rateBurst`. That call did not reach the API.
- Certificate errors on a local server: set `KUBECOST_TLS_SKIP_VERIFY` to `true`. Leave verification on for a shared cluster.
- The `kubecost` profile returns status 401: the RPC is `Unauthenticated`. Set `KUBECOST_API_TOKEN`. The `opencost` profile does not send that token.
- The backend returns status 403: the RPC is `PermissionDenied`. The token in `KUBECOST_API_TOKEN` lacks the permission the endpoint needs.
- `Supports` names `kubernetes:core/v1:Service`: this plugin has no allocation filter for a Service.
- Message `no cost data available for resource`: the window has no matching row. `GetActualCost` maps that to `NotFound`.
- The API returns HTTP 400 as plain text. A rejected window starts with `Invalid 'window' parameter`.
- Health fails while a cached cost call succeeds: `Check` calls `GET /healthz` and does not read the allocation cache.
- `docker run --help` exits 0. The usage text is on stderr and includes `-version` and `-port`.

## Testing

```bash
make test
```

`make e2e-kind` compares `GetActualCost` for namespace `oc-test` with `testdata/opencost-real/expected.json`.
Tests do not rewrite that directory.

## License

Apache-2.0. See [LICENSE](LICENSE).
