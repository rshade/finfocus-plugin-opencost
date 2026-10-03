#!/usr/bin/env bash
# Delete kind cluster oc-e2e. Other clusters and kubeconfig contexts are left alone.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export KUBECONFIG="${root}/hack/kind/kubeconfig"
cluster=oc-e2e

if kind get clusters | grep -qx "${cluster}"; then
  kind delete cluster --name "${cluster}" --kubeconfig "${KUBECONFIG}"
fi
rm -f "${KUBECONFIG}"
