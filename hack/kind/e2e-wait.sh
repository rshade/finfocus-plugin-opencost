#!/usr/bin/env bash
# Poll OpenCost until namespace oc-test has a positive minute count.
# Timeout is OC_WAIT_TIMEOUT seconds (default 900). The poll interval is 10 seconds.
# The awaited condition is the allocation row, not a fixed sleep.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export KUBECONFIG="${root}/hack/kind/kubeconfig"
context=kind-oc-e2e
namespace="${1:-oc-test}"
timeout="${OC_WAIT_TIMEOUT:-900}"
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

deadline=$((SECONDS + timeout))
while [[ "${SECONDS}" -lt "${deadline}" ]]; do
  body="$(mktemp)"
  if curl -fsS "http://127.0.0.1:${port}/allocation?window=60m&aggregate=namespace&includeIdle=false" -o "${body}"; then
    if NAMESPACE="${namespace}" python3 - "${body}" <<'PY'
import json
import os
import sys

with open(sys.argv[1], encoding="utf-8") as handle:
    doc = json.load(handle)
name = os.environ["NAMESPACE"]
for step in doc.get("data") or []:
    row = step.get(name) if isinstance(step, dict) else None
    if isinstance(row, dict) and float(row.get("minutes") or 0) > 0:
        print(f"{name} minutes={row.get('minutes')}")
        sys.exit(0)
sys.exit(1)
PY
    then
      exit 0
    fi
  fi
  rm -f "${body}"
  echo "waiting for ${namespace} in /allocation (timeout ${timeout}s)"
  sleep 10
done

echo "timed out after ${timeout}s waiting for ${namespace}" >&2
exit 1
