#!/usr/bin/env bash
# Open one bug issue for a live key-set difference, or comment on that open
# issue. Both carry the Actions run URL when GITHUB_RUN_ID is set.
# A log without added or removed keys does not create an issue.
set -euo pipefail

repo="${GITHUB_REPOSITORY:?set GITHUB_REPOSITORY}"
title="live drift: allocation key set differs from testdata/opencost-real"
log="$(cat)"

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
