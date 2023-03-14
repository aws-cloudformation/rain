#!/bin/bash

set -eoux pipefail

go build ./cmd/rain
staticcheck ./...
go vet ./...

