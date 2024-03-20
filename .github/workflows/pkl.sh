#!/bin/bash
#
# Create pkl release

set -eoux pipefail

FIXED_REF=${GITHUB_REF##*/}
echo "GITHUB_REF: ${GITHUB_REF}"
echo "FIXED_REF: ${FIXED_REF}"

curl -L -o /tmp/pkl https://github.com/apple/pkl/releases/download/0.25.2/pkl-linux-amd64
chmod +x /tmp/pkl
/tmp/pkl --version

# Modify the PklProject file

cat pkl/PklProject-template| sed s/VERSION/${FIXED_REF}/g > pkl/PklProject

/tmp/pkl project package pkl
gh release upload "${FIXED_REF}" .out/cloudformation@${FIXED_REF}/*

