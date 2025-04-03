#!/bin/bash
# Cache one schema
# $1 is the resource type name
# $RAIN_CACHE_DIR is the root cache dir

set -eou pipefail

if [ ! -d "$RAIN_CACHE_DIR" ]; then
  echo "$RAIN_CACHE_DIR does not exist."
  exit 1
fi

# Create a directory for the schema based on the resource name
dn=$(echo "$1" | awk -F '::' '/^AWS/ {print tolower($1) "/" tolower($2) }')
mkdir -p "$RAIN_CACHE_DIR/$dn"

# The schema filename
fn=$(echo "$1" | awk -F '::' '/^AWS/ {print tolower($1) "/" tolower($2) "/" tolower($3) ".json"}')

ffn="$RAIN_CACHE_DIR/${fn}"

echo Caching "${1}" to "${ffn}"

./rain build -s --no-cache "$1" | jq > "${ffn}"

