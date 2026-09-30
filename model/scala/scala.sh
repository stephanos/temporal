#!/usr/bin/env bash
# scala-cli with an exit code that can be trusted. Through its Bloop server, scala-cli 1.17.1 prints
# a -Werror failure (a non-exhaustive match, an unused import) as "[error]" and still exits 0, and the
# class files it wrote let a following `test` run the rejected code. This wrapper fails whenever
# scala-cli fails or prints an error, so every script here sees the compiler's verdict.
set -uo pipefail
out="$(mktemp)"
trap 'rm -f "$out"' EXIT
mise exec -- scala-cli "$@" --suppress-outdated-dependency-warning 2>&1 | tee "$out" | grep -v '^\S*\[.*hint'
status=${PIPESTATUS[0]}
if [[ $status -ne 0 ]] || sed 's/\x1b\[[0-9;]*m//g' "$out" | grep -q '^\[error\]'; then
  exit 1
fi
