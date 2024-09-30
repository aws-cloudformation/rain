#!/usr/local/bin/bash
set -eou pipefail

SCRIPT_DIR=$(dirname "$0")
echo "SCRIPT_DIR: ${SCRIPT_DIR}"

cd ${SCRIPT_DIR}/site

# Create the config file based on CloudFormation outputs

echo "Editing config file..."
# TODO: "webapp" might not be correct depending on the user-supplied stack name
# This needs to be completely automated...
# If someone downloads this solution and types `rain deploy -s webapp.yaml my-web-app`, 
# We need to fix "webapp" below to be "my-web-app"
# Maybe `Run: buildsite.sh ${AWS::StackName}` and rain replaces it?
OUTPUTS=$(rain ls webapp)
APIGW=$(echo "${OUTPUTS}" | grep RestApiInvokeURL | sed s/\ \ \ \ RestApiInvokeURL:\ /""/)

echo "APIGW: $APIGW"

ESCAPED_APIGW=$(printf '%s\n' "${APIGW}" | sed -e 's/[\/&]/\\&/g')
cat js/config-template.js | sed s/__APIGW__/"${ESCAPED_APIGW}"/ > js/config.js

#echo "Config file:"
#cat js/config.js

echo "Linting..."
npm run lint

echo "Building site..."
npm run build

