#!/usr/bin/env bash
set -u
set -o pipefail
export PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:/usr/local/bin:/usr/bin:/bin
export GOENV=off GOWORK=off GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS= GOMAXPROCS=2
unset GOMADSEED GOMAD3_CHILD_SEED GOMAD3_SEED
repo=/Users/stephan/Workspace/skunkworks/gomad/temporal
scratch=/tmp/fn109-canonical-base.aMBIU3D3
cd "$repo" || exit 2
git rev-parse HEAD > "$scratch/head.txt"
git ls-files -z tools/gomad3 Makefile .github/.golangci.yml cmd/tools/lintcode go.mod go.sum | xargs -0 sha256sum > "$scratch/source.sha256"
sha256sum /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/gofmt /tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 /tmp/fn109-lint-tools.ZdNe1t50/errortype >> "$scratch/source.sha256"
cd tools/gomad3 || exit 2
date -u +%FT%TZ > "$scratch/start.txt"
go test -count=1 -tags test_dep -json ./internal/canonicaljson > "$scratch/tests.jsonl" 2>&1
test_status=$?
/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 run --config=../../.github/.golangci.yml --build-tags=test_dep --timeout=10m --fix=false ./internal/canonicaljson > "$scratch/lint.log" 2>&1
lint_status=$?
date -u +%FT%TZ > "$scratch/end.txt"
cd "$repo" || exit 2
sha256sum -c "$scratch/source.sha256" > "$scratch/stability.log" 2>&1
stability_status=$?
printf 'tests_exit=%s\nlint_exit=%s\nsource_stability_exit=%s\n' "$test_status" "$lint_status" "$stability_status" > "$scratch/results.txt"
cat "$scratch/results.txt"
cat "$scratch/lint.log"
exit "$test_status"
