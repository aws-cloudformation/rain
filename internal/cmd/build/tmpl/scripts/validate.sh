#!/usr/local/bin/bash

set -eou pipefail

SOURCE="internal/cmd/build/tmpl"
OUT="test/templates/build"
FILES="${OUT}/**/*.yaml"
RULES="${SOURCE}/scripts/rules.guard"

echo "Building templates..."
./rain build -r bucket bucket -o ${OUT}/bucket/bucket.yaml
./rain build -r bucket website -o ${OUT}/bucket/website.yaml
./rain build -r pipeline codecommit -o ${OUT}/pipeline/codecommit.yaml
./rain build -r pipeline s3 -o ${OUT}/pipeline/s3.yaml

echo "Linting..."
cfn-lint ${FILES}

echo "Checkov..."
checkov --framework cloudformation --quiet -f ${FILES}

echo "Guard..."
cfn-guard validate --data ${OUT}/bucket --rules ${RULES} --show-summary fail

echo "Success"
