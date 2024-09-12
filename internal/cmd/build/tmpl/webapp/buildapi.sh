#!/usr/bin/env bash

set -eou pipefail

SCRIPT_DIR=$(dirname "$0")
cd $SCRIPT_DIR/api/resources/test

GOOS=linux GOARCH=amd64 go build -o bootstrap main.go
zip ../../dist/lambda-handler.zip bootstrap

