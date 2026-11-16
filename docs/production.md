# Production setup

A `kubectl port-forward` is a development path. It needs a terminal that stays
open, a free local port, and `kubectl` on the machine. A CI runner or a pod
cannot rely on that, so the plugin can reach OpenCost through the Kubernetes API
server service proxy with the credentials you already have.

## How it works

Set `kube.service` and the plugin sends every allocation and health request to

```text
<API server>/api/v1/namespaces/<namespace>/services/<service>:<port>/proxy/<path>
```

It authenticates with a kubeconfig or, inside a pod, the pod's service account.
Static certificates, static tokens, and exec plugins such as `aws eks get-token`
and `gke-gcloud-auth-plugin` all work.

```yaml
profile: opencost
currency: EUR
kube:
  context: prod-east        # empty uses the current context
  namespace: opencost       # default
  service: opencost
  port: 9003                # default
```

`kube.service` replaces `baseUrl`. The kubeconfig owns the address and the trust
for that connection, so `baseUrl`, `tlsSkipVerify`, `caCertFile`, and
`allowInsecureHttp` next to it stop startup, and so does profile `kubecost`.

The environment can set the same keys without a file:
`OPENCOST_KUBE_CONTEXT`, `OPENCOST_KUBE_NAMESPACE`, `OPENCOST_KUBE_SERVICE`,
and `OPENCOST_KUBE_PORT`. The kubeconfig is `kube.kubeconfig`, then
`KUBECONFIG`, then `~/.kube/config`, then the pod's service account.

## Permissions

The caller needs one grant: `get` on `services/proxy` for the OpenCost service.
Give the plugin its own service account rather than an administrator's
kubeconfig, because a leaked kubeconfig reaches more than a bare OpenCost URL.

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: finfocus-opencost
  namespace: opencost
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: finfocus-opencost-proxy
  namespace: opencost
rules:
  - apiGroups: [""]
    resources: ["services/proxy"]
    resourceNames: ["opencost:9003"]
    verbs: ["get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: finfocus-opencost-proxy
  namespace: opencost
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: finfocus-opencost-proxy
subjects:
  - kind: ServiceAccount
    name: finfocus-opencost
    namespace: opencost
```

`resourceNames` is the service name and the port, `opencost:9003`. The name
`opencost` alone is denied, because the proxy path carries the port. If you set
`kube.port`, change the name to match.

## CI runner

The runner already has a kubeconfig. Point the plugin at it with the standard
`KUBECONFIG`, and name the service in the environment:

```bash
export KUBECONFIG=/path/to/runner-kubeconfig
export OPENCOST_KUBE_CONTEXT=prod-east
export OPENCOST_KUBE_SERVICE=opencost
export OPENCOST_CURRENCY=EUR
```

If the kubeconfig authenticates with an exec plugin, its command must be on the
runner's `PATH`: `aws` for EKS, `gke-gcloud-auth-plugin` for GKE. When the
command is missing or fails, the plugin says `kubeconfig credential plugin
"<command>" failed` and does not add the command's output to that error.
The command's own stderr passes through to the plugin's stderr, so its
diagnostics can appear in the run log.

## DevOps workstation

Use the current context, or pick one:

```yaml
kube:
  context: prod-east
  service: opencost
```

The plugin reads `~/.kube/config`, so `aws sso login` or `gcloud auth login`
must be current. A context that is not in the kubeconfig stops startup and the
message names it.

## A pod running `finfocus --web`

With no kubeconfig, the plugin uses the pod's service account. Run the pod as
the `finfocus-opencost` account from the example above, or bind another account
to the `Role`, and set only the service:

```bash
OPENCOST_KUBE_SERVICE=opencost
```

## When a request fails

| Symptom | gRPC code | Meaning |
| --- | --- | --- |
| Message names `services/proxy`, the namespace, and the service | `PermissionDenied` | The API server returned 403. Add the grant above. |
| Message names the context, namespace, and service | `Unavailable` | The service does not exist, has no ready endpoints, or the API server cannot be reached. |
| The API server rejected the credentials for a context | `Unauthenticated` | The token or certificate is expired or wrong. |
| `kube: context "x" is not in the kubeconfig` at startup | none, the process exits | The context name is wrong. |
| `kube: no kubeconfig found` at startup | none, the process exits | No kubeconfig and no pod service account. |

`Unavailable` here means the plugin never reached OpenCost. A reply from
OpenCost with no matching row is the no-data response, not `Unavailable`.

A long query can hit a timeout in the API server or a proxy in front of it
before the plugin's `timeout` passes. That also returns `Unavailable`. Shorten
the window or raise the limit on the proxy.

## Development

`kubectl port-forward` and `baseUrl` still work and stay the way to try the
plugin on a laptop:

```bash
kubectl -n opencost port-forward svc/opencost 9003:9003
```

```yaml
baseUrl: http://localhost:9003
```

`make e2e-kind` exercises both paths against a kind cluster.
