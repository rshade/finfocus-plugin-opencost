#!/usr/bin/env bash
# Bring up kind cluster oc-e2e, Prometheus 29.35.0, and OpenCost 2.5.32.
# Success is an HTTP 200 from /allocation/compute?window=60m. The cluster stays up.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
kind_dir="${root}/hack/kind"
export KUBECONFIG="${kind_dir}/kubeconfig"
cluster=oc-e2e
context="kind-${cluster}"
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
  if [[ "${status}" -ne 0 ]]; then
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
  --version 2.5.32 \
  --namespace opencost \
  --create-namespace \
  -f "${kind_dir}/opencost-values.yaml" \
  --wait \
  --timeout 10m

kubectl --context "${context}" apply -f "${kind_dir}/workload.yaml"
kubectl --context "${context}" --namespace oc-test rollout status deployment/fixed --timeout=180s

forward_log="$(mktemp)"
kubectl --context "${context}" --namespace opencost port-forward svc/opencost :9003 >"${forward_log}" 2>&1 &
forward_pid=$!

port=""
deadline=$((SECONDS + 60))
while [[ "${SECONDS}" -lt "${deadline}" ]]; do
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
http_code=""
deadline=$((SECONDS + 180))
while [[ "${SECONDS}" -lt "${deadline}" ]]; do
  http_code="$(curl -sS -o "${body}" -w '%{http_code}' "http://127.0.0.1:${port}/allocation/compute?window=60m" || true)"
  if [[ "${http_code}" == "200" ]]; then
    break
  fi
  sleep 2
done

echo "opencost /allocation/compute?window=60m http=${http_code}"
head -c 200 "${body}" || true
echo
if [[ "${http_code}" != "200" ]]; then
  exit 1
fi
