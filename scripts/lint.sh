#!/usr/bin/env bash
set -euo pipefail

cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.."

scripts/fmt.sh --check
go vet ./...
golangci-lint run
shellcheck scripts/*.sh tests/scripts.sh
