#!/usr/local/bin/bash
#
# Rain integration tests

set -eoux pipefail

./scripts/test.sh

./rain --profile rain ls

./rain --profile rain deploy test/templates/success.template success-test7 -y --params BucketName=ezbeardatamazon-rain-test-1

./rain --profile rain cat success-test7

./rain --profile rain logs success-test7

./rain --profile rain logs --chart success-test7

./rain --profile rain rm success-test7 -y

# Unnamed stack
./rain --profile rain deploy test/templates/success.template -y --params BucketName=ezbeardatamazon-rain-test-1
./rain --profile rain cat success
./rain --profile rain ls success
./rain --profile rain rm -y success

# Change sets
# Create a named changeset
#./rain --profile rain deploy --no-exec test/templates/success.template success-test7 success-changeset-name -y --params BucketName=ezbeardatamazon-rain-test-1
#./rain --profile rain ls -c success-test7 success-changeset-name

#./rain --profile rain rm -c -y success-test7 success-changeset-name
# This leaves the stack as review in progress, and then you can't delete it
# Seems like a bug!
# It also fails if we try to delete the stack, not just the stackset
#./rain --profile rain rm -y success-test7 

./rain --profile rain build AWS::S3::Bucket
./rain --profile rain build -l

./rain fmt test/templates/fmtfindinmap.yaml
./rain fmt test/templates/fmtmultiwithgt.yaml
./rain fmt test/templates/fmtziplinesok.yaml

./rain --profile rain pkg cft/pkg/tmpl/s3-props-template.yaml
./rain --profile rain pkg cft/pkg/tmpl/embed-template.yaml
./rain --profile rain pkg cft/pkg/tmpl/include-template.yaml
./rain --profile rain pkg cft/pkg/tmpl/s3-template.yaml
./rain --profile rain pkg cft/pkg/tmpl/s3http-template.yaml
# Given a Template with an Extension value of txt, When Packaged, Then the S3 URI ends '.txt'
./rain --profile rain pkg cft/pkg/tmpl/s3-extension-template.yaml | yq --exit-status '.Resources.Test.Properties.TheS3URI | test("\.txt$")'

# Make sure merge works
./rain merge test/templates/merge-out-1.yaml test/templates/merge-out-2.yaml

# Make sure build recommendations work
./internal/cmd/build/tmpl/scripts/validate.sh

# Make sure modules package and lint
./rain pkg -x --profile rain test/webapp/webapp.yaml | cfn-lint

# Make sure pkl generation works
./rain fmt test/templates/success.template --pkl
./rain fmt test/templates/success.template --pkl --pkl-basic
./rain fmt test/templates/condition-stringlike.yaml --pkl > test/pkl/condition-stringlike.pkl
pkl eval --project-dir test/pkl test/pkl/condition-stringlike.pkl -f yaml
./rain --profile rain pkg test/webapp/webapp.pkl | cfn-lint

