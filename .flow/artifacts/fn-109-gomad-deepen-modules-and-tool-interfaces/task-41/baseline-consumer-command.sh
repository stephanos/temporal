#!/usr/bin/env bash
set -u
set -o pipefail
export PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:/usr/local/bin:/usr/bin:/bin
export GOENV=off GOWORK=off GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS= GOMAXPROCS=2
unset GOMADSEED GOMAD3_CHILD_SEED GOMAD3_SEED
repo=/Users/stephan/Workspace/skunkworks/gomad/temporal
scratch=/tmp/fn109-canonical-base.aMBIU3D3
cd "$repo" || exit 2
sha256sum -c "$scratch/source.sha256" > "$scratch/consumer-stability-before.log" 2>&1 || exit 2
cd tools/gomad3 || exit 2
go test -count=1 -tags test_dep -json ./record ./world ./internal/compatibilitypack ./internal/compatibilitypack/authoring > "$scratch/consumer-tests.jsonl" 2>&1
consumer_status=$?
go test -count=1 -tags test_dep -json ./internal/gomadtool/architecture > "$scratch/architecture-tests.jsonl" 2>&1
architecture_status=$?
go test -count=1 -tags test_dep -json . -run '^(TestPackageArchitecture|TestPureModulesHaveNoHostEffects|TestPackageOwnershipEdges)$' > "$scratch/boundary-tests.jsonl" 2>&1
boundary_status=$?
make validate > "$scratch/validate.log" 2>&1
validate_status=$?
cd "$repo" || exit 2
sha256sum -c "$scratch/source.sha256" > "$scratch/consumer-stability-after.log" 2>&1
stability_status=$?
printf 'consumer_exit=%s\narchitecture_exit=%s\nboundary_exit=%s\nvalidate_exit=%s\nsource_stability_exit=%s\n' "$consumer_status" "$architecture_status" "$boundary_status" "$validate_status" "$stability_status" > "$scratch/consumer-results.txt"
cat "$scratch/consumer-results.txt"
exit "$stability_status"
