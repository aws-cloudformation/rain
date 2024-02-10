#!/usr/local/bin/bash

set -eou pipefail

BASE="internal/cmd/build/tmpl"
FILES="${BASE}/**/*.yaml"
RULES="${BASE}/scripts/rules.guard"

echo "Linting..."
cfn-lint ${FILES}

echo "Checkov..."
checkov --framework cloudformation --quiet -f ${FILES}

echo "Guard..."
cfn-guard validate --data ${BASE}/bucket --rules ${RULES} --show-summary fail

echo "Success"
