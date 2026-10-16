#!/usr/bin/env bash
# fn-182.3 matched functional/replay cells, with explicit installed inputs.
# Usage: run-cells.sh <worktree> <evidence-root> [cells...]
set -euo pipefail
WT=$1 EV=$2; shift 2
CELLS=${*:-"onebox:success onebox:canary multi-instance:success multi-instance:canary"}
export GIT_DISCOVERY_ACROSS_FILESYSTEM=1
STOCK=${GOMAD3_STOCK_GO:?set the absolute pinned stock compiler outside the module cache}
CLUSTER_INPUTS=${FN182_CLUSTER_INPUTS:?set the fn-178 detached cluster configuration directory}
mkdir -p "$EV/tmp"
wasm=$WT/tools/gomad/internal/backends/wasm
fixture_hash=$(cat "$wasm/testdata/temporal_storage/recovery_test.go" "$wasm/testdata/temporal_broker/replication_test.go" | sha256sum | cut -d' ' -f1)
staged=.tmp/wasm-backend-temporal/temporal_broker-${fixture_hash:0:16}
mkdir -p "$WT/$staged"
cp "$wasm/testdata/temporal_storage/recovery_test.go" "$wasm/testdata/temporal_broker/replication_test.go" "$WT/$staged/"
manifest=$EV/source.sha256
git -C "$WT" ls-files -z -- . ':!.flow' | while IFS= read -r -d '' source; do
  [[ -d $WT/$source ]] && continue
  sha256sum "$(realpath -e "$WT/$source")"
done | sort -u > "$manifest"
digest=$(sha256sum "$manifest" | cut -d' ' -f1)
{ echo "head=$(git -C "$WT" rev-parse HEAD)"; echo "index_digest=sha256:$digest"; git -C "$WT" status --porcelain; sha256sum "$wasm/wasmhost/target/release/gomad3-wasmhost" "$STOCK"; } > "$EV/binding.txt"
for entry in $CELLS; do
  workload=${entry%%:*} cell=${entry#*:}
  case $workload in onebox) gate=TestActualOneboxQualification;; multi-instance) gate=TestActualTemporalClusterQualification;; esac
  label=$workload-$cell
  echo "$(date -u +%FT%TZ) START $label" >> "$EV/stages.log"
  if [[ -n ${FN182_QUALIFICATION_BINARY:-} ]]; then
    command=("$FN182_QUALIFICATION_BINARY" -test.v -test.count=1 -test.timeout=90m -test.run "^$gate\$")
  else
    command=("$STOCK" test -v -tags test_dep -count=1 -timeout=90m -run "^$gate\$" .)
  fi
  unset_env=()
  while IFS= read -r name; do unset_env+=(-u "$name"); done < <(compgen -A variable GOMAD_WASM_ || true)
  rc=0
  ( cd "$wasm/qualification" && env -u GOROOT -u GOMADSEED -u GOMAD3_CHILD_SEED -u GOEXPERIMENT -u GOOS -u GOARCH \
      "${unset_env[@]}" \
      GOENV=off GOTOOLCHAIN=local GOWORK=off GOFLAGS= TMPDIR="$EV/tmp" \
      GOMAD3_STOCK_GO=$STOCK GOMAD_WASM_STOCK_GO=$STOCK \
      GOMAD_WASM_INTEGRATED_SOURCE_MANIFEST="$manifest" GOMAD_WASM_INTEGRATED_SOURCE_SHA256=sha256:$digest \
      GOMAD_WASM_QUALIFICATION_ROOT="$EV/workloads" GOMAD_WASM_QUALIFICATION_PROFILE=cooperative GOMAD_WASM_QUALIFICATION_CELL=$cell \
      GOMAD_WASM_TEMPORAL_QUALIFICATION_INPUTS=$CLUSTER_INPUTS GOMAD_WASM_TEMPORAL_PREPARATION_SOURCE=./$staged \
      "${command[@]}" ) > "$EV/$label.log" 2>&1 || rc=$?
  echo "$(date -u +%FT%TZ) EXIT $label $rc" >> "$EV/stages.log"
  if (( rc != 0 )); then tail -30 "$EV/$label.log" >&2; exit "$rc"; fi
done
