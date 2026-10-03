# Kubecost chart on kind

answer: yes

```text
cluster: oc-kubecost
paid token: not set
chart: cost-analyzer
version: 2.9.7
chart: kubecost
version: 3.3.0
```

The run did not create or delete oc-e2e. The e2e kind up target was not used.
The cluster was deleted after the probe. `kind get clusters` was empty before
and after. This spike does not change the plugin. Profile `kubecost` stays on
the contract fixtures. The frontend of chart `kubecost` 3.3.0 returned HTTP 200,
and `GET /model/allocation` returned HTTP 404.

## Chart cost-analyzer 2.9.7

`helm template` exits 1 when no token and no federated store are set. Upstream
says this release is only for preparing an existing install to upgrade to 3.x.
It was not installed.

```text
helm template kubecost cost-analyzer \
  --repo https://kubecost.github.io/cost-analyzer/ \
  --version 2.9.7 \
  --namespace kubecost \
  --set kubecostToken= \
  --set global.clusterId=oc-kubecost \
  --set prometheus.server.global.external_labels.cluster_id=oc-kubecost
```

```text
Error: execution error at (cost-analyzer/templates/NOTES.txt:13:4):

Error: Missing global federated-store. Please set a global federated-store.
See examples at: https://github.com/kubecost/kubecost/tree/v2.9/examples.
NOTE: Kubecost v2.9 is only used for preparing agents to upgrade to v3.0.

COST_ANALYZER_TEMPLATE_EXIT:1
```

## Chart Kubecost 3.3.0

Checked 2026-10-03. Kind node image `kindest/node:v1.35.0`. The node was
labeled `topology.kubernetes.io/region=kind` and
`topology.kubernetes.io/zone=kind` so the network-costs DaemonSet could
schedule. No product key was passed. `helm get values` shows only `clusterId`.

```text
kind create cluster --name oc-kubecost --wait 180s
KIND_CREATE_EXIT:0

helm upgrade --install kubecost kubecost \
  --repo https://kubecost.github.io/kubecost \
  --version 3.3.0 \
  --namespace kubecost \
  --create-namespace \
  --set global.clusterId=oc-kubecost \
  --timeout 15m \
  --wait
HELM_EXIT:0
STATUS: deployed
REVISION: 1
```

```text
USER-SUPPLIED VALUES:
global:
  clusterId: oc-kubecost
```

```text
NAME                                           READY   STATUS    RESTARTS   AGE
kubecost-aggregator-0                          1/1     Running   0          109s
kubecost-cloud-cost-869c494b5c-fk2x6           1/1     Running   0          109s
kubecost-cluster-controller-69f5684bff-wzcn7   1/1     Running   0          109s
kubecost-finopsagent-5db58dfb88-d4frj          1/1     Running   0          109s
kubecost-forecasting-7fdb7b4cf6-gl6hq          1/1     Running   0          109s
kubecost-frontend-794c58b7b4-6cnvl             1/1     Running   0          109s
kubecost-local-store-68f45f4bc9-8cshz          1/1     Running   0          109s
kubecost-mcp-5d4b95887f-tcgqd                  1/1     Running   0          109s
kubecost-network-costs-nr7bf                   1/1     Running   0          109s
READY_COUNT:9
```

Port-forward `svc/kubecost-frontend` port 9090:

```text
ROOT_HTTP:200
MODEL_HTTP:404
MODEL_HEAD:
404 page not found
```

`kind delete cluster --name oc-kubecost` removed only that cluster.
