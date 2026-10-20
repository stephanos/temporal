# shellcheck shell=bash
# Shared by run-validation.sh (driver) and stage.sh (one process per stage).
# Every stage command, timeout, admission, oracle and post-check below is the
# q1 serial driver's (../q1-serial/run-qualification.sh), split into one step
# per stage label; stage_set.py compares the stage calls and the shared helper
# bodies against that reference. Differences are scheduling plumbing only:
# lane ids in stages.log, locked appends, atomic writes of files that other
# concurrent stages read, run state passed through $QUAL_ROOT/state, and the
# frozen check's scratch registry for concurrently running stages.

qual_context() {
  # Inputs: QUALDRV_DIR QUALDRV_CONFIG QUALDRV_HEAD QUALDRV_LINT_BASE QUAL_ROOT QUALDRV_SCRATCH
  # shellcheck source=/dev/null
  source "$QUALDRV_CONFIG"
  driver_dir="$QUALDRV_DIR"
  freeze_root="$FREEZE_ROOT"
  retained_freeze="$RETAINED_FREEZE"
  host_tools="$retained_freeze/host-tools"
  expected_head="$QUALDRV_HEAD"
  lint_base="$QUALDRV_LINT_BASE"
  scratch="$QUALDRV_SCRATCH"
  export QUAL_REPO QUAL_ROOT
  [[ "$expected_head" =~ ^[0-9a-f]{40}$ ]]
  test "$expected_head" = "$EXPECTED_HEAD"
  [[ "$QUAL_ROOT" == /* && "$scratch" == /* ]]
  module_candidates=()
  for candidate in "$QUAL_REPO/tools/gomad3" "$QUAL_REPO/tools/gomad"; do
    [[ ! -f "$candidate/go.mod" ]] || module_candidates+=("$candidate")
  done
  test "${#module_candidates[@]}" -eq 1
  module="${module_candidates[0]}"
  module_name="${module##*/}"
  wasm="$module/internal/backends/wasm"
  installed_helper="$QUAL_ROOT/helpers/ordinary/gomad-wasmhost"
  primary_dir="$QUAL_ROOT/workloads/wasm-qualifications/v1"
  selected_dir="$QUAL_ROOT/sqlite-selections/wasm-qualifications/v1"
  source_manifest_ready=false
  helper_ready=false
  expected_helper=
  expected_source=
  admitted_helper=
  helper_admission=
}

qual_base_env() {
  # The serial driver's environment before its first stage.
  export qualification_go="/Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad/.toolchain/stock/2198cf19890232fc/go/bin/go"
  export GOMAD_WASM_STOCK_GO="$qualification_go" GOMAD_STOCK_GO="$qualification_go"
  export GOENV=off GOTOOLCHAIN=local GOWORK=off GOFLAGS=-p=1
  [[ "$PATH" == "$(dirname "$qualification_go")":* ]] || export PATH="$(dirname "$qualification_go"):$PATH"
  unset GOROOT GOMADSEED GOMAD_CHILD_SEED GOMAD3_CHILD_SEED GOEXPERIMENT GOOS GOARCH
  # Do not inherit opt-in producer roots, selection indices or diagnostic inputs.
  while IFS= read -r variable; do unset "$variable"; done < <(compgen -A variable GOMAD_)
  export GOMAD_STOCK_GO="$qualification_go"
  export GOMAD_WASM_STOCK_GO="$qualification_go"
  export TMPDIR="$scratch" GOCACHE="$scratch/go-cache"
  export CARGO_TARGET_DIR="$scratch/rust-check-target" CARGO_INCREMENTAL=0
  export RUSTUP_TOOLCHAIN=1.99.1 RUSTC_WRAPPER= RUSTC_WORKSPACE_WRAPPER=
  unset RUSTFLAGS CARGO_ENCODED_RUSTFLAGS
  export GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD=1
}

qual_stage_state() {
  # Run state a stage process inherits from earlier steps (set in the serial
  # driver as shell variables at the same points).
  expected_source="$EXPECTED_SOURCE_SHA256"
  source_manifest_ready=true
  RUSTC="$(cat "$QUAL_ROOT/state/rustc.path")"
  export RUSTC
  load_helper_state
}

load_helper_state() {
  if [[ -e "$QUAL_ROOT/state/helper.env" ]]; then
    # shellcheck source=/dev/null
    source "$QUAL_ROOT/state/helper.env"
    helper_ready=true
  fi
}

atomic_output() {
  # atomic_output FILE CMD...: readers in concurrent stages never see a partial file.
  local destination="$1" temporary
  shift
  test ! -e "$destination"
  temporary="$(mktemp "$destination.XXXXXX")"
  "$@" > "$temporary"
  mv -n -- "$temporary" "$destination"
  test ! -e "$temporary"
}

ledger() {
  (flock 9; printf '%s\n' "$1" | tee -a "$QUAL_ROOT/stages.log") 9>> "$QUAL_ROOT/stages.lock"
}

prune_bulk() {
  # Retain a full hash inventory, then drop large non-evidence intermediates.
  local dir="$1" label="$2"
  test -d "$dir" || return 0
  find "$dir" -type f -print0 | sort -z | xargs -0 -r sha256sum > "$QUAL_ROOT/pruned-$label.sha256"
  find "$dir" -type f -size +4M ! -name '*.json' ! -name '*.log' ! -name '*.txt' -print > "$QUAL_ROOT/pruned-$label.removed"
  xargs -r -d '\n' rm -f -- < "$QUAL_ROOT/pruned-$label.removed"
  df -h / >> "$QUAL_ROOT/disk.log"
}
publish_ordinary_helper() {
  local destination="$wasm/wasmhost/target/release/gomad-wasmhost" temporary
  mkdir -p "$(dirname "$destination")" || return $?
  test ! -L "$destination" || return $?
  temporary="$(mktemp "$(dirname "$destination")/qualification-helper.XXXXXX")" || return $?
  cp -- "$admitted_helper" "$temporary" || return $?
  chmod 755 "$temporary" || return $?
  cmp "$admitted_helper" "$temporary" || return $?
  mv -f -- "$temporary" "$destination" || return $?
  python3 "$driver_dir/helper-files.py" "$destination" "$installed_helper" "$admitted_helper" || return $?
  atomic_output "$QUAL_ROOT/canonical-helper.sha256" sha256sum -- "$destination"
}
clean_measure() (
  local limit="$1" variable
  shift
  while IFS= read -r variable; do unset "$variable"; done < <(compgen -A variable GOMAD_)
  sha256sum -c "$QUAL_ROOT/wasmmeasure.sha256" > /dev/null || return $?
  timeout --foreground "$limit" "$QUAL_ROOT/bin/wasmmeasure" "$@" --repo "$QUAL_REPO" --go "$qualification_go" \
    --prototype-parallel 1 --prototype-heavy 1
)

busy12() {
  # Twelve CPU-bound shell loops for the fn-178.8 revocation-after-reply
  # analogue gates; all are killed and reaped before the stage returns.
  local pids=() status=0 i
  for i in $(seq 1 12); do (while :; do :; done) & pids+=("$!"); done
  "$@" || status=$?
  kill "${pids[@]}" 2>/dev/null || true
  wait "${pids[@]}" 2>/dev/null || true
  return "$status"
}
digest_file() { sha256sum -- "$1" | cut -d ' ' -f1; }
check_source() {
  test "$(git -C "$QUAL_REPO" rev-parse HEAD)" = "$expected_head"
  python3 "$driver_dir/frozen.py" "$QUAL_REPO" "$freeze_root" "$EXPECTED_HEAD" "$EXPECTED_SOURCE_SHA256" \
    ${QUALDRV_STAGE:+--scratch-registry "$QUAL_ROOT/sched/registry" --self "$QUALDRV_STAGE"}
  sha256sum -c "$retained_freeze/host-tools.sha256" > /dev/null
  if [[ -e "$QUAL_ROOT/compiler.sha256" ]]; then sha256sum -c "$QUAL_ROOT/compiler.sha256" > /dev/null; fi
  [[ "$helper_ready" == true ]] || load_helper_state
  if [[ "$source_manifest_ready" == true ]]; then
    test "$(digest_file "$QUAL_ROOT/source.sha256")" = "$expected_source"
    sha256sum -c "$QUAL_ROOT/source.sha256" > /dev/null
    sha256sum -c "$QUAL_ROOT/compiler.sha256" > /dev/null
  fi
  if [[ "$helper_ready" == true ]]; then
    test "$(digest_file "$installed_helper")" = "$expected_helper"
    test "$(digest_file "$admitted_helper")" = "$expected_helper"
    sha256sum -c "$QUAL_ROOT/helper.sha256" > /dev/null
    if [[ -e "$QUAL_ROOT/conformance-helper.sha256" ]]; then sha256sum -c "$QUAL_ROOT/conformance-helper.sha256" > /dev/null; fi
    if [[ -e "$QUAL_ROOT/canonical-helper.sha256" ]]; then sha256sum -c "$QUAL_ROOT/canonical-helper.sha256" > /dev/null; fi
  fi
}
stage() {
  local label="$1" status=0 disposition=NEW_RUN category=WASM_OR_HOST lane="${QUALDRV_LANE:-0}"
  shift
  test "$label" = "${QUALDRV_STAGE:-$label}"
  check_source
  ledger "$(printf '%s START %s mem_avail_kb=%s load=%s lane=%s' "$(date -u +%FT%TZ)" "$label" "$(awk '/MemAvailable/{print $2}' /proc/meminfo)" "$(cut -d' ' -f1-3 /proc/loadavg)" "$lane")"
  printf 'cwd=%q\n' "$PWD" > "$QUAL_ROOT/$label.command"
  printf '%q ' "$@" >> "$QUAL_ROOT/$label.command"
  printf '\n' >> "$QUAL_ROOT/$label.command"
  "$@" > "$QUAL_ROOT/$label.log" 2>&1 || status=$?
  ledger "$(printf '%s EXIT %s %d %s lane=%s' "$(date -u +%FT%TZ)" "$label" "$status" "$disposition" "$lane")"
  jq -n --arg label "$label" --arg disposition "$disposition" --arg category "$category" --arg source "$expected_source" --argjson exit "$status" --argjson lane "$lane" \
    '{label:$label,disposition:$disposition,category:$category,source_manifest_sha256:$source,exit:$exit,lane:$lane}' > "$QUAL_ROOT/$label.receipt.json"
  check_source
  test "$status" -eq 0
  case "$label" in
    terminal-controls) python3 "$driver_dir/required-roots.py" "$wasm/wasi" '^TestTerminalExit' "$QUAL_ROOT/$label.log" 1;;
    cluster-provider) python3 "$driver_dir/required-roots.py" "$wasm/cluster" '^Test' "$QUAL_ROOT/$label.log" 1;;
    cluster-functional) python3 "$driver_dir/required-roots.py" "$wasm/wasi" 'TestCluster|TestBrokerOrders|TestWorld|TestClock|TestStorageCrash|TestImplementationIdentity' "$QUAL_ROOT/$label.log" 1;;
    busy-boundaries) python3 "$driver_dir/required-roots.py" "$wasm/wasi" '^(TestClusterOutputBoundaryParity|TestClusterBrokerDurableRestartParity)$' "$QUAL_ROOT/$label.log" 1;;
    busy-grouped) python3 "$driver_dir/required-roots.py" "$wasm/cluster" '^TestProviderGroupedAcknowledgmentCrashSourceLessReplay$' "$QUAL_ROOT/$label.log" 10;;
    retained-cluster) python3 "$driver_dir/required-roots.py" "$wasm/cluster" 'Retained|Replay' "$QUAL_ROOT/$label.log" 1;;
    ordinary) python3 "$driver_dir/required-roots.py" "$wasm/wasi" '^TestNamespaceConformance' "$QUAL_ROOT/$label.log" 1;;
  esac
}
named_test() {
  local label="$1" name="$2" timeout="$3" package="$4"
  stage "$label" "$qualification_go" test -v -tags test_dep -count=1 \
    "-timeout=$timeout" -run "^$name$" "$package"
  rg -q "^--- PASS: $name " "$QUAL_ROOT/$label.log"
  if rg -q "^--- SKIP: $name " "$QUAL_ROOT/$label.log"; then return 1; fi
}
one_path() {
  local destination="$1" root="$2" filename="$3"
  local paths=()
  mapfile -d '' -t paths < <(find "$root" -type f -name "$filename" -print0)
  test "${#paths[@]}" -eq 1
  printf -v "$destination" '%s' "${paths[0]}"
}
check_pair() {
  local result="$1" hash pid
  test -f "$result" && test ! -L "$result"
  test -f "$result.publication.json" && test ! -L "$result.publication.json"
  hash="$(digest_file "$result")"
  test "$(basename "$result")" = "result-$hash.json"
  pid="$(jq -er '.OperatorProcessID | select(type == "number" and . > 0)' "$result")"
  jq -e --arg hash "sha256:$hash" --argjson pid "$pid" \
    '.ResultSHA256 == $hash and .OperatorProcessID == $pid and .OperatorAfter != null' "$result.publication.json" > /dev/null
  jq -e --arg source "sha256:$expected_source" --arg helper "sha256:$expected_helper" \
    '.Manifest.Provisional == false and .Manifest.IntegratedSourceSHA256 == $source and
     .Manifest.HelperSHA256 == $helper and .Cost != null and .CostFailure == "" and
     .Budget.WithinBudget == true and .BudgetFailure == ""' "$result" > /dev/null
}
check_primary() {
  local files=("$primary_dir"/*)
  test "${#files[@]}" -eq 20
  sha256sum -- "${files[@]}" | cmp - "$QUAL_ROOT/primary-before.sha256"
}
capture_host_cpu() {
  local label="$1"
  test "$(uname -s)" = Linux
  test "$(uname -m)" = aarch64
  head -c 1048577 /proc/cpuinfo > "$QUAL_ROOT/$label-cpuinfo.txt"
  test -s "$QUAL_ROOT/$label-cpuinfo.txt"
  test "$(wc -c < "$QUAL_ROOT/$label-cpuinfo.txt")" -le 1048576
  rg -q '^CPU architecture[[:space:]]*:[[:space:]]*8[[:space:]]*$' "$QUAL_ROOT/$label-cpuinfo.txt"
  rg -q '^Features[[:space:]]*:.*[[:space:]]asimd([[:space:]]|$)' "$QUAL_ROOT/$label-cpuinfo.txt"
  sha256sum -- "$QUAL_ROOT/$label-cpuinfo.txt" > "$QUAL_ROOT/$label-cpuinfo.sha256"
}
rust_test() {
  local label="$1"; shift
  stage "$label" env -u GOMAD_HELPER_CORPUS_REGENERATE CARGO_BUILD_JOBS=1 RUST_TEST_THREADS=1 \
    cargo +1.99.1 test --manifest-path wasmhost/Cargo.toml --release --locked "$@"
  if rg -q -- '^test result: FAILED|^error: test failed' "$QUAL_ROOT/$label.log"; then return 1; fi
}

# ---------------------------------------------------------------- environments
# The GOMAD_* environment each stage had in the serial driver at its position.
env_helper() {
  # After helper admission and the conformance helper build.
  export GOMAD_WASM_TEST_HELPER="$installed_helper"
  export GOMAD_WASM_CONFORMANCE_HELPER="$QUAL_ROOT/helpers/conformance/gomad-wasmhost"
  export GOMAD_WASM_CONFORMANCE_REQUIRED=1
}
env_integrated() {
  # From the runtime qualification onward.
  env_helper
  export GOMAD_WASM_INTEGRATED_SOURCE_MANIFEST="$QUAL_ROOT/source.sha256"
  export GOMAD_WASM_QUALIFICATION_MANIFEST="$QUAL_ROOT/source.sha256"
  export GOMAD_WASM_INTEGRATED_SOURCE_SHA256="sha256:$expected_source"
}
env_after_runtime() {
  # Serial leftovers after the runtime qualification: its root stays exported until the
  # workload matrix, the storage recovery root through inspection and network.
  env_integrated
  export GOMAD_WASM_QUALIFICATION_ROOT="$QUAL_ROOT/runtime"
}
env_workloads() {
  env_integrated
  fixture_relative="$STAGED_FIXTURE_RELATIVE"
  export GOMAD_WASM_QUALIFICATION_ROOT="$QUAL_ROOT/workloads"
  export GOMAD_WASM_TEMPORAL_QUALIFICATION_INPUTS="$QUAL_ROOT/clusters"
  export GOMAD_WASM_TEMPORAL_PREPARATION_SOURCE="./$fixture_relative"
  export GOMAD_WASM_QUALIFY_WORKLOADS=1
}
load_primary_state() {
  mapfile -t primary_results < "$QUAL_ROOT/state/primary-results.list"
  test "${#primary_results[@]}" -eq 10
  GOMAD_WASM_SQLITE_NATIVE_CATALOGUE_RESULT="$(cat "$QUAL_ROOT/primary-sqlite.path")"
  export GOMAD_WASM_SQLITE_NATIVE_CATALOGUE_RESULT
  primary_sha="sha256:$(digest_file "$GOMAD_WASM_SQLITE_NATIVE_CATALOGUE_RESULT")"
  test "$primary_sha" = "$(cat "$QUAL_ROOT/state/primary-sqlite.sha256")"
}

# ------------------------------------------------------------- static stages
step_validate() {
  cd "$module"
  stage validate make validate
}
step_architecture() {
  cd "$module"
  stage architecture env GOMAD_ARCH_ALL_PLATFORMS=1 "$qualification_go" test -v -tags test_dep -count=1 -timeout=15m ./internal/archcheck
}
step_portable() {
  cd "$module"
  stage portable "$qualification_go" test -v -tags test_dep -count=1 -timeout=20m ./internal/model/... ./internal/evidence/... ./internal/host/... ./internal/backends/target/...
}
step_default_build() {
  cd "$module"
  stage default-build "$qualification_go" build ./...
}
step_default_vet() {
  cd "$module"
  stage default-vet "$qualification_go" vet -tags test_dep ./...
}
step_compile_only() {
  cd "$module"
  stage compile-only "$qualification_go" test -exec /bin/true -tags test_dep -count=1 -run '^$' ./internal/backends/wasm/... ./internal/runner/... ./cmd/...
}
step_lint_router() {
  cd "$QUAL_REPO"
  stage lint-router "$qualification_go" test -v -tags test_dep -count=1 ./cmd/tools/lintcode
}
step_lint_router_vet() {
  cd "$QUAL_REPO"
  stage lint-router-vet "$qualification_go" vet -tags test_dep ./cmd/tools/lintcode
}
actionlint_pin() {
  actionlint_version="$(sed -n 's/^ACTIONLINT_VER := //p' Makefile)"
  test "$actionlint_version" = v1.7.7
}
step_actionlint_build() {
  cd "$QUAL_REPO"
  actionlint_pin
  stage actionlint-build env "GOBIN=$QUAL_ROOT/bin" "$qualification_go" install "github.com/rhysd/actionlint/cmd/actionlint@$actionlint_version"
}
step_actionlint_version() {
  cd "$QUAL_REPO"
  actionlint_pin
  stage actionlint-version "$QUAL_ROOT/bin/actionlint" -version
  rg -qx "$actionlint_version" "$QUAL_ROOT/actionlint-version.log"
  atomic_output "$QUAL_ROOT/actionlint.sha256" sha256sum -- "$QUAL_ROOT/bin/actionlint"
}
step_workflow_lint() {
  cd "$QUAL_REPO"
  workflow_prefixes=()
  for prefix in gomad3 gomad; do
    if [[ -f ".github/workflows/$prefix.yml" && -f ".github/workflows/$prefix-smoke.yml" ]] && \
      rg -Fq "tools/$module_name/" ".github/workflows/$prefix.yml" && \
      rg -Fq "tools/$module_name/" ".github/workflows/$prefix-smoke.yml"; then
      workflow_prefixes+=("$prefix")
    fi
  done
  test "${#workflow_prefixes[@]}" -eq 1
  workflow_prefix="${workflow_prefixes[0]}"
  stage workflow-lint "$QUAL_ROOT/bin/actionlint" ".github/workflows/$workflow_prefix.yml" ".github/workflows/$workflow_prefix-smoke.yml"
  sha256sum -c "$QUAL_ROOT/actionlint.sha256" > /dev/null
}
step_lint_root() {
  cd "$QUAL_REPO"
  stage lint-root env GOCACHE=/home/agent/.cache/go-build-local make lint-code-fast "GOLANGCI_LINT_BASE_REV=$lint_base" GOLANGCI_LINT_FIX=false \
    "GOLANGCI_LINT=$host_tools/golangci-lint-v2.13.0" "ERRORTYPE=$host_tools/errortype"
}
step_lint_module() {
  cd "$QUAL_REPO"
  lint_target="lint-code-$module_name"
  if ! rg -q "^$lint_target:" Makefile; then lint_target=lint-code-gomad-wasm; fi
  sed -n "/^$lint_target:/,+1p" Makefile | rg -q -- "-module tools/$module_name$"
  stage lint-module env GOCACHE=/home/agent/.cache/go-build-local make "$lint_target" "GOLANGCI_LINT_BASE_REV=$lint_base" GOLANGCI_LINT_FIX=false \
    "GOLANGCI_LINT=$host_tools/golangci-lint-v2.13.0" "ERRORTYPE=$host_tools/errortype"
}

# -------------------------------------------------- host, Rust and helper builds
step_test_harness() {
  cd "$module"
  stage test-harness env GOMAXPROCS=2 make test-harness
}
step_world_race() {
  cd "$module"
  stage world-race "$qualification_go" test -race -tags=test_dep -p=1 -parallel=1 -count=1 ./internal/model/world/...
}
# Fresh Rust matrix: pinned fmt + six-mode strict Clippy, then the six retained
# test modes with one build job and one test thread (one shared Cargo target).
step_rust_lint() {
  cd "$wasm"
  stage rust-lint env CARGO_BUILD_JOBS=1 make lint-rust
}
step_rust_default_test() {
  cd "$wasm"
  rust_test rust-default-test
}
step_rust_conformance_unit() {
  cd "$wasm"
  rust_test rust-conformance-unit --features conformance --bin gomad-wasmhost
}
step_rust_diagnostic_test() {
  cd "$wasm"
  rust_test rust-diagnostic-test --features diagnostic-profile
}
step_rust_fuel_off_test() {
  cd "$wasm"
  rust_test rust-fuel-off-test --features diagnostic-fuel-off
}
step_rust_guest_profile_test() {
  cd "$wasm"
  rust_test rust-guest-profile-test --features guest-profile
}
step_rust_combined_test() {
  cd "$wasm"
  rust_test rust-combined-test --features guest-profile,diagnostic-fuel-off
}
# fn-187 R1/R4: regenerating the helper transcript corpus twice must leave the
# tree byte-identical (check_source enforces a clean worktree after each stage).
step_helper_corpus_regen_1() {
  cd "$wasm"
  stage helper-corpus-regen-1 env GOMAD_HELPER_CORPUS_REGENERATE=1 CARGO_BUILD_JOBS=1 RUST_TEST_THREADS=1 \
    cargo +1.99.1 test --manifest-path wasmhost/Cargo.toml --release --locked --test helper_corpus
}
step_helper_corpus_regen_2() {
  cd "$wasm"
  stage helper-corpus-regen-2 env GOMAD_HELPER_CORPUS_REGENERATE=1 CARGO_BUILD_JOBS=1 RUST_TEST_THREADS=1 \
    cargo +1.99.1 test --manifest-path wasmhost/Cargo.toml --release --locked --test helper_corpus
  rm -rf -- "${CARGO_TARGET_DIR:?}"
}
step_wasmmeasure_build() {
  cd "$wasm"
  stage wasmmeasure-build "$qualification_go" build -trimpath=false -o "$QUAL_ROOT/bin/wasmmeasure" ./cmd/wasmmeasure
  atomic_output "$QUAL_ROOT/wasmmeasure.sha256" sha256sum -- "$QUAL_ROOT/bin/wasmmeasure"
}
step_terminal_controls() {
  cd "$wasm"
  stage terminal-controls "$qualification_go" test -v -tags test_dep -count=1 -timeout=15m -run '^TestTerminalExit' ./wasi
}
step_helper_build() {
  cd "$wasm"
  # This producer owns pinned Cargo/rustc provenance, no wrappers, empty features,
  # raw build capture and the single-link immutable helper admission receipt.
  export GOMAD_WASM_TERMINAL_EXIT_HELPER_BUILD_ROOT="$QUAL_ROOT/helper-build"
  stage helper-build clean_measure 60m terminal-exit-helper-build --terminal-exit-helper-build-root "$QUAL_ROOT/helper-build"
  unset GOMAD_WASM_TERMINAL_EXIT_HELPER_BUILD_ROOT
  one_path helper_admission "$QUAL_ROOT/helper-build" helper-admission.json
  admitted_helper="$(jq -er '.Helper' "$helper_admission")"
  expected_helper="$(jq -er '.HelperSHA256' "$helper_admission")"
  test "$(digest_file "$admitted_helper")" = "$expected_helper"
  mkdir -p "$(dirname "$installed_helper")"
  if [[ -e "$installed_helper" ]]; then
    test -f "$installed_helper" && test ! -L "$installed_helper"
    cp -p -- "$installed_helper" "$QUAL_ROOT/previous-installed-helper"
  fi
  install_tmp="$(mktemp "$(dirname "$installed_helper")/qualification-helper.XXXXXX")"
  cp -- "$admitted_helper" "$install_tmp"
  chmod 755 "$install_tmp"
  cmp "$admitted_helper" "$install_tmp"
  mv -f -- "$install_tmp" "$installed_helper"
  atomic_output "$QUAL_ROOT/helper.sha256" sha256sum -- "$installed_helper" "$admitted_helper"
  atomic_output "$QUAL_ROOT/state/helper.env" printf 'expected_helper=%q\nadmitted_helper=%q\nhelper_admission=%q\n' \
    "$expected_helper" "$admitted_helper" "$helper_admission"
  helper_ready=true
  check_source
}
step_conformance_helper_build() {
  # Independent of the release helper's bytes, so it builds next to
  # helper-build; installation and the distinct-file check follow both builds
  # (ordinary-helper-publication), as in the serial order.
  cd "$wasm"
  export GOMAD_WASM_TEST_HELPER="$installed_helper"
  stage conformance-helper-build cargo +1.99.1 build --manifest-path wasmhost/Cargo.toml --release --locked \
    --no-default-features --features conformance --target-dir "$scratch/conformance-target"
}
step_ordinary_helper_publication() {
  cd "$wasm"
  test "$helper_ready" = true
  mkdir -p "$QUAL_ROOT/helpers/conformance"
  install -m 755 "$scratch/conformance-target/release/gomad-wasmhost" "$QUAL_ROOT/helpers/conformance/gomad-wasmhost"
  cmp "$scratch/conformance-target/release/gomad-wasmhost" "$QUAL_ROOT/helpers/conformance/gomad-wasmhost"
  rm -rf -- "${scratch:?}/conformance-target"
  env_helper
  python3 "$driver_dir/helper-files.py" "$installed_helper" "$admitted_helper" "$GOMAD_WASM_CONFORMANCE_HELPER"
  atomic_output "$QUAL_ROOT/conformance-helper.sha256" sha256sum -- "$GOMAD_WASM_CONFORMANCE_HELPER"
  stage ordinary-helper-publication publish_ordinary_helper
}

# ------------------------------------------------- functional stages (helper)
step_ordinary() {
  cd "$module"; env_helper
  stage ordinary make test-wasm "GOMAD_STOCK_GO=$qualification_go"
}
step_retained_durable_restart() {
  cd "$wasm"; env_helper
  named_test retained-durable-restart TestProviderActualCrashDurableRestart 15m ./cluster
}
# fn-178 in-process cluster gates on the authenticated release helper.
step_cluster_provider() {
  cd "$wasm"; env_helper
  stage cluster-provider "$qualification_go" test -v -tags test_dep -count=1 -timeout=30m ./cluster
}
step_cluster_functional() {
  cd "$wasm"; env_helper
  stage cluster-functional "$qualification_go" test -v -tags test_dep -count=1 -timeout=30m \
    -run 'TestCluster|TestBrokerOrders|TestWorld|TestClock|TestStorageCrash|TestImplementationIdentity' ./wasi
}
step_busy_boundaries() {
  cd "$wasm"; env_helper
  stage busy-boundaries busy12 "$qualification_go" test -v -tags test_dep -count=1 -timeout=30m \
    -run '^(TestClusterOutputBoundaryParity|TestClusterBrokerDurableRestartParity)$' ./wasi
}
step_busy_grouped() {
  cd "$wasm"; env_helper
  stage busy-grouped busy12 "$qualification_go" test -v -tags test_dep -count=10 -timeout=30m \
    -run '^TestProviderGroupedAcknowledgmentCrashSourceLessReplay$' ./cluster
}
step_network_streams() {
  cd "$wasm"; env_helper
  export GOMAD_WASM_TEMPORAL_RECOVERY_ROOT="$QUAL_ROOT/network-streams"
  named_test network-streams TestBrokerIndependentGuestStreamsReplay 30m ./wasi
  unset GOMAD_WASM_TEMPORAL_RECOVERY_ROOT
  jq -e --arg helper "$expected_helper" '.network_owner == "rust-cluster" and (.network_owner_implementation_sha256 == $helper or .network_owner_implementation_sha256 == ("sha256:" + $helper)) and
    .reference_network_owner == "go-broker-reference" and .distinct_choice_traces >= 2' "$QUAL_ROOT/network-streams/broker-network-streams-report.json" > /dev/null
}
step_retained_cluster() {
  cd "$wasm"; env_helper
  stage retained-cluster "$qualification_go" test -v -tags test_dep -count=1 -timeout=30m -run 'Retained|Replay' ./cluster
}

# The actual ./tests producer supplies authenticated preparation inputs. The
# measured helper is its retained admission copy, byte-identical to installation.
step_terminal_module() {
  cd "$wasm"; env_helper
  export GOMAD_WASM_TERMINAL_EXIT_PREPARE_ROOT="$QUAL_ROOT/terminal-module"
  named_test terminal-module TestPrepareTemporalTerminalExitModule 60m ./backend
  unset GOMAD_WASM_TERMINAL_EXIT_PREPARE_ROOT
  one_path module_receipt "$QUAL_ROOT/terminal-module" temporal-producer.json
  atomic_output "$QUAL_ROOT/state/module-receipt.path" printf '%s\n' "$module_receipt"
}
env_terminal_exit() {
  env_helper
  module_receipt="$(cat "$QUAL_ROOT/state/module-receipt.path")"
  export GOMAD_WASM_TERMINAL_EXIT_MODULE_RECEIPT="$module_receipt"
  export GOMAD_WASM_TERMINAL_EXIT_MODULE="$(jq -er '.Prepared.Path' "$module_receipt")"
  export GOMAD_WASM_TERMINAL_EXIT_HELPER_RECEIPT="$helper_admission"
  export GOMAD_WASM_TERMINAL_EXIT_ROOT="$QUAL_ROOT/terminal-exit"
  export GOMAD_WASM_TEST_HELPER="$admitted_helper"
}
step_terminal_host_admission() {
  cd "$wasm"; env_terminal_exit
  capture_host_cpu terminal-host
  named_test terminal-host-admission TestDiagnosticPerformanceCurrentHostAdmission 15m ./internal/measure
  jq -n --arg head "$expected_head" --arg source "sha256:$expected_source" \
    --arg helper "sha256:$expected_helper" --arg module "$(digest_file "$GOMAD_WASM_TERMINAL_EXIT_MODULE")" \
    --arg cpu "$(digest_file "$QUAL_ROOT/terminal-host-cpuinfo.txt")" \
    --arg receipt "$module_receipt" --arg admission "$helper_admission" \
    '{source_revision:$head,source_manifest_sha256:$source,helper_sha256:$helper,module_sha256:("sha256:"+$module),
      cpuinfo_sha256:("sha256:"+$cpu),module_receipt:$receipt,helper_admission:$admission,
      required_host:"linux/arm64",cpuinfo_byte_limit:1048576}' > "$QUAL_ROOT/terminal-host-binding.json"
}
step_terminal_exit() {
  cd "$wasm"; env_terminal_exit
  stage terminal-exit clean_measure 60m terminal-exit \
    --stock-go "$qualification_go" --test-helper "$admitted_helper" \
    --terminal-exit-root "$QUAL_ROOT/terminal-exit" \
    --terminal-exit-helper-receipt "$helper_admission" \
    --terminal-exit-module "$GOMAD_WASM_TERMINAL_EXIT_MODULE" \
    --terminal-exit-module-receipt "$module_receipt"
  sha256sum -c "$QUAL_ROOT/terminal-host-cpuinfo.sha256" > /dev/null
  prune_bulk "$QUAL_ROOT/terminal-module" terminal-module
  prune_bulk "$QUAL_ROOT/terminal-exit" terminal-exit
}

step_runtime() {
  local index="$1"
  cd "$wasm"; env_integrated
  [[ "$index" =~ ^([0-9]|[12][0-9]|3[01])$ ]]
  export GOMAD_WASM_QUALIFICATION_ROOT="$QUAL_ROOT/runtime"
  export GOMAD_WASM_QUALIFICATION_INDEX="$index"
  named_test "runtime-$index" TestCooperativeRuntimeQualification 10m ./wasi
}
step_runtime_aggregate() {
  cd "$wasm"; env_integrated
  export GOMAD_WASM_QUALIFICATION_ROOT="$QUAL_ROOT/runtime"
  named_test runtime-aggregate TestCooperativeRuntimeQualificationAggregate 15m ./wasi
}
step_artifact() {
  cd "$wasm"; env_after_runtime
  export GOMAD_WASM_TASK4_EVIDENCE_ROOT="$QUAL_ROOT/artifact"
  named_test artifact TestIntegratedFailureArtifactFreshObservedReplays 15m ./backend
}
step_operator() {
  cd "$wasm"; env_after_runtime
  named_test operator TestInstalledOperatorDiscoversInspectsAndReplaysWithoutBuildInputs 30m ./command
}
step_storage() {
  cd "$wasm"; env_after_runtime
  export GOMAD_WASM_TEMPORAL_RECOVERY_ROOT="$QUAL_ROOT/storage"
  named_test storage TestTemporalCommittedWorkflowFreshGuestCrashReopen 90m ./wasi
  prune_bulk "$QUAL_ROOT/storage" storage
}
step_temporal_inspection() {
  cd "$wasm"; env_after_runtime
  export GOMAD_WASM_TEMPORAL_RECOVERY_ROOT="$QUAL_ROOT/storage"
  export GOMAD_WASM_TEMPORAL_INSPECTION_ROOT="$QUAL_ROOT/inspection"
  named_test temporal-inspection TestTemporalInspectionRetainedFailure 90m ./backend
  for owner in shard workflow onebox; do
    rg -q "^[[:space:]]+--- PASS: TestTemporalInspectionRetainedFailure/$owner " "$QUAL_ROOT/temporal-inspection.log"
  done
  unset GOMAD_WASM_TEMPORAL_INSPECTION_ROOT
  prune_bulk "$QUAL_ROOT/inspection" inspection
}
step_network() {
  cd "$wasm"; env_after_runtime
  export GOMAD_WASM_TEMPORAL_RECOVERY_ROOT="$QUAL_ROOT/storage"
  export GOMAD_WASM_NETWORK_TEMPORAL_ROOT="$QUAL_ROOT/network"
  named_test network TestTemporalNetworkNamedResetRetriesAndExactReplay 90m ./wasi
  unset GOMAD_WASM_NETWORK_TEMPORAL_ROOT
  prune_bulk "$QUAL_ROOT/network" network
}
step_clusters() {
  cd "$wasm"; env_after_runtime
  export GOMAD_WASM_TEMPORAL_RECOVERY_ROOT="$QUAL_ROOT/clusters"
  named_test clusters TestTemporalIndependentClustersPartitionHealWholeClusterCrash 90m ./wasi
  unset GOMAD_WASM_TEMPORAL_RECOVERY_ROOT
  one_path clusters_owner "$QUAL_ROOT/clusters" two-cluster-owner.json
  jq -e --arg helper "$expected_helper" '.network_owner == "rust-cluster" and (.network_owner_implementation_sha256 == $helper or .network_owner_implementation_sha256 == ("sha256:" + $helper))' "$clusters_owner" > /dev/null
}

# ------------------------------------------------------- workload matrix chain
workload_inputs() {
  # Match the current producer's ordered concatenation, never a historical pin.
  sha256sum -- "$wasm/testdata/temporal_storage/recovery_test.go" "$wasm/testdata/temporal_broker/replication_test.go" > "$QUAL_ROOT/broker-inputs.sha256"
  fixture_hash="$(cat -- "$wasm/testdata/temporal_storage/recovery_test.go" "$wasm/testdata/temporal_broker/replication_test.go" | sha256sum | cut -d ' ' -f1)"
  fixture_relative="$STAGED_FIXTURE_RELATIVE"
  cmp "$QUAL_REPO/$fixture_relative/recovery_test.go" "$wasm/testdata/temporal_storage/recovery_test.go"
  cmp "$QUAL_REPO/$fixture_relative/replication_test.go" "$wasm/testdata/temporal_broker/replication_test.go"
  printf '%s\n' "$fixture_relative" > "$QUAL_ROOT/broker-source.path"
}
step_workload_cell() {
  local wanted="$1" entry workload profile gate cell
  cd "$wasm"
  if [[ "$wanted" == workload-fixture-controlled-success ]]; then workload_inputs; fi
  for entry in 'fixture controlled TestActualMatchedFixtureQualification' 'fixture cooperative TestActualMatchedFixtureQualification' \
    'sqlite cooperative TestActualSQLiteDurableQualification' 'onebox cooperative TestActualOneboxQualification' \
    'multi-instance cooperative TestActualTemporalClusterQualification'; do
    read -r workload profile gate <<< "$entry"
    for cell in success canary; do
      [[ "workload-$workload-$profile-$cell" == "$wanted" ]] || continue
      env_workloads
      export GOMAD_WASM_QUALIFICATION_PROFILE="$profile"
      export GOMAD_WASM_QUALIFICATION_CELL="$cell"
      named_test "workload-$workload-$profile-$cell" "$gate" 90m ./qualification
      return 0
    done
  done
  return 1
}
step_workload_corpus() {
  cd "$wasm"; env_workloads
  export GOMAD_WASM_QUALIFICATION_PROFILE=cooperative
  export GOMAD_WASM_QUALIFICATION_CELL=success
  named_test workload-corpus TestActualMandatoryWorkloadCorpus 15m ./qualification
  files=("$primary_dir"/*)
  test "${#files[@]}" -eq 20
  primary_results=()
  sqlite_sources=()
  for result in "${files[@]}"; do
    [[ "$result" == *.publication.json ]] && continue
    check_pair "$result"
    primary_results+=("$result")
    if [[ "$(jq -r '[.Manifest.Workload,.Manifest.ID,.Manifest.Cell] | join("/")' "$result")" == sqlite/sqlite-durable-recovery/success ]]; then sqlite_sources+=("$result"); fi
  done
  test "${#primary_results[@]}" -eq 10 && test "${#sqlite_sources[@]}" -eq 1
  sha256sum -- "${files[@]}" > "$QUAL_ROOT/primary-before.sha256"
  export GOMAD_WASM_SQLITE_NATIVE_CATALOGUE_RESULT="${sqlite_sources[0]}"
  primary_sha="sha256:$(digest_file "$GOMAD_WASM_SQLITE_NATIVE_CATALOGUE_RESULT")"
  printf '%s\n' "$GOMAD_WASM_SQLITE_NATIVE_CATALOGUE_RESULT" > "$QUAL_ROOT/primary-sqlite.path"
  jq -e '[.NativeCrashCatalogues[] | select(.Handle.node == "sqlite" and .Handle.incarnation == 2 and .Volume == "sqlite")] | length == 1 and (.[0].Enumeration.Complete == true) and (.[0].Enumeration.States | length == 4)' "$GOMAD_WASM_SQLITE_NATIVE_CATALOGUE_RESULT" > /dev/null
  mkdir -p "$selected_dir"
  files=("$selected_dir"/*)
  test "${#files[@]}" -eq 0
  printf '%s\n' "${primary_results[@]}" > "$QUAL_ROOT/state/primary-results.list"
  printf '%s\n' "$primary_sha" > "$QUAL_ROOT/state/primary-sqlite.sha256"
}
step_sqlite_selection() {
  local index="$1"
  cd "$wasm"; env_workloads
  [[ "$index" =~ ^[0-3]$ ]]
  load_primary_state
  export GOMAD_WASM_QUALIFICATION_PROFILE=cooperative
  export GOMAD_WASM_QUALIFICATION_CELL=success
  check_primary
  export GOMAD_WASM_QUALIFICATION_ROOT="$QUAL_ROOT/workloads"
  export GOMAD_WASM_SQLITE_SELECTION_INDEX="$index"
  named_test "sqlite-selection-$index" TestActualSQLiteSelectedCrashQualification 90m ./qualification
  sha256sum -c "$QUAL_ROOT/primary-before.sha256" > /dev/null
  files=("$primary_dir"/*)
  test "${#files[@]}" -eq 22
  new_results=()
  for result in "${files[@]}"; do
    [[ "$result" == *.publication.json ]] && continue
    known=false
    for original in "${primary_results[@]}"; do [[ "$result" != "$original" ]] || known=true; done
    [[ "$known" == true ]] || new_results+=("$result")
  done
  test "${#new_results[@]}" -eq 1
  result="${new_results[0]}"
  check_pair "$result"
  jq -e --arg source "$primary_sha" --argjson index "$index" --slurpfile primary "$GOMAD_WASM_SQLITE_NATIVE_CATALOGUE_RESULT" \
    '([$primary[0].NativeCrashCatalogues[] | select(.Handle.node == "sqlite" and .Handle.incarnation == 2 and .Volume == "sqlite")][0].Enumeration.States[$index]) as $state |
     .Manifest.Workload == "sqlite" and .Manifest.Cell == "success" and .Qualification.qualified == true and
     .Manifest.ID == ("sqlite-durable-selection-" + ($state.Identity | ltrimstr("sha256:"))) and
     .NativeCrashSelection == $state and .NativeCrashSourceResultSHA256 == $source' "$result" > /dev/null
  destination="$selected_dir/$(basename "$result")"
  test ! -e "$destination" && test ! -L "$destination"
  test ! -e "$destination.publication.json" && test ! -L "$destination.publication.json"
  result_sha="$(digest_file "$result")"
  receipt_sha="$(digest_file "$result.publication.json")"
  sha256sum -- "$result" "$result.publication.json" > "$QUAL_ROOT/sqlite-selection-$index-before.sha256"
  printf '%s\t%s\t%s\t%s\t%s\n' "$index" "$result" "$destination" "$result_sha" "$receipt_sha" >> "$QUAL_ROOT/sqlite-relocations.tsv"
  mv -n -- "$result" "$destination"
  test ! -e "$result"
  mv -n -- "$result.publication.json" "$destination.publication.json"
  test ! -e "$result.publication.json"
  test "$(digest_file "$destination")" = "$result_sha"
  test "$(digest_file "$destination.publication.json")" = "$receipt_sha"
  check_pair "$destination"
  sha256sum -- "$destination" "$destination.publication.json" > "$QUAL_ROOT/sqlite-selection-$index-after.sha256"
  check_primary
  check_source
}
step_workload_corpus_after_relocation() {
  cd "$wasm"; env_workloads; load_primary_state
  export GOMAD_WASM_QUALIFICATION_PROFILE=cooperative
  export GOMAD_WASM_QUALIFICATION_CELL=success
  export GOMAD_WASM_QUALIFICATION_ROOT="$QUAL_ROOT/workloads"
  named_test workload-corpus-after-relocation TestActualMandatoryWorkloadCorpus 15m ./qualification
}
step_sqlite_catalogue() {
  cd "$wasm"; env_workloads; load_primary_state
  export GOMAD_WASM_QUALIFICATION_PROFILE=cooperative
  export GOMAD_WASM_QUALIFICATION_CELL=success
  export GOMAD_WASM_QUALIFICATION_ROOT="$QUAL_ROOT/sqlite-selections"
  named_test sqlite-catalogue TestActualSQLiteSelectionCatalogue 15m ./qualification
}

run_step() {
  local label="$1"
  case "$label" in
    runtime-aggregate) step_runtime_aggregate;;
    runtime-*) step_runtime "${label#runtime-}";;
    sqlite-selection-*) step_sqlite_selection "${label#sqlite-selection-}";;
    sqlite-catalogue) step_sqlite_catalogue;;
    workload-corpus) step_workload_corpus;;
    workload-corpus-after-relocation) step_workload_corpus_after_relocation;;
    workload-*) step_workload_cell "$label";;
    *)
      local function="step_${label//-/_}"
      [[ "$(type -t "$function")" == function ]] || { echo "no step for stage $label" >&2; return 1; }
      "$function";;
  esac
}
