#!/usr/bin/env bash
# Print the OpenCost chart version to install. OC_CHART_VERSION overrides the
# pin in versions.env. The value latest prints nothing, so the caller omits
# --version and helm installs the newest published chart.
set -euo pipefail

dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=versions.env
source "${dir}/versions.env"
version="${OC_CHART_VERSION:-${OPENCOST_CHART_VERSION}}"
if [[ "${version}" != "latest" ]]; then
  echo "${version}"
fi
