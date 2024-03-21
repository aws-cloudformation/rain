#!/bin/bash
#
# Create pkl release

set -eoux pipefail


# refs/tags/v1.8.2-test1
echo "GITHUB_REF: ${GITHUB_REF}"

FIXED_REF=${GITHUB_REF##*/}
# v1.8.2-test1
echo "FIXED_REF: ${FIXED_REF}"

SEMVER=$(echo ${FIXED_REF} | sed s/^v//g)
# 1.8.2-test1
echo "SEMVER: ${SEMVER}"

curl -L -o /tmp/pkl https://github.com/apple/pkl/releases/download/0.25.2/pkl-linux-amd64
chmod +x /tmp/pkl
/tmp/pkl --version

# Modify the PklProject file

cat pkl/PklProject-template| sed s/VERSION/${SEMVER}/g > pkl/PklProject

/tmp/pkl project package pkl
gh release upload "${FIXED_REF}" .out/cloudformation@${SEMVER}/*

