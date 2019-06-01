#!/bin/bash

# This script will build rain for all platforms

# Run tests first

NAME=rain
VERSION=$(git describe --abbrev=0)

declare -A PLATFORMS=([linux]=linux [darwin]=osx [windows]=windows)
declare -A ARCHITECTURES=([386]=i386 [amd64]=amd64)

go vet ./... || exit 1

go test ./... || exit 1

echo "Building $NAME"

for platform in ${!PLATFORMS[@]}; do
    for architecture in ${!ARCHITECTURES[@]}; do
        echo "... $platform $architecture..."

        full_name=${NAME}-${VERSION}_${PLATFORMS[$platform]}-${ARCHITECTURES[$architecture]}
        bin_name=$NAME

        if [ "$platform" == "windows" ]; then
            bin_name=${NAME}.exe
        fi

        mkdir $full_name

        GOOS=$platform GOARCH=$architecture go build -o ${full_name}/${bin_name}

        zip -9 -r ${full_name}.zip $full_name

        rm -r $full_name
    done
done

echo "All done."
