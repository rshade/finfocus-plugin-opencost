#!/usr/bin/env bash
# Run the live oracle against kind cluster oc-e2e. Create the cluster when it is absent.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export KUBECONFIG="${KUBECONFIG:-${root}/hack/kind/kubeconfig}"
cd "${root}"

if ! kind get clusters | grep -qx oc-e2e; then
	hack/kind/e2e-up.sh
fi
hack/kind/e2e-wait.sh
OC_E2E=1 go test -tags e2e -count=1 -timeout 20m -v ./test/e2e
