#!/usr/bin/env bash
set -euo pipefail

cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.."

if [[ $# -gt 1 || ( $# -eq 1 && "$1" != --check ) ]]; then
    printf '%s\n' '[Usage] scripts/fmt.sh [--check]' >&2
    exit 2
fi

if [[ $# -eq 1 ]]; then
    unformatted="$(gofmt -l src)"
    if [[ -n "$unformatted" ]]; then
        printf '%s\n' 'The following Go files need formatting.' "$unformatted" >&2
        exit 1
    fi
else
    gofmt -w src
fi
