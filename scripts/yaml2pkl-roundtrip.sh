#!/usr/local/bin/bash
#
# $1 is the name of the yaml file
# If $2 is "modules", use modules instead of basic syntax
#

set -euo pipefail

mkdir -p /tmp/pkl

# Link Pkl modules so we can evaluate high level conversions
ln -F -s -v -h $(realpath pkl) /tmp/pkl/modules

name=$(basename ${1} | sed s/\.yaml/\.pkl/g)
pklOutput="/tmp/pkl/${name}"
yamlOutput="/tmp/pkl/$(basename ${1})"

# Default to basic
pklBasic="--pkl-basic"

if [ $# -eq 2 ]
then
    if [ "$2" == "modules" ]
    then
        pklBasic="--pkl-package modules"
    fi
fi

echo "Converting to Pkl"
echo ""
./rain fmt --pkl ${pklBasic} "$1" > "${pklOutput}"
cat "${pklOutput}"
echo ""

echo "Converting back to YAML"
echo ""
pkl eval "${pklOutput}" -f yaml | ./rain fmt > "${yamlOutput}"
cat "${yamlOutput}"
echo ""

echo "Linting"
cfn-lint "${yamlOutput}"

echo "Success!"


