#!/usr/local/bin/bash

set -eou pipefail

echo "Bucket..."
pkl eval test/pkl/bucket.pkl | ./rain fmt | cfn-lint

echo "VPC..."
pkl eval test/pkl/vpc-pattern.pkl -f yaml | ./rain fmt | cfn-lint

echo "Lambda..."
pkl eval test/pkl/lambda.pkl -f yaml | ./rain fmt | cfn-lint -i E3002

