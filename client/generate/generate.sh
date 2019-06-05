#!/bin/bash

cat <<EOF > cred_dumper.go
package client

// Abomination
var credDumperPython = \`
$(cat generate/cred_dumper.py)
\`
EOF
