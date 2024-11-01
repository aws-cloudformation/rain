#!/bin/bash

set -eoux pipefail

# Zip up the modules directory and create a sha256 hash
mkdir -p dist
zip -r dist/modules.zip modules
sha256sum -b dist/modules.zip > dist/modules.sha256
