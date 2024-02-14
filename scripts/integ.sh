#!/usr/local/bin/bash
#
# Rain integration tests

set -eoux pipefail

./scripts/test.sh

./rain ls

./rain deploy test/templates/success.template success-test -y --params BucketName=ezbeardatamazon-rain-test-1

./rain cat success-test

./rain logs success-test

./rain logs --chart success-test

./rain rm success-test -y

./rain build AWS::S3::Bucket
./rain build -l

./rain fmt test/templates/fmtfindinmap.yaml
./rain fmt test/templates/fmtmultiwithgt.yaml
./rain fmt test/templates/fmtziplinesok.yaml

./rain pkg cft/pkg/tmpl/s3-props-template.yaml

# Make sure build recommendations work
./internal/cmd/build/tmpl/scripts/validate.sh

