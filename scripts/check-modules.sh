#!/bin/sh
set -eu
scratch=
trap 'if [ -n "$scratch" ]; then rm -rf "$scratch"; fi' EXIT INT TERM
# Проверяем актуальность module graph в отдельной копии, не меняя рабочие файлы.
check_module() {
  module_file=$1
  sum_file=${module_file%.mod}.sum
  scratch=$(mktemp -d)
  cp "$module_file" "$scratch/go.mod"
  if [ -f "$sum_file" ]; then cp "$sum_file" "$scratch/go.sum"; fi
  go -C "$scratch" mod tidy -diff
  go mod verify -modfile="$module_file"
  rm -rf "$scratch"
}
# Для основного модуля Go сам выводит diff и возвращает ошибку без записи файлов.
go mod tidy -diff
go mod verify
for module_file in tools/*.mod; do check_module "$module_file"; done
