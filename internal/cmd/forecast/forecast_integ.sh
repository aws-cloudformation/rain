#!/bin/bash
#
# Integration test for the forecast command.
#
# Run this from the root directory
#
# ./internal/cmd/forecast/forecast_integ.sh

set -x

go build ./cmd/rain

if [ $? -ne 0 ]
then
    exit 1
fi

# First, run forecast on a template that has not been deployed, 
# to test the ability to create the stack
create_result=$(./rain forecast --debug test/templates/forecast-fail.yml forecast-integ)

echo $create_result | grep "2 checks failed out of 8 total checks"
echo "Result: $?"

# Deploy the stack, then forecast again to check updates and deletes
./rain deploy test/templates/forecast-succeed.yml forecast-integ -y

update_result=$(./rain forecast --debug test/templates/forecast-succeed.yml forecast-integ)
echo $create_result | grep "All 5 checks passed"
echo "Result: $?"

# TODO Add an object to the bucket
# Make sure we get the warning about a bucket not being empty
# Delete the object
# Make sure we still get the warning, since there is an object version
# Delete all object versions
# Check to make sure we don't get the warning

# Delete the stack
./rain delete forecast-integ -y

# TODO - Test with various roles to make sure it correctly predicts auth failures
# Roles will need to be created ahead of time ("Admin", "ReadOnly")
