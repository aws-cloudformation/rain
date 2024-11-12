#!/bin/bash

set -eoux pipefail

# Zip up the modules directory and create a sha256 hash
mkdir -p dist
cd modules
zip -r ../dist/modules.zip *
sha256sum -b ../dist/modules.zip | cut -d " " -f 1  > ../dist/modules.sha256
