#!/usr/bin/env bash
set -u
set -o pipefail
cd /Users/stephan/Workspace/skunkworks/gomad/temporal || exit 90
export PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:/usr/local/bin:/usr/bin:/bin
export GOENV=off GOWORK=off GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS= GOMAXPROCS=2
unset GOMADSEED GOMAD3_CHILD_SEED GOMAD3_SEED
scratch=/tmp/fn109-pack-policy-base.hc9osLjg
git rev-parse HEAD > "$scratch/head.txt"
git ls-files -z tools/gomad3 Makefile .github/.golangci.yml cmd/tools/lintcode go.mod go.sum | xargs -0 sha256sum > "$scratch/source.sha256"
sha256sum /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/gofmt /tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 /tmp/fn109-lint-tools.ZdNe1t50/errortype >> "$scratch/source.sha256"
date -u '+%Y-%m-%dT%H:%M:%SZ' > "$scratch/start.txt"
cd tools/gomad3 || exit 91
go test -count=1 -tags test_dep -json ./internal/compatibilitypack ./internal/compatibilitypack/authoring > "$scratch/tests.jsonl" 2>&1
tests_status=$?
/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 run --config=../../.github/.golangci.yml --build-tags=test_dep --timeout=10m --fix=false ./internal/compatibilitypack > "$scratch/lint.log" 2>&1
lint_status=$?
/tmp/fn109-lint-tools.ZdNe1t50/errortype -test=true ./internal/compatibilitypack > "$scratch/errortype.log" 2>&1
errortype_status=$?
cd ../.. || exit 92
sha256sum -c "$scratch/source.sha256" > "$scratch/source-check.log"
stability_status=$?
date -u '+%Y-%m-%dT%H:%M:%SZ' > "$scratch/end.txt"
printf 'tests=%s\nlint=%s\nerrortype=%s\nstability=%s\n' "$tests_status" "$lint_status" "$errortype_status" "$stability_status" > "$scratch/results.txt"
cat "$scratch/results.txt"
cat "$scratch/lint.log"
test "$tests_status" -eq 0 && test "$errortype_status" -eq 0 && test "$stability_status" -eq 0
