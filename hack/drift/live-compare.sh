#!/usr/bin/env bash
# Compare live /allocation keys and HTTP 400 bodies with the recorded fixtures.
# The fixtures directory is only read.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export KUBECONFIG="${KUBECONFIG:?set KUBECONFIG}"
context="${OC_DRIFT_CONTEXT:-kind-oc-drift}"
forward_pid=""

stop_forward() {
	if [[ -n "${forward_pid}" ]] && kill -0 "${forward_pid}" 2>/dev/null; then
		kill "${forward_pid}" 2>/dev/null || true
		wait "${forward_pid}" 2>/dev/null || true
	fi
}
trap stop_forward EXIT

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

cd "${root}"
go run ./hack/drift/livekeyset \
	-base "http://127.0.0.1:${port}" \
	-fixtures "${root}/testdata/opencost-real"
