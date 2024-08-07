#!/usr/local/bin/bash
#
# Rain integration tests

set -eoux pipefail

./scripts/test.sh

./rain ls

./rain deploy test/templates/success.template success-test7 -y --params BucketName=ezbeardatamazon-rain-test-1

./rain cat success-test7

./rain logs success-test7

./rain logs --chart success-test7

./rain rm success-test7 -y

# Unnamed stack
./rain deploy test/templates/success.template -y --params BucketName=ezbeardatamazon-rain-test-1
./rain cat success
./rain ls success
./rain rm -y success

# Change sets
# Create a named changeset
#./rain deploy --no-exec test/templates/success.template success-test7 success-changeset-name -y --params BucketName=ezbeardatamazon-rain-test-1
#./rain ls -c success-test7 success-changeset-name

#./rain rm -c -y success-test7 success-changeset-name
# This leaves the stack as review in progress, and then you can't delete it
# Seems like a bug!
# It also fails if we try to delete the stack, not just the stackset
#./rain rm -y success-test7 

./rain build AWS::S3::Bucket
./rain build -l

./rain fmt test/templates/fmtfindinmap.yaml
./rain fmt test/templates/fmtmultiwithgt.yaml
./rain fmt test/templates/fmtziplinesok.yaml

./rain pkg cft/pkg/tmpl/s3-props-template.yaml
./rain pkg cft/pkg/tmpl/embed-template.yaml
./rain pkg cft/pkg/tmpl/include-template.yaml
./rain pkg cft/pkg/tmpl/s3-template.yaml
./rain pkg cft/pkg/tmpl/s3http-template.yaml

# Make sure build recommendations work
./internal/cmd/build/tmpl/scripts/validate.sh

# Make sure pkl generation works
./rain fmt test/templates/success.template --pkl
./rain fmt test/templates/success.template --pkl --pkl-basic
./rain fmt test/templates/condition-stringlike.yaml --pkl > test/pkl/condition-stringlike.pkl
pkl eval --project-dir test/pkl test/pkl/condition-stringlike.pkl -f yaml

