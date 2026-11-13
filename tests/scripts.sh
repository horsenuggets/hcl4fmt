#!/usr/bin/env bash
set -euo pipefail

root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)"
temporary="$(mktemp -d)"
trap 'rm -rf -- "$temporary"' EXIT
fixture="$temporary/repository with spaces"
mkdir -p "$fixture/scripts" "$fixture/src" "$temporary/dist"
cp "$root"/scripts/*.sh "$fixture/scripts/"
printf '%s\n' '0.0.0' > "$fixture/VERSION"
export script_test_log="$temporary/commands"
export script_test_goos=linux
export script_test_unformatted=''

# Exported functions replace only external Go tools in child Bash processes.
go() {
    if [[ "$*" == 'env GOOS' ]]; then
        printf '%s\n' "$script_test_goos"
    else
        printf '%s\n' "$PWD|${GOOS-}|${GOARCH-}|${CGO_ENABLED-}|$*" >> "$script_test_log"
    fi
}
gofmt() {
    printf '%s\n' "$PWD|$*" >> "$script_test_log"
    if [[ "$1" == -l && -n "$script_test_unformatted" ]]; then
        printf '%s\n' "$script_test_unformatted"
    fi
}
export -f go gofmt

expect_status() {
    local expected="$1" actual=0
    shift
    "$@" > "$temporary/output" 2>&1 || actual=$?
    if [[ "$actual" -ne "$expected" ]]; then
        printf 'Expected exit status %s but received %s.\n' "$expected" "$actual" >&2
        exit 1
    fi
}
assert_contains() {
    if [[ "$1" != *"$2"* ]]; then
        printf 'Expected output to contain "%s".\n' "$2" >&2
        exit 1
    fi
}

# Run from outside the fixture to verify path resolution and quoted paths.
cd -- "$temporary"
expect_status 0 bash "$fixture/scripts/build.sh"
log="$(< "$script_test_log")"
assert_contains "$log" "$fixture|"
assert_contains "$log" '-X github.com/horsenuggets/hcl4fmt/src/version.Version=0.0.0'
assert_contains "$log" '-o dist/hcl4fmt ./src/cmd/hcl4fmt'

export script_test_goos=windows
expect_status 0 bash "$fixture/scripts/build.sh"
log="$(< "$script_test_log")"
assert_contains "$log" '-o dist/hcl4fmt.exe ./src/cmd/hcl4fmt'

: > "$script_test_log"
expect_status 0 bash "$fixture/scripts/cross.sh"
map_count=0
while IFS= read -r _; do
    ((map_count += 1))
done < "$script_test_log"
[[ "$map_count" -eq 6 ]]
log="$(< "$script_test_log")"
for os in darwin linux windows; do
    for arch in amd64 arm64; do
        suffix=''
        [[ "$os" != windows ]] || suffix='.exe'
        assert_contains "$log" "$fixture|$os|$arch|0|build"
        assert_contains "$log" "-o dist/hcl4fmt-$os-$arch$suffix ./src/cmd/hcl4fmt"
    done
done

for script in build cross; do
    for version in '' 'invalid version' $'two\nlines'; do
        printf '%s\n' "$version" > "$fixture/VERSION"
        : > "$script_test_log"
        expect_status 1 bash "$fixture/scripts/$script.sh"
        [[ ! -s "$script_test_log" ]]
        output="$(< "$temporary/output")"
        assert_contains "$output" 'The "VERSION" file must contain'
    done
done

expect_status 0 bash "$fixture/scripts/fmt.sh"
log="$(< "$script_test_log")"
assert_contains "$log" "$fixture|-w src"
expect_status 0 bash "$fixture/scripts/fmt.sh" --check
export script_test_unformatted='src/example.go'
expect_status 1 bash "$fixture/scripts/fmt.sh" --check
expect_status 2 bash "$fixture/scripts/fmt.sh" --unknown
expect_status 2 bash "$fixture/scripts/fmt.sh" --check extra

mkdir -p "$fixture/coverage"
printf '%s\n' 'keep' > "$temporary/dist/sentinel"
printf '%s\n' 'coverage' > "$fixture/coverage.out"
expect_status 0 bash "$fixture/scripts/clean.sh"
[[ ! -e "$fixture/dist" && ! -e "$fixture/coverage" && ! -e "$fixture/coverage.out" ]]
[[ -f "$temporary/dist/sentinel" ]]
expect_status 0 bash "$fixture/scripts/clean.sh"
printf '%s\n' 'All script checks passed.'
