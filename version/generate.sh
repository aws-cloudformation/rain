#!/bin/bash

cat <<EOF > version.go
//go:generate bash generate.sh
package version

const NAME = "Rain"
const VERSION = "$(git describe)"
EOF
