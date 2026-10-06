#!/usr/bin/env bash
# Exit 0 when any filename is under internal/, test/e2e/, or hack/kind/.
set -euo pipefail
grep -Eq '^(internal/|test/e2e/|hack/kind/)'
