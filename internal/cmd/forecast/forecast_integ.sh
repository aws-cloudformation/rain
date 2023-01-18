#!/bin/bash
#
# Integration test for the forecast command.
#
# Run this from the root directory after building:
#
# go build ./cmd/rain
# ./internal/cmd/forecast/forecast_integ.sh

# First, run forecast on a template that has not been deployed, 
# to test the ability to create the stack
create_result=$(./rain forecast test/templates/forecast.yml forecast-integ)

# Deploy the stack, then forecast again to check updates and deletes
./rain deploy test/templates/forecast.yml forecast-integ
update_result=$(./rain forecast test/templates/forecast.yml forecast-integ)

# Delete the stack
./rain delete forecast-integ

# TODO - Test with various roles to make sure it correctly predicts auth failures
# Roles will need to be created ahead of time ("Admin", "ReadOnly")
