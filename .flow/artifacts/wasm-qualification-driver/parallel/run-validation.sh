#!/usr/bin/env bash
# Parallel WASM validation ladder (owner limit 2026-10-19: one hour wall clock).
# Same stages, commands, budgets, oracles and post-checks as
# ../q1-serial/run-qualification.sh minus the retired native-* stages, run as a
# dependency DAG under scheduler.py. Matched-release measurement is run-perf.sh.
#
# Usage:
#   run-validation.sh [options] HEAD LINT_BASE FRESH_EVIDENCE FRESH_NATIVE_SCRATCH
#   run-validation.sh --plan [options]      DAG, equivalence check and wall-time estimate; runs nothing
# Options:
#   --config FILE          candidate binding (default config/q1.env)
#   --allow-background     let `bg` stages overlap exclusive ones (owner decision; default off)
#   --deadline-s N         wall-clock limit in seconds (default 3600)
#   --stop-at-deadline     start no stage after the limit (default: finish, exit 3)
#   --mem-margin-gib N     memory cap = MemAvailable at start minus N (default 2)
# Exit: 0 pass; 1 stage failure; 2 setup refusal; 3 wall-clock limit exceeded
#       (the run still completes so its evidence is whole); 4 stage-set/command/
#       environment equivalence failure; 130 interrupted.
set -euo pipefail
QUALDRV_DIR="$(cd -- "${BASH_SOURCE[0]%/*}" && pwd -P)"
export QUALDRV_DIR
QUALDRV_CONFIG="$QUALDRV_DIR/config/q1.env"
plan=false background=() deadline_s=3600 mem_margin=2
while [[ "${1:-}" == --* ]]; do
  case "$1" in
    --plan) plan=true; shift;;
    --config) QUALDRV_CONFIG="$(cd -- "$(dirname "$2")" && pwd -P)/$(basename "$2")"; shift 2;;
    --allow-background) background+=(--allow-background); shift;;
    --stop-at-deadline) background+=(--stop-at-deadline); shift;;
    --deadline-s) deadline_s="$2"; shift 2;;
    --mem-margin-gib) mem_margin="$2"; shift 2;;
    *) echo "unknown option $1" >&2; exit 2;;
  esac
done
export QUALDRV_CONFIG
reference="$QUALDRV_DIR/../q1-serial/run-qualification.sh"
if [[ "$plan" == true ]]; then
  python3 "$QUALDRV_DIR/stage_set.py" static --reference "$reference" --lib "$QUALDRV_DIR/lib.sh" --dag "$QUALDRV_DIR/dag.tsv"
  column -t -s $'\t' < <(grep -v '^#' "$QUALDRV_DIR/dag.tsv" | cut -f1-10)
  for mode in strict background; do
    flags=(); [[ "$mode" == strict ]] || flags=(--allow-background)
    python3 "$QUALDRV_DIR/scheduler.py" simulate --dag "$QUALDRV_DIR/dag.tsv" --cores 12 --host-mem-gib 19 \
      --mem-margin-gib 4 --deadline-s "$deadline_s" "${flags[@]}" || true
  done
  exit 0
fi

set -o noclobber
shopt -s nullglob dotglob
export LC_ALL=C
test "$#" -eq 4
run_started="$(date +%s)"
export QUALDRV_HEAD="$1" QUALDRV_LINT_BASE="$2" QUAL_ROOT="$3" QUALDRV_SCRATCH="$4"
unset QUALDRV_STAGE QUALDRV_LANE
# shellcheck source=lib.sh
source "$QUALDRV_DIR/lib.sh"
qual_context
test -d "$QUAL_ROOT" && test ! -e "$QUAL_ROOT/stages.log"
sched_status=-1
finish() {
  # Always record wall time and the critical path; enforce the limit; PASS last.
  local status=$? wall
  trap - EXIT
  set +e
  wall=$(( $(date +%s) - run_started ))
  [[ -s "$QUAL_ROOT/sched/schedule.json" ]] || printf '{"wall_s":null,"critical_path":[],"failed":[],"not_run":[]}\n' >| "$QUAL_ROOT/sched/schedule.json"
  if [[ "$status" -eq 0 && "$wall" -gt "$deadline_s" ]]; then
    printf '%s FAIL wall-clock limit exceeded (%ds > %ds)\n' "$(date -u +%FT%TZ)" "$wall" "$deadline_s" | tee -a "$QUAL_ROOT/stages.log"
    status=3
  fi
  jq -n --argjson wall "$wall" --argjson limit "$deadline_s" --argjson scheduler "$sched_status" --argjson exit "$status" \
    --slurpfile schedule "$QUAL_ROOT/sched/schedule.json" \
    '{wall_seconds:$wall,limit_seconds:$limit,driver_exit:$exit,scheduler_exit:$scheduler,scheduler_wall_seconds:$schedule[0].wall_s,
      critical_path:$schedule[0].critical_path,failed:$schedule[0].failed,not_run:$schedule[0].not_run}' >| "$QUAL_ROOT/wall-time.json" || status=1
  printf '%s WALL %ds limit=%ds scheduler_exit=%d driver_exit=%d\n' "$(date -u +%FT%TZ)" "$wall" "$deadline_s" "$sched_status" "$status" | tee -a "$QUAL_ROOT/stages.log"
  if [[ "$status" -eq 0 ]]; then
    printf '%s PASS qualification-driver module=%s matched-release=perf-entry-point\n' "$(date -u +%FT%TZ)" "$module_name" | tee -a "$QUAL_ROOT/stages.log"
  fi
  printf 'driver_exit=%d\n' "$status" >| "$QUAL_ROOT/driver-exit.txt"
  exit "$status"
}
trap finish EXIT
qual_base_env
test "$(stat -f -c %T "$(dirname "$scratch")")" != fuseblk
case "$(stat -f -c %T "$(dirname "$scratch")")" in *fuse*|virtiofs) exit 2;; esac
# Concurrent stages keep evidence that the serial order pruned in between.
for volume in "$QUAL_ROOT" "$(dirname "$scratch")"; do
  (( $(df --output=avail -k "$volume" | tail -1) >= 12 * 1024 * 1024 )) || { echo "less than 12 GiB free on $volume" >&2; exit 2; }
done
mkdir "$scratch"
mkdir "$QUAL_ROOT/state" "$QUAL_ROOT/sched" "$QUAL_ROOT/driver" "$QUAL_ROOT/driver/parallel" "$QUAL_ROOT/driver/q1-serial"

# Run from a recorded copy of the driver, so edits during the hour cannot change it.
cp -- "$QUALDRV_DIR"/*.sh "$QUALDRV_DIR"/*.py "$QUALDRV_DIR/dag.tsv" "$QUAL_ROOT/driver/parallel/"
cp -- "$QUALDRV_CONFIG" "$QUAL_ROOT/driver/parallel/candidate.env"
cp -- "$reference" "$QUAL_ROOT/driver/q1-serial/run-qualification.sh"
(cd "$QUAL_ROOT/driver" && find . -type f -print0 | sort -z | xargs -0 sha256sum) > "$QUAL_ROOT/driver.sha256"
cmp "$QUALDRV_DIR/lib.sh" "$QUAL_ROOT/driver/parallel/lib.sh"
QUALDRV_DIR="$QUAL_ROOT/driver/parallel"
QUALDRV_CONFIG="$QUALDRV_DIR/candidate.env"
reference="$QUAL_ROOT/driver/q1-serial/run-qualification.sh"
driver_dir="$QUALDRV_DIR"
export QUALDRV_DIR QUALDRV_CONFIG

# Stage-set, command and environment equivalence before anything runs.
python3 "$QUALDRV_DIR/stage_set.py" static --reference "$reference" --lib "$QUALDRV_DIR/lib.sh" \
  --dag "$QUALDRV_DIR/dag.tsv" > "$QUAL_ROOT/stage-set.json" || exit 4

cd "$QUAL_REPO"
check_source
test -d "$module"
if [[ "$module_name" == gomad ]]; then test ! -e "$QUAL_REPO/tools/gomad3"; fi
test -x "$qualification_go"
rg -Fxq "module go.temporal.io/server/tools/$module_name" "$module/go.mod"
git merge-base --is-ancestor "$lint_base" "$expected_head"
# Exclusive creation makes the driver one-shot and preserves failed attempts.
mkdir "$QUAL_ROOT/build-temp" "$GOCACHE" "$CARGO_TARGET_DIR"
printf '%s\n' "$expected_head" > "$QUAL_ROOT/revision"
printf '%s\n' "$lint_base" > "$QUAL_ROOT/lint-base"
git status --short > "$QUAL_ROOT/initial-status.log"
df -h "$QUAL_ROOT" > "$QUAL_ROOT/host-volume.log"
uname -a > "$QUAL_ROOT/host.log"
capture_host_cpu initial-host
"$qualification_go" version > "$QUAL_ROOT/go-version.log"
rg -q 'go1\.27\.1 linux/arm64$' "$QUAL_ROOT/go-version.log"
RUSTC="$(rustup which --toolchain 1.99.1 rustc)"
export RUSTC
"$RUSTC" -vV > "$QUAL_ROOT/rust-version.log"
rg -q '^rustc 1\.99\.1 ' "$QUAL_ROOT/rust-version.log"
cargo +1.99.1 --version > "$QUAL_ROOT/cargo-version.log"
sha256sum -- "$qualification_go" "$RUSTC" > "$QUAL_ROOT/compiler.sha256"
printf '%s\n' "$RUSTC" > "$QUAL_ROOT/state/rustc.path"

# Clean frozen worktree: manifest = every tracked regular non-.flow file.
cp -- "$freeze_root/source.sha256" "$QUAL_ROOT/source.sha256"
cp -- "$freeze_root/source-symlinks.json" "$QUAL_ROOT/source-symlinks.json"
cp -- "$freeze_root/preflight.json" "$QUAL_ROOT/preflight.json"
expected_source="$(digest_file "$QUAL_ROOT/source.sha256")"
test "$expected_source" = "$EXPECTED_SOURCE_SHA256"
source_manifest_ready=true
printf 'sha256:%s\n' "$expected_source" > "$QUAL_ROOT/source.digest"
check_source
mkdir "$QUAL_ROOT/bin"
# Concurrent stages stage Temporal fixtures beneath this ignored directory and
# create its missing parents with a non-idempotent Mkdir; create it once here.
git check-ignore -q .tmp/wasm-backend-temporal/probe
mkdir -p .tmp/wasm-backend-temporal

# ------------------------------------------------------------ parallel stages
python3 "$QUALDRV_DIR/scheduler.py" run --dag "$QUALDRV_DIR/dag.tsv" --state-dir "$QUAL_ROOT/sched" \
  --cores "$(nproc)" --mem-margin-gib "$mem_margin" --deadline-epoch "$((run_started + deadline_s))" \
  "${background[@]}" -- bash "$QUALDRV_DIR/stage.sh" &
sched_pid=$!
# Forward termination: the scheduler stops every running stage's process group.
trap 'kill -TERM "$sched_pid" 2>/dev/null' TERM INT HUP
set +e
wait "$sched_pid"; sched_status=$?
while kill -0 "$sched_pid" 2>/dev/null; do wait "$sched_pid"; sched_status=$?; done
set -e
trap - TERM INT HUP
# Exit 3 alone (deadline passed, every stage passed): finish the final checks so the
# evidence is complete; finish() still exits 3 because the wall time is over the limit.
if [[ "$sched_status" -ne 0 && "$sched_status" -ne 3 ]]; then
  printf '%s FAIL scheduler exit %d (see sched/scheduler.log)\n' "$(date -u +%FT%TZ)" "$sched_status" | tee -a "$QUAL_ROOT/stages.log"
  exit "$sched_status"
fi

# ------------------------------------------------------ final checks (strict)
cd "$wasm"
load_helper_state
mapfile -t primary_results < "$QUAL_ROOT/state/primary-results.list"
check_primary
files=("$primary_dir"/*)
sha256sum -- "${files[@]}" > "$QUAL_ROOT/primary-after.sha256"
cmp "$QUAL_ROOT/primary-before.sha256" "$QUAL_ROOT/primary-after.sha256"
for index in 0 1 2 3; do sha256sum -c "$QUAL_ROOT/sqlite-selection-$index-after.sha256" > /dev/null; done
check_source
python3 "$QUALDRV_DIR/stage_set.py" verify --dag "$QUALDRV_DIR/dag.tsv" --stages-log "$QUAL_ROOT/stages.log" \
  --root "$QUAL_ROOT" --source "$expected_source" > "$QUAL_ROOT/stage-set-executed.json" || exit 4
jq -s '[.[] | .Manifest | {ID,Profile,HelperSHA256,ModelSHA256,RuntimeSHA256,EngineSHA256,CompilerSHA256,IntegratedSourceSHA256}]' \
  "${primary_results[@]}" > "$QUAL_ROOT/current-identities.json"
printf '%s\n' 'Matched-release candidate comparison is the separate perf entry point (run-perf.sh); no speedup or whole-task completion asserted. Native backend retired 2026-10-19: native-* stages not run.' > "$QUAL_ROOT/remaining-obligations.txt"
printf '%s PASS-STAGES qualification-driver (bulk pruning follows)\n' "$(date -u +%FT%TZ)" >> "$QUAL_ROOT/stages.log"
prune_bulk "$QUAL_ROOT/runtime" runtime
prune_bulk "$QUAL_ROOT/artifact" artifact
