# Real OpenCost responses

Genuine responses from a kind cluster running the real charts. **Owner-owned: the run
agent may not create, edit or regenerate anything here.** Nothing in this directory is
hand-written except this file and `expected.json`.

## Captured

- Date: 2026-10-03 (about 3 minutes after the workload started).
- Charts: `prometheus` 29.35.0 (app v3.15.0) and `opencost` 2.5.32 (app 1.121.3). See `provenance/helm-releases.json`.
- Cluster: kind, one node. Values and workload: `provenance/oc-values.yaml`, `provenance/workload.yaml`
  (fixed custom pricing: CPU 2.0 per core-hour, RAM 1.0 per GiB-hour; two idle pods requesting 500m and 512Mi).
- Query base: `http://localhost:19003` (a port-forward to service `opencost` port 9003).

## Files

| File | Query |
| --- | --- |
| `allocation-namespace-60m.json` | `/allocation?window=60m&aggregate=namespace&includeIdle=false` |
| `allocation-pod-60m.json` | `...&aggregate=namespace,pod` |
| `allocation-controller-60m.json` | `...&aggregate=namespace,controller` |
| `allocation-namespace-step-1m.json` | `window=10m&step=1m&aggregate=namespace`; early steps are empty `{}` |
| `allocation-namespace-filtered.json` | `aggregate=pod&filter=namespace:"oc-test"` |
| `allocation-namespace-idle.json` | `includeIdle=true` |
| `allocation-compute-60m.json` | `/allocation/compute?window=60m&aggregate=namespace` |
| `error-bad-window.json`, `error-no-window.json` | HTTP 400, **plain text, not JSON** |

## What the data shows

- Envelope: `{"code":200,"data":[{"<name>":{...}}]}`, an array of name-keyed maps (one map per step).
- Aggregating by namespace and controller produces keys such as `oc-test/deployment:fixed` and `kube-system/__unallocated__`.
- Errors are HTTP 400 with a plain-text body. A client that always JSON-decodes the body will report a decode error instead of the real message.
- `__unmounted__` appears for unmounted volumes.

## Oracle

`expected.json` holds the formulas. For `oc-test` in `allocation-namespace-60m.json` (`minutes` 2.81201): `cpuCost` 0.09373 against 0.09373 expected
and `ramCost` 0.04687 against 0.04687 expected, relative difference below 1e-4 in all three files that carry it.
Use the response's `minutes` field, not the window length: the window start is rounded to Prometheus data.
This holds only while the pods are idle, because OpenCost bills the larger of request and usage.
The oracle shares no code with the plugin. It does not show that any cloud bills these rates.

## Regenerate (owner only)

Create a kind cluster, then install in this order:

```bash
helm install prometheus --repo https://prometheus-community.github.io/helm-charts prometheus \
  --namespace prometheus-system --create-namespace \
  --set prometheus-pushgateway.enabled=false --set alertmanager.enabled=false \
  -f https://raw.githubusercontent.com/opencost/opencost/develop/kubernetes/prometheus/extraScrapeConfigs.yaml
helm install opencost --repo https://opencost.github.io/opencost-helm-chart opencost \
  --namespace opencost --create-namespace -f provenance/oc-values.yaml
kubectl apply -f provenance/workload.yaml
```

Wait until `minutes` for `oc-test` is positive and `cpuCoreRequestAverage` is 1, then capture with `curl`.
