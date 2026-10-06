#!/bin/sh
set -eu
scratch=$(mktemp -d)
trap 'rm -rf "$scratch"' EXIT HUP INT TERM
lint_binary=$(go tool -n -modfile=tools/golangci.mod golangci-lint)
printf 'module example.com/quality-probe\n\ngo 1.27.1\n' > "$scratch/go.mod"
cp .golangci.yml "$scratch/.golangci.yml"
git -C "$scratch" init -q
# readonly, без сторонних зависимостей: каждая отрицательная проба проверяет один реальный анализатор.
probe() {
  linter=$1
  source=$2
  printf '%s\n' "$source" > "$scratch/probe.go"
  if (cd "$scratch" && "$lint_binary" run --enable-only="$linter" --output.text.path "$scratch/report" ./... >"$scratch/output" 2>&1); then
    printf 'Policy probe failed: %s accepted an invalid program\n' "$linter" >&2
    exit 1
  fi
  if ! grep -Eq "\($linter\)" "$scratch/report"; then
    printf 'Policy probe did not produce expected analyzer: %s\n' "$linter" >&2
    cat "$scratch/report" >&2
    exit 1
  fi
}
probe errcheck 'package probe; import "os"; func Remove() { os.Remove("file") }'
probe gosec 'package probe; import "crypto/tls"; func Config() *tls.Config { return &tls.Config{InsecureSkipVerify:true} }'
probe errorlint 'package probe; import "errors"; func Is(err error) bool { return err == errors.New("sentinel") }'
probe depguard 'package probe; import "io/ioutil"; func Read() ([]byte,error) { return ioutil.ReadFile("file") }'
probe nolintlint 'package probe; //nolint
func Value() int { return 1 }'
printf 'All negative policy probes passed.\n'
