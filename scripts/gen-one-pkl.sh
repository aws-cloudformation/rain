#!/bin/bash

set -eou pipefail

# Create a directory for the pkl class based on the resource name
dn=$(echo "$1" | awk -F '::' '/^AWS/ {print "pkl/" tolower($1) "/" tolower($2) }')
mkdir -p "$dn"

# The pkl class filename
fn=$(echo "$1" | awk -F '::' '/^AWS/ {print "pkl/" tolower($1) "/" tolower($2) "/" tolower($3) ".pkl"}')

echo "${1}"

./rain build --pkl-class $1 > ${fn} || exit 255

pkl eval ${fn} || exit 255

