#!/usr/local/bin/bash
#
# $1 is the dir to use for storing schemas

set -eou pipefail

export RAIN_CACHE_DIR=$1

cat internal/cmd/forecast/all-types.txt | xargs -n1 scripts/one-schema.sh


