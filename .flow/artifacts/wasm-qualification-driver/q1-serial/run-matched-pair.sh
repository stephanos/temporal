#!/usr/bin/env bash
# q1 matched-release pair: fresh baseline (51604889c9, retained admitted 1.94.1 helper)
# then fresh candidate (frozen q1 e50c6476fb, q1-admitted 1.99.1 helper), same host/window.
# Producers, admission checks (incl. 8 GiB MemAvailable boundaries), seeds and
# repetitions are unchanged; this script only binds paths.
set -euo pipefail
repo=/home/agent/wt/q1
base=/home/agent/wt/fn189-cluster-baseline
root=/home/agent/wt/evidence/q1/matched-pair
qual=/home/agent/wt/evidence/q1/qual
freeze=/home/agent/wt/evidence/q1/freeze
driver=/home/agent/wt/evidence/q1/driver
stock=/Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad/.toolchain/stock/2198cf19890232fc/go/bin/go
baseline_admission=/home/agent/wt/evidence/fn189/matched-release-current-2026-10-18/baseline-build/release-helper-3849881496/admitted-helper-815678292/helper-admission.json
export GIT_DISCOVERY_ACROSS_FILESYSTEM=1
rg -q 'PASS qualification-driver' "$qual/stages.log"
mapfile -t admissions < <(find "$qual/helper-build" -name helper-admission.json -type f)
test "${#admissions[@]}" = 1
candidate_admission="${admissions[0]}"
candidate_helper="$(jq -er .Helper "$candidate_admission")"
test "$(git -C "$base" rev-parse HEAD)" = 51604889c9a9062f981cd02aa3daf04ca55fd940
test -z "$(git -C "$base" status --porcelain --untracked-files=all)"
python3 "$driver/frozen.py" "$repo" "$freeze"
test ! -e "$root"
mkdir -p "$root/baseline" "$root/temp"
export TMPDIR="$root/temp"
export GOENV=off GOTOOLCHAIN=local GOWORK=off GOFLAGS=-p=1 GOMAXPROCS=2 CARGO_BUILD_JOBS=1 RUST_TEST_THREADS=1
export GOCACHE=/home/agent/wt/q1-scratch/go-cache
unset GOROOT RUSTFLAGS CARGO_ENCODED_RUSTFLAGS CARGO_TARGET_DIR RUSTC RUSTC_WRAPPER RUSTC_WORKSPACE_WRAPPER LLVM_PROFILE_FILE GOOS GOARCH GOEXPERIMENT RUSTUP_TOOLCHAIN
while IFS= read -r variable; do unset "$variable"; done < <(compgen -A variable GOMAD_)
log_phase() { printf '%s %s mem_avail_kb=%s load=%s\n' "$(date -u +%FT%TZ)" "$1" "$(awk '/MemAvailable/{print $2}' /proc/meminfo)" "$(cut -d' ' -f1-3 /proc/loadavg)" >> "$root/phases.log"; }
# Baseline
export GOMAD3_STOCK_GO="$stock"
export GOMAD_WASM_CLUSTER_HELPER_ADMISSION="$baseline_admission"
export GOMAD_WASM_TEST_HELPER="$(jq -er .Helper "$baseline_admission")"
test "$(sha256sum "$GOMAD_WASM_TEST_HELPER" | cut -d ' ' -f1)" = "$(jq -er .HelperSHA256 "$baseline_admission")"
export GOMAD_WASM_CLUSTER_MEASUREMENT_ROOT="$root/baseline"
cd "$base/tools/gomad"
printf '%q ' "$stock" test -v -tags test_dep -count=1 -timeout=6h -run '^TestMatchedReleaseClusterBaseline$' ./internal/backends/wasm/wasi > "$root/baseline.command"; printf '\n' >> "$root/baseline.command"
log_phase "baseline START"
status=0
"$stock" test -v -tags test_dep -count=1 -timeout=6h -run '^TestMatchedReleaseClusterBaseline$' ./internal/backends/wasm/wasi > "$root/baseline.log" 2>&1 || status=$?
printf '%s\n' "$status" > "$root/baseline.exit"
log_phase "baseline EXIT $status"
test "$status" -eq 0
rg -q '^--- PASS: TestMatchedReleaseClusterBaseline ' "$root/baseline.log"
test -z "$(git -C "$base" status --porcelain --untracked-files=all)"
while IFS= read -r variable; do unset "$variable"; done < <(compgen -A variable GOMAD_)
unset GOMAD3_STOCK_GO
# Candidate (authentic CLI; command-owned admission boundaries)
cd "$repo"
sha256sum -c "$qual/wasmmeasure.sha256" > /dev/null
sha256sum -c "$qual/compiler.sha256" > /dev/null
test "$(sha256sum "$candidate_helper" | cut -d ' ' -f1)" = "$(jq -er .HelperSHA256 "$candidate_admission")"
python3 "$driver/helper-files.py" "$candidate_helper"
command=("$qual/bin/wasmmeasure" matched-release-candidate --repo "$repo" --go "$stock"
  --stock-go "$stock" --test-helper "$candidate_helper" --cluster-helper-admission "$candidate_admission"
  --cluster-measurement-root "$root/candidate" --prototype-parallel 1 --prototype-heavy 1)
mkdir "$root/candidate"
printf '%q ' "${command[@]}" > "$root/candidate.command"; printf '\n' >> "$root/candidate.command"
log_phase "candidate START"
status=0
"${command[@]}" > "$root/candidate.log" 2>&1 || status=$?
printf '%s\n' "$status" > "$root/candidate.exit"
log_phase "candidate EXIT $status"
python3 "$driver/frozen.py" "$repo" "$freeze"
exit "$status"
