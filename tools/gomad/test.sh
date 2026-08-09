#!/bin/bash
cd "${0%/*}"
set -e

unformatted=$(go run mvdan.cc/gofumpt -l .)
if [[ -n "$unformatted" ]]; then
	printf '%s\n' "$unformatted"
	exit 1
fi

mkdir -p .gomad
go build -o .gomad/gomadtool ./cmd/gomad
.gomad/gomadtool prepare-selftest

.gomad/gomadtool build-tests ./internal/tests/behavior ./nemesis
go test -ldflags=-checklinkname=0 -tags=linkname,test_dep ./...

.gomad/gomadtool build-tests -race ./internal/tests/behavior ./nemesis ./internal/tests/race/testdata
go test -ldflags=-checklinkname=0 -tags=linkname,test_dep -race ./...
