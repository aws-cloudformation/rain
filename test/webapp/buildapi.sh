#!/usr/bin/env bash

set -eou pipefail

SCRIPT_DIR=$(dirname "$0")
echo "SCRIPT_DIR: ${SCRIPT_DIR}"

echo "Building test..."
cd $SCRIPT_DIR/api/resources/test
GOOS=linux GOARCH=amd64 go build -o bootstrap main.go
mkdir -p ../../dist/test
zip ../../dist/test/lambda-handler.zip bootstrap

echo "Building jwt..."
cd ../jwt
GOOS=linux GOARCH=amd64 go build -o bootstrap main.go
mkdir -p ../../dist/jwt
zip ../../dist/jwt/lambda-handler.zip bootstrap

