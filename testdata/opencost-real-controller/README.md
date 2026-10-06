# Controller filter capture

Live responses from kind cluster `oc-e2e` on 2026-10-04.
Charts: Prometheus 29.35.0 (app v3.15.0) and OpenCost 2.5.32 (app 1.121.3).
Query base: `http://localhost:19003`, a port-forward to service `opencost` port 9003.
Window `60m`. Aggregate `namespace,controller`.

## Files

| File | Result |
| --- | --- |
| `error-controller-filter.json` | HTTP 500. Filter `controller:"fixed"+namespace:"oc-test"`. |
| `allocation-controllername-60m.json` | HTTP 200. Filter `controllerName:"fixed"+namespace:"oc-test"`. Row `oc-test/deployment:fixed`, `totalCost` 0.02425. |

## Command

```bash
curl -sG http://localhost:19003/allocation --data-urlencode 'window=60m' \
  --data-urlencode 'aggregate=namespace,controller' \
  --data-urlencode 'filter=controller:"fixed"+namespace:"oc-test"'
curl -sG http://localhost:19003/allocation --data-urlencode 'window=60m' \
  --data-urlencode 'aggregate=namespace,controller' \
  --data-urlencode 'filter=controllerName:"fixed"+namespace:"oc-test"'
```
