#!/usr/local/bin/bash

set -eou pipefail

echo "Building rain..."
./scripts/build.sh

echo "Generating go pkl modules..."
# We don't actually need these for anything yet...
pkl-gen-go pkl/cloudformation.pkl --base-path github.com/aws-cloudformation/rain
pkl-gen-go pkl/template.pkl --base-path github.com/aws-cloudformation/rain

echo "Building pkl classes..."

#echo "AWS::S3::Bucket"
#./rain build --pkl-class AWS::S3::Bucket > pkl/aws/s3/bucket.pkl
#pkl eval pkl/aws/s3/bucket.pkl
#
#echo "AWS::IAM::RolePolicy"
#./rain build --pkl-class AWS::IAM::RolePolicy > pkl/aws/iam/rolepolicy.pkl
#pkl eval pkl/aws/iam/rolepolicy.pkl
#
#echo "AWS::IAM::Role"
#./rain build --pkl-class AWS::IAM::Role > pkl/aws/iam/role.pkl
#pkl eval pkl/aws/iam/role.pkl

# shellcheck disable=SC2002
cat internal/cmd/forecast/all-types.txt | xargs -n1 scripts/gen-one-pkl.sh

echo "Testing patterns..."
pkl eval test/pkl/bucket.pkl | ./rain fmt | cfn-lint

echo "Success!"
