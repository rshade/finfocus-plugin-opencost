# Real OpenCost responses, 14 days

Genuine `/allocation` responses from the owner's long-running kind cluster `oc-e2e`, captured by the PM. **Owner-owned:
the run agent may not create, edit or regenerate anything here.** Nothing is hand-written except this file.
Prometheus keeps 15 days, so this window cannot be captured again.

## Captured

- Date: 2026-10-21T12:04:53Z. Window: `2026-10-07T12:04:53Z,2026-10-21T12:04:53Z` (20155.1 minutes).
- Charts: `prometheus` 29.35.0, `opencost` 2.5.32 (app 1.121.3). Cluster: kind, one node `oc-e2e-control-plane`.
- Workload: namespace `oc-test`, Deployment `fixed`, 2 idle pods (pods `fixed-7996d494d-s7qfz` and `fixed-7996d494d-vr7vr`) requesting 500m CPU
  and 512Mi each. Custom pricing: CPU 2.0 per core-hour, RAM 1.0 per GiB-hour.

## Files

| File | Query (base `/allocation`) |
| --- | --- |
| `ns-14d.json` | `aggregate=namespace&includeIdle=false` |
| `ns-14d-daily.json` | same plus `step=24h`: 14 steps |
| `ns-14d-idle.json` | `aggregate=namespace&includeIdle=true` |
| `pod-14d.json` | `aggregate=namespace,pod` |
| `controller-14d.json` | `aggregate=namespace,controller` |
| `node-14d.json` | `aggregate=node` |
| `filtered-octest-14d.json` | `aggregate=pod&filter=namespace:"oc-test"` |

## Expected values

Use the response's own fields, never the plugin. For any row: `cpuCost = cpuCoreHours x 2.0` and
`ramCost = ramByteHours / 2^30 x 1.0`. Checked on 2026-10-21 for `oc-test` in `ns-14d.json`: `cpuCost` 669.72111
against 669.72112, `ramCost` 334.86056 against 334.8606.

Do not use window length times the request: the pods were not present for the whole window, so request-based cost
(1007.76) is 0.3% above the returned `totalCost` (1004.58). OpenCost bills the larger of request and usage, and these
pods are idle, so request drives the bill here.

Namespace totals add up to the node total. The oracle's inputs come from OpenCost itself, so it proves the plugin
passes OpenCost's numbers through correctly and that OpenCost's arithmetic follows the configured rates. It does not
show that any cloud bills these rates.
