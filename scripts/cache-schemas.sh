#!/usr/local/bin/bash
#
# $1 is the dir to use for storing schemas

set -eou pipefail

export RAIN_CACHE_DIR=$1

rain build -l | grep "^AWS::" | xargs -n1 scripts/one-schema.sh


