#!/bin/sh
set -eu
required=$(awk '$1 == "go" {print $2}' go.mod)
actual=$(go env GOVERSION)
if [ "$actual" != "go$required" ]; then
  printf 'Go %s is required, got %s. Run with GOTOOLCHAIN=go%s.\n' "$required" "$actual" "$required" >&2
  exit 1
fi
