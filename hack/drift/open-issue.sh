#!/usr/bin/env bash
# Open one bug issue for a nightly drift failure, or comment on that open issue.
# Usage: open-issue.sh [keyset|oracle] < log
#   keyset (default): the livekeyset output. A log without added or removed keys
#                     does not create an issue.
#   oracle:           go test output of the cost oracle on the latest chart. A log
#                     without a failed test does not create an issue.
# Both carry the Actions run URL when GITHUB_RUN_ID is set.
set -euo pipefail

repo="${GITHUB_REPOSITORY:?set GITHUB_REPOSITORY}"
mode="${1:-keyset}"
log="$(cat)"

case "${mode}" in
keyset)
	title="live drift: allocation key set differs from testdata/opencost-real"
	differs=false
	while IFS= read -r line; do
		case "${line}" in
		"added: none" | "removed: none") ;;
		added:* | removed:*) differs=true ;;
		esac
	done <<< "${log}"
	if [[ "${differs}" != true ]]; then
		echo "no key-set difference"
		exit 0
	fi
	body="${log}"
	;;
oracle)
	title="live oracle: cost differs from testdata/opencost-real/expected.json"
	if ! grep -q -- '^--- FAIL: ' <<< "${log}"; then
		echo "no oracle failure"
		exit 0
	fi
	body="The cost oracle failed against the latest published OpenCost chart. A failure is either a plugin regression or a deliberate OpenCost pricing change; the kind job on the pinned chart tells the two apart."
	body+=$'\n\n'"$(tail -n 150 <<< "${log}")"
	;;
*)
	echo "unknown mode ${mode}; want keyset or oracle" >&2
	exit 2
	;;
esac

if [[ -n "${GITHUB_RUN_ID:-}" ]]; then
	body+=$'\n\n'"Run: ${GITHUB_SERVER_URL:-https://github.com}/${repo}/actions/runs/${GITHUB_RUN_ID}"
fi

number="$(
	gh issue list --repo "${repo}" --state open --limit 100 --json number,title \
		--jq ".[] | select(.title == \"${title}\") | .number" | head -n 1
)"

if [[ -z "${number}" ]]; then
	gh issue create --repo "${repo}" --title "${title}" --label bug --body "${body}"
	exit 0
fi

gh issue comment "${number}" --repo "${repo}" --body "${body}" >/dev/null
echo "https://github.com/${repo}/issues/${number}"
