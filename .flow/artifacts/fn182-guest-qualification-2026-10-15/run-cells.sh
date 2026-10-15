#!/usr/bin/env bash
# fn-182.3 functional/replay cells (provisional; timing is non-acceptance on a shared host).
# Usage: run-cells.sh <worktree> <evidence-root> [cells...]
set -uo pipefail
WT=$1 EV=$2; shift 2
CELLS=${*:-"onebox:success onebox:canary multi-instance:success multi-instance:canary"}
export GIT_DISCOVERY_ACROSS_FILESYSTEM=1
STOCK=/Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad/.toolchain/stock/878c9b2935e2c9e6/go/bin/go
CLUSTER_INPUTS=/Users/stephan/Workspace/skunkworks/gomad-fn176-qualification-live-20261015.1hg1RG/clusters
mkdir -p "$EV/tmp"
wasm=$WT/tools/gomad/internal/backends/wasm
fixture_hash=$(cat "$wasm/testdata/temporal_storage/recovery_test.go" "$wasm/testdata/temporal_broker/replication_test.go" | sha256sum | cut -d' ' -f1)
staged=.tmp/wasm-backend-temporal/temporal_broker-${fixture_hash:0:16}
mkdir -p "$WT/$staged"
cp "$wasm/testdata/temporal_storage/recovery_test.go" "$wasm/testdata/temporal_broker/replication_test.go" "$WT/$staged/"
digest=$(git -C "$WT" ls-files -s | sha256sum | cut -d' ' -f1)
{ echo "head=$(git -C "$WT" rev-parse HEAD)"; echo "index_digest=sha256:$digest"; git -C "$WT" status --porcelain; sha256sum "$wasm/wasmhost/target/release/gomad3-wasmhost" "$STOCK"; } > "$EV/binding.txt"
for entry in $CELLS; do
  workload=${entry%%:*} cell=${entry#*:}
  case $workload in onebox) gate=TestActualOneboxQualification;; multi-instance) gate=TestActualTemporalClusterQualification;; esac
  label=$workload-$cell
  echo "$(date -u +%FT%TZ) START $label" >> "$EV/stages.log"
  ( cd "$wasm" && env -u GOROOT -u GOMADSEED -u GOMAD3_CHILD_SEED -u GOEXPERIMENT -u GOOS -u GOARCH \
      $(compgen -A variable GOMAD_WASM_ | sed 's/^/-u /') \
      GOENV=off GOTOOLCHAIN=local GOWORK=off GOFLAGS= TMPDIR="$EV/tmp" \
      GOMAD3_STOCK_GO=$STOCK GOMAD_WASM_STOCK_GO=$STOCK \
      GOMAD_WASM_INTEGRATED_SOURCE_SHA256=sha256:$digest \
      GOMAD_WASM_QUALIFICATION_ROOT="$EV/workloads" GOMAD_WASM_QUALIFICATION_PROFILE=cooperative GOMAD_WASM_QUALIFICATION_CELL=$cell \
      GOMAD_WASM_TEMPORAL_QUALIFICATION_INPUTS=$CLUSTER_INPUTS GOMAD_WASM_TEMPORAL_PREPARATION_SOURCE=./$staged \
      $STOCK test -v -tags test_dep -count=1 -timeout=90m -run "^$gate\$" ./qualification ) > "$EV/$label.log" 2>&1
  rc=$?
  echo "$(date -u +%FT%TZ) EXIT $label $rc" >> "$EV/stages.log"
done
