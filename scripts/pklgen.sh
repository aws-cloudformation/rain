#!/usr/local/bin/bash

echo "Building rain..."
./scripts/build.sh

echo "Building pkl classes..."

echo "AWS::S3::Bucket"
./rain build --pkl-class AWS::S3::Bucket > pkl/aws/s3/bucket.pkl
pkl eval pkl/aws/s3/bucket.pkl
pkl eval test/pkl/bucket.pkl | ./rain fmt | cfn-lint

