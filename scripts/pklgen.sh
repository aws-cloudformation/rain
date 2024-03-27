#!/usr/local/bin/bash

set -eou pipefail

echo "Building rain..."
./scripts/build.sh

echo "Generating go pkl modules..."
# We don't actually need these for anything yet...

pkl-gen-go pkl/cloudformation.pkl --base-path github.com/aws-cloudformation/rain
pkl-gen-go pkl/template.pkl --base-path github.com/aws-cloudformation/rain

echo "Building pkl classes..."

# shellcheck disable=SC2002
rain build -l | grep "^AWS::" | xargs -n1 scripts/gen-one-pkl.sh

echo "Testing patterns..."
./scripts/pkl-patterns.sh

echo "Success!"
