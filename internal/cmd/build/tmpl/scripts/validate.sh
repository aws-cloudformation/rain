#!/usr/local/bin/bash

set -eou pipefail

echo "Building rain..."
./scripts/build.sh

SOURCE="internal/cmd/build/tmpl"
OUT="test/templates/build"
FILES="${OUT}/**/*.yaml"
RULES="${SOURCE}/scripts/rules.guard"

echo "Building templates..."
echo "Building bucket..."
./rain build -r bucket bucket -o ${OUT}/bucket/bucket.yaml
echo "Building website..."
./rain build -r bucket website -o ${OUT}/bucket/website.yaml
echo "Building codecommit pipeline..."
./rain build -r pipeline codecommit -o ${OUT}/pipeline/codecommit.yaml
echo "Building s3 pipeline..."
./rain build -r pipeline s3 -o ${OUT}/pipeline/s3.yaml
echo "Building ecs cluster..."
./rain build -r ecs cluster -o ${OUT}/ecs/cluster.yaml
echo "Building VPC..."
./rain build -r vpc vpc -o ${OUT}/vpc/vpc.yaml

echo "Linting..."
cfn-lint ${FILES}

echo "Checkov..."
checkov --framework cloudformation --quiet -f ${FILES}

echo "Guard..."
cfn-guard validate --data ${OUT}/bucket --rules ${RULES} --show-summary fail
cfn-guard validate --data ${OUT}/pipeline --rules ${RULES} --show-summary fail
cfn-guard validate --data ${OUT}/ecs --rules ${RULES} --show-summary fail
cfn-guard validate --data ${OUT}/vpc --rules ${RULES} --show-summary fail

echo "Success"
