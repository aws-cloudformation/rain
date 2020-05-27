#!/bin/bash

# This script will build rain for all platforms

set -e

NAME=rain
OUTPUT_DIR=dist

VERSION=$(git describe)

declare -A PLATFORMS=([linux]=linux [darwin]=osx [windows]=windows)
declare -A ARCHITECTURES=([386]=i386 [amd64]=amd64)

# Run tests first
golint ./... || exit 1
go vet ./... || exit 1
go test ./... || exit 1

echo "Building $NAME $VERSION..."

for platform in ${!PLATFORMS[@]}; do
    for architecture in ${!ARCHITECTURES[@]}; do
        echo "$platform/$architecture..."

        full_name="${NAME}-${VERSION}_${PLATFORMS[$platform]}-${ARCHITECTURES[$architecture]}"
        bin_name="$NAME"

        if [ "$platform" == "windows" ]; then
            bin_name="${NAME}.exe"
        fi

        mkdir -p "$OUTPUT_DIR/$full_name"

        GOOS=$platform GOARCH=$architecture go build -o "$OUTPUT_DIR/${full_name}/${bin_name}"

        zip -9 -r "$OUTPUT_DIR/${full_name}.zip" "$OUTPUT_DIR/$full_name"

        rm -r "$OUTPUT_DIR/$full_name"
    done
done

echo "All done."
