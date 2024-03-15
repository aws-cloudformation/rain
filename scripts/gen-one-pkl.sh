#!/usr/local/bin/bash

set -eou pipefail

dn=$(echo "$1" | awk -F '::' '/^AWS/ {print "pkl/" tolower($1) "/" tolower($2) }')
mkdir -p "$dn"

fn=$(echo "$1" | awk -F '::' '/^AWS/ {print "pkl/" tolower($1) "/" tolower($2) "/" tolower($3) ".pkl"}')

echo "${1}"

./rain build --pkl-class $1 > ${fn}

pkl eval ${fn}

