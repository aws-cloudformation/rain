#!/bin/bash

# This script will build rain for all platforms

set -e

NAME=rain
OUTPUT_DIR=dist

VERSION=$(git describe --tags)

declare -A PLATFORMS=([linux]=linux [darwin]=macos [windows]=windows)
declare -A ARCHITECTURES=([386]=i386 [amd64]=amd64 [arm]=arm [arm64]=arm64)
declare -A VARIANTS=([default]="" [nocgo]="CGO_ENABLED=0")

golint -set_exit_status ./... || exit 1

# Run tests
go vet ./... || exit 1
go test ./... || exit 1

# Run functional tests
go vet -tags=func_test ./... || exit 1
go test -tags=func_test ./... || exit 1

echo "Building $NAME $VERSION..."

for platform in ${!PLATFORMS[@]}; do
    for architecture in ${!ARCHITECTURES[@]}; do
        if [[ "$architecture" == arm* && "$platform" != "linux" ]]; then
            continue
        fi

        if [[ "$architecture" == "386" && "$platform" == "darwin" ]]; then
            continue
        fi

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

            bin_path="$OUTPUT_DIR/$full_name/$bin_name"

            eval GOOS=$platform GOARCH=$architecture ${VARIANTS[$variant]} go build -ldflags=-w -o "$bin_path" ./cmd/rain

            cp LICENSE "$OUTPUT_DIR/$full_name"
            cp README.md "$OUTPUT_DIR/$full_name"

            cd "$OUTPUT_DIR"
            zip -9 -r "${full_name}.zip" "$full_name"
            rm -r "$full_name"
            cd -
        done
    done
done

echo "All done."
