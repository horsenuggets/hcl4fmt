#!/usr/bin/env bash
set -euo pipefail

cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.."

version="$(< VERSION)"
if [[ -z "$version" || "$version" == *[[:space:]]* ]]; then
    printf '%s\n' 'The "VERSION" file must contain a nonempty version without whitespace.' >&2
    exit 1
fi

mkdir -p dist
for os in darwin linux windows; do
    for arch in amd64 arm64; do
        extension=''
        if [[ "$os" == windows ]]; then
            extension='.exe'
        fi
        CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -trimpath \
            -ldflags="-X github.com/horsenuggets/tf4fmt/src/version.Version=$version" \
            -o "dist/tf4fmt-$os-$arch$extension" ./src/cmd/tf4fmt
    done
done
