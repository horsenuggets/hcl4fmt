#!/usr/bin/env bash
set -euo pipefail

cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.."

rm -rf -- dist coverage
rm -f -- coverage.html coverage.out coverage.txt
