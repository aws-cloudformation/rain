#!/usr/local/bin/bash
#
# $1 is the name of the yaml file
# If $2 is "basic", don't use modules
#

set -euo pipefail

mkdir /tmp/pkl

name=$(basename ${1} | sed s/\.yaml/\.pkl/g)
pklOutput="/tmp/pkl/${name}"
yamlOutput="/tmp/pkl/$(basename ${1})"

pklBasic="--pkl-basic"
if [ -z "$2" ]
then
    pklBasic="--pkl-package modules"
else
    if [ "$2" != "basic" ]
    then
        echo "Invalid arg"
        exit 1
    fi
fi

echo "Converting to Pkl"
echo ""
./rain fmt --pkl "${pklBasic}" "$1" > "${pklOutput}"
cat "${pklOutput}"
echo ""

echo "Converting back to YAML"
echo ""
pkl eval "${pklOutput}" -f yaml | ./rain fmt > "${yamlOutput}"
cat "${yamlOutput}"
echo ""

echo "Linting"
cfn-lint "${yamlOutput}"



