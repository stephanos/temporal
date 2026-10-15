#!/usr/bin/env bash
# fn-178 quiet-host window, fast variant.
#   - fast frozen qualification driver (static fail-fast, parallel lanes,
#     exclusive stress/timing phase last)
#   - concurrently: prebuild the baseline release helper and both matched
#     measurement test binaries (building is not timing); the driver's
#     exclusive phase waits for the prebuild to finish
#   - then, strictly serial and alone: baseline, candidate, baton spike
# Usage: quiet-window-fast.sh <frozen-head> <window-root (absolute, must not exist)> [--reuse PRIOR_QUAL_ROOT]
set -uo pipefail
test "$#" -ge 2
readonly head="$1" W="$2"
shift 2
reuse=("$@")
[[ "$head" =~ ^[0-9a-f]{40}$ && "$W" == /* ]] || exit 2
mkdir "$W" || exit 2
readonly GO=/Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad/.toolchain/stock/878c9b2935e2c9e6/go/bin/go
readonly FROZEN=/Users/stephan/Workspace/skunkworks/gomad-fn178-frozen-20261015
readonly MEASURE=/Users/stephan/Workspace/skunkworks/gomad-fn178-measurement-20261015
readonly PREP=/Users/stephan/Workspace/skunkworks/gomad-fn178-baseline-module-preparation-20261015.wJj6uN
readonly BASELINE_ROOT=/Users/stephan/Workspace/skunkworks/gomad-fn178-baseline-20261015.L1uv7u
readonly SPIKE=/Users/stephan/Workspace/skunkworks/gomad-fn178-baton-preparation-20261015.8K9eGY/target/release/store-handoff-spike
readonly GOENVS=(env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off GOMAD3_STOCK_GO="$GO" PATH="$(dirname "$GO"):$PATH")
readonly started="$(date +%s)"
log() { printf '%s %s\n' "$(date -u +%FT%TZ)" "$*" | tee -a "$W/window.log"; }
host() { { date -u +%FT%TZ; uptime; free -m; ps -eo pid,pcpu,rss,etime,args --sort=-pcpu | head -15; } > "$W/host-$1.txt" 2>&1; }
step() {
  local label="$1" status=0 t0
  shift
  host "$label-before"
  log "START $label"
  printf '%q ' "$@" > "$W/$label.command"
  t0="$(date +%s)"
  "$@" > "$W/$label.log" 2>&1 || status=$?
  log "EXIT $label $status $(( $(date +%s) - t0 ))s"
  host "$label-after"
  return "$status"
}
head -c 1048576 /proc/cpuinfo > "$W/cpuinfo.txt"
sha256sum "$W/cpuinfo.txt" "$GO" "$SPIKE" > "$W/inputs.sha256"
cat /proc/sys/kernel/random/boot_id > "$W/boot_id.txt"
mkdir "$W/qual" "$W/bin" "$W/baseline-helper" "$W/candidate" "$W/prebuild-go-cache"

prebuild() {
  local failed=0
  step prebuild-baseline-helper bash -c "cd '$MEASURE/tools/gomad' && timeout 5400 $(printf '%q ' "${GOENVS[@]}") TMPDIR=/tmp \
    GOMAD_WASM_TERMINAL_EXIT_HELPER_BUILD_ROOT='$W/baseline-helper' \
    '$GO' test -tags test_dep -count=1 -timeout=90m -v -run '^TestBuildTerminalExitReleaseHelper\$' ./internal/backends/wasm/wasi" || failed=1
  step prebuild-baseline-binary bash -c "cd '$MEASURE/tools/gomad' && $(printf '%q ' "${GOENVS[@]}") TMPDIR=/tmp GOCACHE='$PREP/go-cache' \
    '$GO' test -c -tags test_dep -o '$W/bin/baseline-wasi.test' ./internal/backends/wasm/wasi" || failed=1
  step prebuild-candidate-binary bash -c "cd '$FROZEN/tools/gomad' && $(printf '%q ' "${GOENVS[@]}") TMPDIR=/tmp GOCACHE='$W/prebuild-go-cache' \
    '$GO' test -c -tags test_dep,gomad_cluster_candidate -o '$W/bin/candidate-wasi.test' ./internal/backends/wasm/wasi" || failed=1
  sha256sum "$W"/bin/*.test > "$W/bin.sha256" 2>/dev/null || true
  [[ "$failed" -eq 0 ]] || touch "$W/prebuild.failed"
  touch "$W/prebuild.done"
}
prebuild &
prebuild_pid=$!

# 1. Fast frozen qualification (parallel functional phase overlaps the prebuild).
driver_status=0
step qualification env -u BASH_ENV FN178_PREBUILD_DONE="$W/prebuild.done" setsid --wait bash /home/agent/wt/fn178-tmp/qual/run-qualification-fast.sh \
  "${reuse[@]}" "$head" a1f0be7e96e2746cb9c51598766a5ae96d8552c2 "$W/qual" "$W/qual-scratch" || driver_status=$?
wait "$prebuild_pid"
log "driver_status=$driver_status prebuild_failed=$([[ -e "$W/prebuild.failed" ]] && echo 1 || echo 0)"

# 2-4. Timed executions: strictly serial, alone, last.
admission="$(find "$W/qual/helper-build" -name helper-admission.json -type f 2>/dev/null | head -1)"
baseline_admission="$(find "$W/baseline-helper" -name helper-admission.json -type f 2>/dev/null | head -1)"
if [[ -n "$baseline_admission" && -x "$W/bin/baseline-wasi.test" ]]; then
  baseline_helper="$(jq -er '.Helper' "$baseline_admission")"
  step baseline bash -c "cd '$MEASURE/tools/gomad/internal/backends/wasm/wasi' && timeout 21600 $(printf '%q ' "${GOENVS[@]}") TMPDIR=/tmp \
    GOCACHE='$PREP/go-cache' GOMAD_WASM_TOOLCHAIN_CACHE='$PREP/toolchain' \
    GOMAD_WASM_CLUSTER_MEASUREMENT_ROOT='$BASELINE_ROOT' GOMAD_WASM_TEST_HELPER='$baseline_helper' \
    GOMAD_WASM_CLUSTER_HELPER_ADMISSION='$baseline_admission' \
    '$W/bin/baseline-wasi.test' -test.v -test.count=1 -test.timeout=6h -test.run '^TestMatchedReleaseClusterBaseline\$'"
else
  log "SKIP baseline: missing helper admission or prebuilt binary"
fi
if [[ -n "$admission" && -x "$W/bin/candidate-wasi.test" ]]; then
  candidate_helper="$(jq -er '.Helper' "$admission")"
  # The driver's GOCACHE already holds this Temporal module build from the
  # clusters gate, so preparation inside the timed producer is a cache hit.
  step candidate bash -c "cd '$FROZEN/tools/gomad/internal/backends/wasm/wasi' && timeout 21600 $(printf '%q ' "${GOENVS[@]}") TMPDIR=/tmp \
    GOCACHE='$W/qual-scratch/go-cache' \
    GOMAD_WASM_CLUSTER_MEASUREMENT_ROOT='$W/candidate' GOMAD_WASM_TEST_HELPER='$candidate_helper' \
    GOMAD_WASM_CLUSTER_HELPER_ADMISSION='$admission' \
    '$W/bin/candidate-wasi.test' -test.v -test.count=1 -test.timeout=6h -test.run '^TestMatchedReleaseClusterCandidate\$'"
else
  log "SKIP candidate: missing frozen helper admission or prebuilt binary"
fi
step baton "$SPIKE" 200000 7
log "WINDOW DONE total=$(( $(date +%s) - started ))s driver_status=$driver_status"
