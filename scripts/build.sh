#!/usr/bin/env bash
set -euo pipefail

cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.."

version="$(< VERSION)"
if [[ -z "$version" || "$version" == *[[:space:]]* ]]; then
    printf '%s\n' 'The "VERSION" file must contain a nonempty version without whitespace.' >&2
    exit 1
fi

extension=''
if [[ "$(go env GOOS)" == windows ]]; then
    extension='.exe'
fi

mkdir -p dist
go build -trimpath \
    -ldflags="-X github.com/horsenuggets/tf4fmt/src/version.Version=$version" \
    -o "dist/tf4fmt$extension" ./src/cmd/tf4fmt
