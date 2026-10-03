# finfocus-plugin-opencost

A FinFocus **CostSource** plugin that reads **actual** and **projected** Kubernetes costs from **Kubecost** via its HTTP API (e.g., `/model/allocation`), exposed over gRPC using the `costsource.proto` from `finfocus-spec`.

## Capabilities

- **Actual cost** by Kubernetes dimension (cluster, namespace, controller, pod, node, label)
- **Projected cost** using Kubecost pricing data (CPU/RAM/GPUs, node share, amortized assets)
- Pluggable, isolated process compatible with FinFocus plugin host

## Installation (dev)

```bash
git clone https://github.com/<you>/finfocus-plugin-opencost
cd finfocus-plugin-opencost
go mod tidy
make build
```

This builds bin/finfocus-plugin-opencost. Place it where FinFocus can find it:

```text
~/.finfocus/plugins/opencost/1.0.0/finfocus-plugin-opencost
```

# Folder structure
```text
finfocus-plugin-opencost/
├─ README.md
├─ go.mod
├─ go.sum
├─ cmd/
│  └─ finfocus-plugin-opencost/
│     └─ main.go
├─ internal/
│  ├─ server/
│  │  ├─ kubecost_server.go
│  │  └─ validate.go
│  ├─ kubecost/
│  │  ├─ client.go
│  │  ├─ allocation.go
│  │  └─ config.go
│  └─ util/
│     └─ time.go
├─ pkg/
│  └─ version/
│     └─ version.go
├─ proto/                            # pulled in via submodule or copied from finfocus-spec
│  └─ costsource.proto               # (optional local copy for dev; canonical in finfocus-spec)
├─ plugin.manifest.json
├─ config.example.yaml
├─ Makefile
└─ testdata/
   ├─ sample_request.json
   ├─ sample_response_actual.json
   └─ sample_response_projected.json
```
# Configuration
Use env vars or a YAML config (path can be provided via KUBECOST_CONFIG):
```text
KUBECOST_BASE_URL (e.g., https://kubecost.example.com)

KUBECOST_API_TOKEN (optional, if your Kubecost requires auth)

KUBECOST_DEFAULT_WINDOW (e.g., 30d, fallback if query lacks dates)

KUBECOST_TIMEOUT (e.g., 15s)

KUBECOST_TLS_SKIP_VERIFY (true|false)
```

config.example.yaml shows all fields.

# Protocol
Implements CostSource from finfocus-spec/proto/costsource.proto. Methods:

* Name()
* Supports(ResourceDescriptor)
* GetActualCost(ActualCostQuery)
* GetProjectedCost(ResourceDescriptor)
* GetPricingSpec(ResourceDescriptor)


# Mapping
ResourceDescriptor fields → Kubecost filters:

* `Provider: "gcp"|"aws"|"azure"`: used as a hint (optional)
* `ResourceType`: `"k8s-node" | "k8s-namespace" | "k8s-pod" | "k8s-controller"`
* `Region`: optional filter (mapped via cluster labels if available)
* `SKU`: optional; often unused in K8s context
* `Tags`: maps to label selectors (e.g., app=web)

`ActualCostQuery.ResourceID` accepts flexible IDs:

* `namespace/<name>`
* `pod/<ns>/<podName>`
* `controller/<ns>/<ctrl>`
* `node/<nodeName>`

# Security

* Prefer HTTPS to Kubecost
* Limit token scope; avoid logging secrets
* Redact sensitive fields in errors/logs

# Testing
```bash
make test
```
Use testdata/ JSON fixtures. For live tests, set KUBECOST_BASE_URL and (optional) KUBECOST_API_TOKEN.

# License
[Apache-2.0](LICENSE)

# plugin.manifest.json

```json
{
  "name": "kubecost",
  "version": "1.0.0",
  "kind": "cost",
  "providers": ["kubernetes", "aws", "gcp", "azure"],
  "resourceTypes": ["k8s-namespace", "k8s-pod", "k8s-controller", "k8s-node"],
  "entrypoint": "finfocus-plugin-opencost"
}
