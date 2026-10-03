# Kubernetes example

This stack is a namespace, a deployment, a service, and a daemonset.
`pulumi preview` renders the manifests and does not apply them.
The provider uses `renderYamlToDirectory`, so the preview does not
need a cluster.

## Prerequisites

- Pulumi CLI 3.x
- The `finfocus` CLI, when you want cost output from the preview
- An OpenCost or Kubecost allocation API for a live query
- This plugin on the FinFocus plugin path

## Preview

`preview.json` is the output of this command against stack `dev` on a
file backend. Pulumi 3.264.0 exited 0. The plan creates six resources,
including the four Kubernetes types below. A diagnostic warning in that
file uses `~` in place of the local home directory.

```bash
cd examples/kubernetes
export PULUMI_BACKEND_URL="file://${PWD}/.pulumi-state"
export PULUMI_CONFIG_PASSPHRASE=example
pulumi stack init dev --non-interactive
pulumi preview --non-interactive --stack dev --color never --json
```

The steps are:

- `kubernetes:core/v1:Namespace` named `oc-example`
- `kubernetes:apps/v1:Deployment` named `fixed`
- `kubernetes:core/v1:Service` named `fixed`
- `kubernetes:apps/v1:DaemonSet` named `node-agent`

The plugin prices the namespace, the deployment, and the daemonset.
`Supports` is false for `kubernetes:core/v1:Service`.
A cluster node stays `node/<node name>`. This stack does not declare a
`Node`, because the control plane owns that object.

`rendered/` and `.pulumi-state/` are local and are gitignored.

## Plugin configuration

Copy `opencost.yaml` and set `baseUrl` to the allocation API.
`currency` is required when the allocation body does not name one.
The plugin does not assume USD. The API token is only
`KUBECOST_API_TOKEN`, and only the `kubecost` profile sends it.

```bash
export OPENCOST_CONFIG="${PWD}/opencost.yaml"
export OPENCOST_PROFILE=opencost
# export OPENCOST_PROFILE=kubecost
# export KUBECOST_API_TOKEN="set-this-outside-the-file"
```

## Costs through FinFocus

FinFocus reads a JSON plan whose top-level field is `steps`.
`pulumi preview --json` on Pulumi 3.264.0 writes that field, and each
step carries its type on `newState.type`.

```bash
finfocus cost projected --pulumi-json preview.json
finfocus cost actual --pulumi-json preview.json \
  --from 2026-09-01 --to 2026-10-03
```

Ids the plugin accepts for this stack:

- `namespace/oc-example`
- `controller/oc-example/fixed`
- `controller/oc-example/node-agent`
- `pod/oc-example/<pod name>`
- `node/<node name>`

Projected cost uses the Pulumi type. The namespace id is the name.
A deployment or daemonset id is `namespace/name`.

## Troubleshooting

- Set `PULUMI_CONFIG_PASSPHRASE` before `pulumi stack init` on a file backend.
- Keep `provider: ${render}` on every resource so preview does not call the Kubernetes API.
- A Service result of unsupported is the plugin's `Supports` answer for that type.
- An empty currency is `FailedPrecondition`. Set `currency` or `OPENCOST_CURRENCY`.
- Do not commit `.pulumi-state/`, `rendered/`, or `Pulumi.dev.yaml`.
