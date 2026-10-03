# Kubecost contract sources

Kubecost has no upstream module this plugin can marshal. The OpenCost drift
check compares recorded keys with `github.com/opencost/opencost`. These
fixtures have no equivalent module, so this list is the weaker guard. Each
row names one JSON fixture, its source URL or kubectl-cost commit, and the
date checked. The bodies are contract examples and are not verified against live Kubecost.

| Fixture | Source | Checked |
| --- | --- | --- |
| speccost.json | <https://github.com/kubecost/kubectl-cost> commit 1f45d3085b2ffa84758bfa8131ea8b7784cd8ed1 file pkg/query/prediction_speccost.go | 2026-10-03 |
| budgets.json | <https://www.ibm.com/docs/en/SSW0JQG_1.x/apis/apis-overview/budget-api.html> | 2026-10-03 |
| budgets-namespaces.json | <https://www.ibm.com/docs/en/SSW0JQG_1.x/apis/apis-overview/budget-api.html> | 2026-10-03 |
| budgets-health.json | <https://www.ibm.com/docs/en/SSW0JQG_1.x/apis/apis-overview/budget-api.html> | 2026-10-03 |
