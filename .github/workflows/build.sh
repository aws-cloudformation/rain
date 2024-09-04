#!/bin/bash

set -eoux pipefail

declare -a platforms=(linux/amd64 linux/386 linux/arm linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/386)
declare -A mapping=([darwin]=darwin [linux]=linux [windows]=windows [amd64]=amd64 [386]=i386 [arm]=arm [arm64]=arm64)
declare -a CGO=(0 1)

for platform in "${platforms[@]}"; do
    os="${platform%/*}"
    arch="${platform#*/}"
    cgo_env=""

    for cgo in "${CGO[@]}"; do
      name="rain-${GITHUB_REF##*/}_${mapping[$os]}-${mapping[$arch]}"
      echo "$os $arch $cgo"

      if [ "$cgo" == "0" ]; then
          name+="-nocgo"
      fi

      echo "Building $name"
      echo "Building for $platform"

      if [ "$cgo" == "0" ]; then
          echo "nocgo"
          cgo_env="CGO_ENABLED=0"
      fi

      mkdir -p "dist/${name}"

      # We eval for CGO_ENABLED, which we don't want explicitly set if it's 1, which means we want the default behavior
      eval GOOS="$os" GOARCH="$arch" "$cgo_env" go build -buildvcs=false -ldflags=-w -o "dist/${name}/" ./cmd/rain

      cp LICENSE "dist/${name}/"
      cp README.md "dist/${name}/"

      cd dist || exit
      zip -9 -r "${name}.zip" "$name"
      rm -r "$name"
      cd - || exit
    done
done
