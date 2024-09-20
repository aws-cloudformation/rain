#!/usr/local/bin/bash
set -eou pipefail

SCRIPT_DIR=$(dirname "$0")
echo "SCRIPT_DIR: ${SCRIPT_DIR}"

cd ${SCRIPT_DIR}/site

# Create the config file based on CloudFormation outputs

echo "Editing config file..."
OUTPUTS=$(rain ls webapp)
APIGW=$(echo "${OUTPUTS}" | grep RestApiInvokeURL | sed s/\ \ \ \ RestApiInvokeURL:\ /""/)

echo "APIGW: $APIGW"

ESCAPED_APIGW=$(printf '%s\n' "${APIGW}" | sed -e 's/[\/&]/\\&/g')
cat js/config-template.js | sed s/__APIGW__/"${ESCAPED_APIGW}"/ > js/config.js

echo "Config file:"
cat js/config.js

echo "Linting..."
npm run lint

echo "Building site..."
npm run build

