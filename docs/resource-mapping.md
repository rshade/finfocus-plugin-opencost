# Kubernetes resource mapping

This note records what FinFocus core puts on a cost-source request for a Kubernetes resource, and what `plugins/kubernetes` puts on a descriptor of its own. Citations are paths in the sibling `finfocus` checkout unless a line says `finfocus-spec`.

## What core sends

Core does not rename a Pulumi resource into a `k8s-*` token. Ingest copies the stack export's type onto the engine descriptor:

- `internal/ingest/state.go:218` sets `Type: resource.Type`.

The proto adapter copies that same string into `ResourceDescriptor.resource_type`:

- Projected cost, the call that reaches the plugin: `internal/proto/adapter.go:1387` sets `ResourceType: resource.Type`.
- The pre-flight projected request uses the same assignment at `internal/proto/adapter.go:123`.
- Recommendation targets use it at `internal/proto/adapter.go:1751`.
- Error records keep that string at `internal/proto/adapter.go:144`, `:160`, `:180`, and `:196`. They do not translate it.

`GetActualCost` does not send `resource_type`. The proto request built at `internal/proto/adapter.go:405` carries `ResourceId`, `Start`, `End`, `Tags`, and `Arn`. The internal request still holds `ResourceType` (`internal/proto/adapter.go:525`) so SKU resolution can use it. The id example in the spec is `namespace/default` (`finfocus-spec` `proto/finfocus/v1/costsource.proto:247`).

## Types that appear as literals

These Pulumi tokens appear in sibling tests and docs, so a stack that contains them is what core will forward:

| Token | Where it is spelled |
| --- | --- |
| `kubernetes:core/v1:Pod` | `internal/resourcetype/resourcetype.go:26`, `internal/router/provider.go:30` |
| `kubernetes:apps/v1:Deployment` | `internal/engine/engine_supports_test.go:155`, `internal/ingest/map_resource_test.go:187` |
| `kubernetes:apps/v1:StatefulSet` | `test/integration/routing_patterns_test.go:135` |
| `kubernetes:apps/v1:DaemonSet` | `test/integration/routing_patterns_test.go:143` |
| `kubernetes:core/v1:Service` | `internal/ingest/mapper_test.go:382` |

The routing fixture that names Deployment, StatefulSet, and DaemonSet is `test/integration/routing_patterns_test.go:113`. That regex does not include Pod. Pod is still forwarded when the stack type is `kubernetes:core/v1:Pod`, because ingest copies the type rather than applying that regex (`internal/ingest/state.go:218`).

`kubernetes:core/v1:Namespace`, `kubernetes:core/v1:Node`, `kubernetes:apps/v1:ReplicaSet`, `kubernetes:batch/v1:Job`, and `kubernetes:batch/v1:CronJob` are not spelled in the sibling tests searched for this note. They are accepted because the copy at `internal/ingest/state.go:218` and `internal/proto/adapter.go:1387` forwards whatever token the stack contains, and OpenCost can filter a namespace, a node, or a controller. That is an inference from the copy, not a literal found in core.

## What the Kubernetes plugin sends

`plugins/kubernetes` does not price workloads. `Supports` returns false for every request, including a Deployment (`plugins/kubernetes/plugin.go:151`, exercised with `kubernetes:apps/v1:Deployment` at `plugins/kubernetes/plugin_test.go:274`).

When that plugin builds a descriptor for a node, `ResourceType` is a cloud token, not `kubernetes:core/v1:Node`:

- `plugins/kubernetes/usage/nodes.go:52` documents the token as provider-specific.
- AWS is `aws:ec2/instance:Instance`, GCP is `gcp:compute/instance:Instance`, Azure is `azure-native:compute:VirtualMachine` (`plugins/kubernetes/usage/nodes.go:73`).
- The descriptor is returned at `plugins/kubernetes/usage/nodes.go:87`.
- An EKS Fargate pod is `aws:eks/fargate:Pod` (`plugins/kubernetes/usage/nodes.go:35` and `:121`).

This cost source does not support those `aws:*`, `gcp:*`, or `azure-native:*` tokens. The Kubernetes plugin is the usage source for them. This plugin prices allocation rows for Kubernetes objects.

## How an id becomes an OpenCost filter

`ResourceDescriptor.id` stays an opaque correlation token. A Pulumi URN is not split on `/`. The OpenCost object name comes from `metadata.name` and `metadata.namespace`. Spec v0.7.3 puts the nested form on `ResourceDescriptor.attributes`. When that path is set, it replaces the flattened tag. Core still flattens the same names onto tags, and `GetActualCost` reads tags only. `GetActualCost` still accepts `namespace/<name>`, `pod/<namespace>/<name>`, `controller/<namespace>/<name>`, and `node/<name>`. A bare cloud id such as `oc-example` is a namespace or node when `resource_type` says so. `oc-example/fixed` is a pod or controller. A slash is not a namespace or node name. A kind-prefixed id that disagrees with `resource_type` is `InvalidArgument`. An empty id stays `InvalidArgument` even when `metadata.name` is set.

`Supports` is true for the Pulumi tokens core forwards for objects OpenCost can filter:

- `kubernetes:core/v1:Namespace`
- `kubernetes:core/v1:Pod`
- `kubernetes:core/v1:Node`
- `kubernetes:apps/v1:Deployment`
- `kubernetes:apps/v1:StatefulSet`
- `kubernetes:apps/v1:DaemonSet`
- `kubernetes:apps/v1:ReplicaSet`
- `kubernetes:batch/v1:Job`
- `kubernetes:batch/v1:CronJob`

`kubernetes:core/v1:Service` is a real stack type (`internal/ingest/mapper_test.go:382`). This plugin returns `Supported: false` with a reason. OpenCost allocation filters used here are namespace, pod, controller, and node.

The spec's resource type examples include `k8s-namespace` (`finfocus-spec` `proto/finfocus/v1/costsource.proto:768`). Core docs repeat that example and do not show a translator:

- `docs/src/content/docs/architecture/plugin-protocol.md:271`
- `docs/src/content/docs/reference/api-reference.md:697`

`k8s-namespace`, `k8s-pod`, `k8s-controller`, and `k8s-node` are accepted as aliases of those examples. They are not what the adapter writes at `internal/proto/adapter.go:1387`.
