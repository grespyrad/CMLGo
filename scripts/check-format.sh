#!/bin/sh
set -eu
report=$(mktemp)
trap 'rm -f "$report"' EXIT INT TERM
go tool -modfile=tools/golangci.mod golangci-lint fmt --diff > "$report"
if [ -s "$report" ]; then
 cat "$report"
 printf 'Formatting differs: run task fmt.
' >&2
 exit 1
fi
