#!/usr/local/bin/bash

declare -a platforms=(linux/amd64 linux/386 linux/arm darwin/amd64 windows/amd64 windows/386)
declare -A mapping=([darwin]=macos [linux]=linux [windows]=windows [amd64]=amd64 [386]=i386 [arm]=arm)
declare -a CGO=(0, 1)

for platform in "${platforms[@]}"; do
    os="${platform%/*}"
    arch="${platform#*/}"
    name="rain-${GITHUB_REF##*/}_${mapping[$os]}-${mapping[$arch]}"

    for cgo in "${CGO[@]}"; do
      
      if [ "$cgo" == "0" ]; then
          if [ "$arch" != "amd64" ]; then
              continue
          fi
          name+="-nocgo"
      fi

      echo "Building for $platform"
      if [ "$cgo" == "0" ]; then
          echo "nocgo"
      fi

      mkdir -p "dist/${name}"

      GOOS="$os" GOARCH="$arch" CGO_ENABLED="$cgo" go build -buildvcs=false -ldflags=-w -o "dist/${name}/" ./cmd/rain

      cp LICENSE "dist/${name}/"
      cp README.md "dist/${name}/"

      cd dist || exit
      zip -9 -r "${name}.zip" "$name"
      rm -r "$name"
      cd - || exit
    done
done
