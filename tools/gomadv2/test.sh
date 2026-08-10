#!/bin/bash
cd "${0%/*}"
set -e

unformatted=$(find . -path ./.gomad -prune -o -name '*.go' -print0 | xargs -0 gofmt -l)
if [[ -n "$unformatted" ]]; then
	printf '%s\n' "$unformatted"
	exit 1
fi

mkdir -p .gomad
go -C ../.. build -tags=test_dep -o tools/gomadv2/.gomad/gomadtool ./cmd/tools/gomad
.gomad/gomadtool prepare-selftest

.gomad/gomadtool build-tests -tags=test_dep ./internal/tests/behavior ./nemesis
GOMADTOOL="$PWD/.gomad/gomadtool" go test -ldflags=-checklinkname=0 -tags=linkname,test_dep ./...

.gomad/gomadtool build-tests -race -tags=test_dep ./internal/tests/behavior ./nemesis ./internal/tests/race/testdata
GOMADTOOL="$PWD/.gomad/gomadtool" go test -ldflags=-checklinkname=0 -tags=linkname,test_dep -race ./...
