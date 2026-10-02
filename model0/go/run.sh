#!/usr/bin/env bash
# The one entry point: vet, lint, exhaustiveness, tests and parity against the committed Lean dumps.
# `--views` also regenerates the generated views and fails if they differ from the goldens.
#
# Tools: golangci-lint (the server's pinned version, run with the server's config), exhaustive and
# go-check-sumtype. Set UMPIRE_GO_TOOLS to a directory holding darwin or linux builds of all three;
# see README.md for the install line.
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$here/../.." && pwd)"
tools="${UMPIRE_GO_TOOLS:-/tmp/umpire-go-tools}"
cd "$root"

pkgs=./model/go/...
models=(./model/go/worker/ ./model/go/nexuscaller/ ./model/go/standaloneactivity/)

go vet -tags test_dep "$pkgs"
"$tools/golangci-lint" run --config=.github/.golangci.yml --build-tags disable_grpc_modules,test_dep "$pkgs"
# Strict modes: a default arm must not count as covering every case, or a model switch that
# drops a variant behind `default:` would pass. The server's config sets the opposite for
# exhaustive, so these run on the model packages separately.
"$tools/exhaustive" -default-signifies-exhaustive=false "${models[@]}"
"$tools/go-check-sumtype" -default-signifies-exhaustive=false "$pkgs"
go test -count=1 -tags test_dep "$pkgs"

if [[ "${1:-}" == --views ]]; then
  rendered="$(mktemp -d)"
  trap 'rm -rf "$rendered"' EXIT
  go run -tags test_dep ./model/go/views/cmd/render -out "$rendered"
  diff -r "$here/views/testdata" "$rendered"
fi
echo "umpire-go: all checks passed"
