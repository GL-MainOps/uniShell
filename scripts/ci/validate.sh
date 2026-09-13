#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

cd "$ROOT_DIR"

printf '%s\n' '===== gofmt ====='
if ! gofmt -l . | grep -q .; then
    printf '%s\n' 'gofmt: clean'
else
    printf '%s\n' 'gofmt: files require formatting' >&2
    gofmt -l .
    exit 1
fi

printf '\n%s\n' '===== tests ====='
go test ./... -count=1

printf '\n%s\n' '===== vet ====='
go vet ./...

printf '\n%s\n' '===== diff check ====='
git diff --check

printf '\n%s\n' '===== validation complete ====='
