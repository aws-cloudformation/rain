#!/bin/bash

set -eou pipefail

echo "Updating dependencies..."
./scripts/update.sh
go mod vendor

echo "Generating cached schemas..."
./scripts/cache-schemas.sh internal/aws/cfn/schemas

echo "Generating docs..."
go generate ./...

echo "Running tests..."
./scripts/integ.sh

echo "Finished. Don't forget to update internal/config/version.go"

