#!/bin/bash

# This script will build rain for all platforms

set -e

NAME=rain
OUTPUT_DIR=dist

VERSION=$(git describe)

declare -A PLATFORMS=([linux]=linux [darwin]=osx [windows]=windows)
declare -A ARCHITECTURES=([386]=i386 [amd64]=amd64)
declare -A VARIANTS=([default]="" [nocgo]="CGO_ENABLED=0")

# Run tests first
golint -set_exit_status ./... || exit 1
go vet ./... || exit 1
go test ./... || exit 1

echo "Building $NAME $VERSION..."

for platform in ${!PLATFORMS[@]}; do
    for architecture in ${!ARCHITECTURES[@]}; do
        for variant in ${!VARIANTS[@]}; do
            echo "$platform/$architecture/$variant..."

            full_name="${NAME}-${VERSION}_${PLATFORMS[$platform]}-${ARCHITECTURES[$architecture]}"
            bin_name="$NAME"

            if [ "$variant" != "default" ]; then
                full_name+="-$variant"
            fi

            if [ "$platform" == "windows" ]; then
                bin_name="${NAME}.exe"
            fi

            mkdir -p "$OUTPUT_DIR/$full_name"

            eval GOOS=$platform GOARCH=$architecture ${VARIANTS[$variant]} go build -o "$OUTPUT_DIR/${full_name}/${bin_name}"

            cd "$OUTPUT_DIR"
            zip -9 -r "${full_name}.zip" "$full_name"
            rm -r "$full_name"
            cd -
        done
    done
done

echo "All done."
