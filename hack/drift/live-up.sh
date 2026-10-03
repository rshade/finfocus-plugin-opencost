#!/usr/bin/env bash
# Install the latest published OpenCost chart on kind cluster oc-drift.
# Prometheus stays on the pinned chart. This script does not touch oc-e2e.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export KUBECONFIG="${KUBECONFIG:-${RUNNER_TEMP:-$(mktemp -d)}/oc-drift.kubeconfig}"
cluster=oc-drift
context="kind-${cluster}"
kind_dir="${root}/hack/kind"
forward_pid=""

stop_forward() {
	if [[ -n "${forward_pid}" ]] && kill -0 "${forward_pid}" 2>/dev/null; then
		kill "${forward_pid}" 2>/dev/null || true
		wait "${forward_pid}" 2>/dev/null || true
	fi
	forward_pid=""
}

on_exit() {
	status=$?
	stop_forward
	if [[ "${status}" -ne 0 && "${OC_KEEP_FAILED_CLUSTER:-}" != "1" ]]; then
		kind delete cluster --name "${cluster}" --kubeconfig "${KUBECONFIG}" >/dev/null 2>&1 || true
	fi
}
trap on_exit EXIT

if kind get clusters | grep -qx "${cluster}"; then
	kind delete cluster --name "${cluster}" --kubeconfig "${KUBECONFIG}"
fi

kind create cluster \
	--name "${cluster}" \
	--kubeconfig "${KUBECONFIG}" \
	--config "${kind_dir}/cluster.yaml" \
	--wait 180s

helm upgrade --install prometheus prometheus \
	--kube-context "${context}" \
	--repo https://prometheus-community.github.io/helm-charts \
	--version 29.35.0 \
	--namespace prometheus-system \
	--create-namespace \
	--set prometheus-pushgateway.enabled=false \
	--set alertmanager.enabled=false \
	-f "${kind_dir}/extraScrapeConfigs.yaml" \
	--wait \
	--timeout 15m

helm upgrade --install opencost opencost \
	--kube-context "${context}" \
	--repo https://opencost.github.io/opencost-helm-chart \
	--namespace opencost \
	--create-namespace \
	-f "${kind_dir}/opencost-values.yaml" \
	--wait \
	--timeout 10m

helm --kube-context "${context}" list --namespace opencost

kubectl --context "${context}" apply -f "${kind_dir}/workload.yaml"

forward_log="$(mktemp)"
kubectl --context "${context}" --namespace opencost port-forward svc/opencost :9003 >"${forward_log}" 2>&1 &
forward_pid=$!

port=""
ready_deadline=$((SECONDS + 60))
while [[ "${SECONDS}" -lt "${ready_deadline}" ]]; do
	port="$(sed -n 's/.*127\.0\.0\.1:\([0-9][0-9]*\).*/\1/p' "${forward_log}" | head -n 1)"
	if [[ -n "${port}" ]]; then
		break
	fi
	if ! kill -0 "${forward_pid}" 2>/dev/null; then
		cat "${forward_log}" >&2
		exit 1
	fi
	sleep 1
done
if [[ -z "${port}" ]]; then
	cat "${forward_log}" >&2
	exit 1
fi

body="$(mktemp)"
deadline=$((SECONDS + 900))
while [[ "${SECONDS}" -lt "${deadline}" ]]; do
	code="$(curl -sS -o "${body}" -w '%{http_code}' --max-time 30 \
		"http://127.0.0.1:${port}/allocation?window=60m&aggregate=namespace&includeIdle=true" || true)"
	if [[ "${code}" == "200" ]] && python3 - "${body}" <<'PY'
import json, sys
body = json.load(open(sys.argv[1]))
for step in body.get("data") or []:
    if isinstance(step, dict):
        for item in step.values():
            if isinstance(item, dict) and item:
                sys.exit(0)
sys.exit(1)
PY
	then
		echo "allocation object is present"
		exit 0
	fi
	sleep 10
done
echo "timed out waiting for an allocation object" >&2
exit 1
