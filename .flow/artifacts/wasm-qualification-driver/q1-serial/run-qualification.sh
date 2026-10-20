#!/usr/bin/env bash
# q1: combined fn-189/fn-187 qualification of frozen e50c6476fb (re-frozen from 173d3ef560 by conductor) in /home/agent/wt/q1.
# Derived from fn189/continuation-2026-10-19/driver/run-qualification.sh; every stage
# command, timeout, admission and post-check is unchanged. Differences: frozen
# worktree/manifest bindings, ALL stages executed fresh (no receipt reuse), Rust
# matrix + helper-corpus regeneration and host harness/World race added, and
# bulky intermediate pruning (hash-inventoried) after a stage's checks pass.
# Usage: bash run-qualification.sh HEAD LINT_BASE FRESH_EVIDENCE FRESH_NATIVE_SCRATCH
# --plan prints the prepared mapping with shell builtins only.
set -euo pipefail
if [[ "${1:-}" == --plan ]]; then
  while IFS= read -r line; do printf '%s\n' "$line"; done < "${BASH_SOURCE[0]%/*}/stage-map.tsv"
  exit 0
fi
readonly driver_dir="$(cd -- "${BASH_SOURCE[0]%/*}" && pwd -P)"
readonly freeze_root=/home/agent/wt/evidence/q1/freeze
readonly retained_freeze=/home/agent/wt/evidence/fn187/final-integrated-2026-10-19
readonly host_tools="$retained_freeze/host-tools"
set -o noclobber
shopt -s nullglob dotglob
export LC_ALL=C
test "$#" -eq 4
readonly expected_head="$1" lint_base="$2"
[[ "$expected_head" =~ ^[0-9a-f]{40}$ ]]
export QUAL_REPO=/home/agent/wt/q1
export QUAL_ROOT="$3"
readonly scratch="$4"
[[ "$QUAL_ROOT" == /* && "$scratch" == /* ]]
test -d "$QUAL_ROOT" && test ! -e "$QUAL_ROOT/stages.log"
trap 'driver_status=$?; printf "driver_exit=%d\n" "$driver_status" > "$QUAL_ROOT/driver-exit.txt"' EXIT
module_candidates=()
for candidate in "$QUAL_REPO/tools/gomad3" "$QUAL_REPO/tools/gomad"; do
  [[ ! -f "$candidate/go.mod" ]] || module_candidates+=("$candidate")
done
test "${#module_candidates[@]}" -eq 1
readonly module="${module_candidates[0]}"
readonly module_name="${module##*/}"
readonly wasm="$module/internal/backends/wasm"
readonly installed_helper="$QUAL_ROOT/helpers/ordinary/gomad-wasmhost"
readonly primary_dir="$QUAL_ROOT/workloads/wasm-qualifications/v1"
readonly selected_dir="$QUAL_ROOT/sqlite-selections/wasm-qualifications/v1"
export qualification_go="/Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad/.toolchain/stock/2198cf19890232fc/go/bin/go"
export GOMAD_WASM_STOCK_GO="$qualification_go" GOMAD_STOCK_GO="$qualification_go"
export GOENV=off GOTOOLCHAIN=local GOWORK=off GOFLAGS=-p=1
export PATH="$(dirname "$qualification_go"):$PATH"
unset GOROOT GOMADSEED GOMAD_CHILD_SEED GOMAD3_CHILD_SEED GOEXPERIMENT GOOS GOARCH
# Do not inherit opt-in producer roots, selection indices or diagnostic inputs.
while IFS= read -r variable; do unset "$variable"; done < <(compgen -A variable GOMAD_)
export GOMAD_STOCK_GO="$qualification_go"
export GOMAD_WASM_STOCK_GO="$qualification_go"
test "$expected_head" = e50c6476fba40d3b31ddb20d1030abf08807412c
test "$(stat -f -c %T "$(dirname "$scratch")")" != fuseblk
case "$(stat -f -c %T "$(dirname "$scratch")")" in *fuse*|virtiofs) exit 2;; esac
mkdir "$scratch"
export TMPDIR="$scratch" GOCACHE="$scratch/go-cache"
export CARGO_TARGET_DIR="$scratch/rust-check-target" CARGO_INCREMENTAL=0
export RUSTUP_TOOLCHAIN=1.99.1 RUSTC_WRAPPER= RUSTC_WORKSPACE_WRAPPER=
unset RUSTFLAGS CARGO_ENCODED_RUSTFLAGS
export GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD=1
source_manifest_ready=false
helper_ready=false
expected_helper=
expected_source=


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
  sha256sum -- "$destination" > "$QUAL_ROOT/canonical-helper.sha256"
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
  python3 "$driver_dir/frozen.py" "$QUAL_REPO" "$freeze_root"
  sha256sum -c "$retained_freeze/host-tools.sha256" > /dev/null
  if [[ -e "$QUAL_ROOT/compiler.sha256" ]]; then sha256sum -c "$QUAL_ROOT/compiler.sha256" > /dev/null; fi
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
  local label="$1" status=0 disposition=NEW_RUN category=WASM_OR_HOST
  shift
  case "$label" in native-*) category=STATIC;; esac
  check_source
  printf '%s START %s mem_avail_kb=%s load=%s\n' "$(date -u +%FT%TZ)" "$label" "$(awk '/MemAvailable/{print $2}' /proc/meminfo)" "$(cut -d' ' -f1-3 /proc/loadavg)" | tee -a "$QUAL_ROOT/stages.log"
  printf 'cwd=%q\n' "$PWD" > "$QUAL_ROOT/$label.command"
  printf '%q ' "$@" >> "$QUAL_ROOT/$label.command"
  printf '\n' >> "$QUAL_ROOT/$label.command"
  case "$label" in
    *) "$@" > "$QUAL_ROOT/$label.log" 2>&1 || status=$?;;
  esac
  printf '%s EXIT %s %d %s\n' "$(date -u +%FT%TZ)" "$label" "$status" "$disposition" | tee -a "$QUAL_ROOT/stages.log"
  jq -n --arg label "$label" --arg disposition "$disposition" --arg category "$category" --arg source "$expected_source" --argjson exit "$status" \
    '{label:$label,disposition:$disposition,category:$category,source_manifest_sha256:$source,exit:$exit}' > "$QUAL_ROOT/$label.receipt.json"
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
named_native_test() {
  local label="$1" name="$2" timeout="$3" package="$4"
  stage "$label" "$qualification_go" test -v -tags test_dep,gomad_native -count=1 \
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

# Clean frozen worktree: manifest = every tracked regular non-.flow file.
cp -- "$freeze_root/source.sha256" "$QUAL_ROOT/source.sha256"
cp -- "$freeze_root/source-symlinks.json" "$QUAL_ROOT/source-symlinks.json"
cp -- "$freeze_root/preflight.json" "$QUAL_ROOT/preflight.json"
expected_source="$(digest_file "$QUAL_ROOT/source.sha256")"
test "$expected_source" = 503ccf73ae1b9c651ad2544175b854464febf296409d2da6192f0ecad21a2641
source_manifest_ready=true
printf 'sha256:%s\n' "$expected_source" > "$QUAL_ROOT/source.digest"
check_source

cd "$module"
stage validate make validate
stage architecture env GOMAD_ARCH_ALL_PLATFORMS=1 "$qualification_go" test -v -tags test_dep -count=1 -timeout=15m ./internal/archcheck
stage portable "$qualification_go" test -v -tags test_dep -count=1 -timeout=20m ./internal/model/... ./internal/evidence/... ./internal/host/... ./internal/backends/target/...
stage default-build "$qualification_go" build ./...
stage default-vet "$qualification_go" vet -tags test_dep ./...
stage compile-only "$qualification_go" test -exec /bin/true -tags test_dep -count=1 -run '^$' ./internal/backends/wasm/... ./internal/runner/... ./cmd/...
stage native-profile-inventory "$qualification_go" test -v -tags test_dep,gomad_native -count=1 -timeout=15m \
  -run '^TestDeterministicProfileCompatibilityGolden$' go.temporal.io/server/tools/gomad/internal/backends/native/transcript
# This reviewed function validates both portable inventory/implementation rows
# before its unsupported-host guard. Only that precise native-frame SKIP is valid.
test "$(rg -c '^=== RUN   TestDeterministicProfileCompatibilityGolden$' "$QUAL_ROOT/native-profile-inventory.log")" -eq 1
test "$(rg -c '^--- SKIP: TestDeterministicProfileCompatibilityGolden ' "$QUAL_ROOT/native-profile-inventory.log")" -eq 1
test "$(rg -c '^[[:space:]]+profile_test.go:[0-9]+: no profile golden for linux/arm64$' "$QUAL_ROOT/native-profile-inventory.log")" -eq 1
test "$(rg -c -- '--- SKIP:' "$QUAL_ROOT/native-profile-inventory.log")" -eq 1
if rg -q -- '--- (FAIL|PASS):|^FAIL' "$QUAL_ROOT/native-profile-inventory.log"; then exit 1; fi
printf '%s\n' 'Portable Darwin/arm64 and Linux/amd64 inventory and implementation assertions reached the reviewed Linux/arm64 host guard without failure. Unsupported native-frame branch SKIPPED; no native execution pass.' > "$QUAL_ROOT/native-profile-disposition.txt"
named_native_test native-bootstrap-pin TestPortableBootstrapPlatformGoldens 15m go.temporal.io/server/tools/gomad/internal/backends/native/transcript
for platform in darwin/arm64 linux/amd64; do
  rg -q "^[[:space:]]+--- PASS: TestPortableBootstrapPlatformGoldens/$platform " "$QUAL_ROOT/native-bootstrap-pin.log"
done
named_native_test native-semantic-pin TestSemanticInstrumentationIdentityBindsProbeNamesAndStableIDs 15m go.temporal.io/server/tools/gomad/internal/backends/native/transcript
named_test native-minimizer-preservation TestChoiceModePreservesNativeControllerAndStateEncoding 15m ./internal/runner/internal/minimizer
stage native-source "$qualification_go" test -v -tags test_dep,gomad_native -count=1 -timeout=15m \
  go.temporal.io/server/tools/gomad/internal/backends/native/sourceinventory go.temporal.io/server/tools/gomad/internal/backends/native/toolchain/version go.temporal.io/server/tools/gomad/internal/backends/native/generation/...
cd "$QUAL_REPO"
stage lint-router "$qualification_go" test -v -tags test_dep -count=1 ./cmd/tools/lintcode
stage lint-router-vet "$qualification_go" vet -tags test_dep ./cmd/tools/lintcode
actionlint_version="$(sed -n 's/^ACTIONLINT_VER := //p' Makefile)"
test "$actionlint_version" = v1.7.7
mkdir "$QUAL_ROOT/bin"
stage actionlint-build env "GOBIN=$QUAL_ROOT/bin" "$qualification_go" install "github.com/rhysd/actionlint/cmd/actionlint@$actionlint_version"
stage actionlint-version "$QUAL_ROOT/bin/actionlint" -version
rg -qx "$actionlint_version" "$QUAL_ROOT/actionlint-version.log"
sha256sum -- "$QUAL_ROOT/bin/actionlint" > "$QUAL_ROOT/actionlint.sha256"
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
stage lint-root env GOCACHE=/home/agent/.cache/go-build-local make lint-code-fast "GOLANGCI_LINT_BASE_REV=$lint_base" GOLANGCI_LINT_FIX=false \
  "GOLANGCI_LINT=$host_tools/golangci-lint-v2.13.0" "ERRORTYPE=$host_tools/errortype"
lint_target="lint-code-$module_name"
if ! rg -q "^$lint_target:" Makefile; then lint_target=lint-code-gomad-wasm; fi
sed -n "/^$lint_target:/,+1p" Makefile | rg -q -- "-module tools/$module_name$"
stage lint-module env GOCACHE=/home/agent/.cache/go-build-local make "$lint_target" "GOLANGCI_LINT_BASE_REV=$lint_base" GOLANGCI_LINT_FIX=false \
  "GOLANGCI_LINT=$host_tools/golangci-lint-v2.13.0" "ERRORTYPE=$host_tools/errortype"

cd "$module"
stage test-harness env GOMAXPROCS=2 make test-harness
stage world-race "$qualification_go" test -race -tags=test_dep -p=1 -parallel=1 -count=1 ./internal/model/world/...
cd "$wasm"
# Fresh Rust matrix (previously reused receipts): pinned fmt + six-mode strict Clippy,
# then the six retained test modes with one build job and one test thread.
stage rust-lint env CARGO_BUILD_JOBS=1 make lint-rust
rust_test() {
  local label="$1"; shift
  stage "$label" env -u GOMAD_HELPER_CORPUS_REGENERATE CARGO_BUILD_JOBS=1 RUST_TEST_THREADS=1 \
    cargo +1.99.1 test --manifest-path wasmhost/Cargo.toml --release --locked "$@"
  if rg -q -- '^test result: FAILED|^error: test failed' "$QUAL_ROOT/$label.log"; then return 1; fi
}
rust_test rust-default-test
rust_test rust-conformance-unit --features conformance --bin gomad-wasmhost
rust_test rust-diagnostic-test --features diagnostic-profile
rust_test rust-fuel-off-test --features diagnostic-fuel-off
rust_test rust-guest-profile-test --features guest-profile
rust_test rust-combined-test --features guest-profile,diagnostic-fuel-off
# fn-187 R1/R4: regenerating the helper transcript corpus twice must leave the
# tree byte-identical (check_source enforces a clean worktree after each stage).
stage helper-corpus-regen-1 env GOMAD_HELPER_CORPUS_REGENERATE=1 CARGO_BUILD_JOBS=1 RUST_TEST_THREADS=1 \
  cargo +1.99.1 test --manifest-path wasmhost/Cargo.toml --release --locked --test helper_corpus
stage helper-corpus-regen-2 env GOMAD_HELPER_CORPUS_REGENERATE=1 CARGO_BUILD_JOBS=1 RUST_TEST_THREADS=1 \
  cargo +1.99.1 test --manifest-path wasmhost/Cargo.toml --release --locked --test helper_corpus
rm -rf -- "${CARGO_TARGET_DIR:?}"
stage wasmmeasure-build "$qualification_go" build -trimpath=false -o "$QUAL_ROOT/bin/wasmmeasure" ./cmd/wasmmeasure
sha256sum -- "$QUAL_ROOT/bin/wasmmeasure" > "$QUAL_ROOT/wasmmeasure.sha256"
stage terminal-controls "$qualification_go" test -v -tags test_dep -count=1 -timeout=15m -run '^TestTerminalExit' ./wasi
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
sha256sum -- "$installed_helper" "$admitted_helper" > "$QUAL_ROOT/helper.sha256"
export GOMAD_WASM_TEST_HELPER="$installed_helper"
helper_ready=true
check_source
stage conformance-helper-build cargo +1.99.1 build --manifest-path wasmhost/Cargo.toml --release --locked \
  --no-default-features --features conformance --target-dir "$scratch/conformance-target"
mkdir -p "$QUAL_ROOT/helpers/conformance"
install -m 755 "$scratch/conformance-target/release/gomad-wasmhost" "$QUAL_ROOT/helpers/conformance/gomad-wasmhost"
cmp "$scratch/conformance-target/release/gomad-wasmhost" "$QUAL_ROOT/helpers/conformance/gomad-wasmhost"
rm -rf -- "${scratch:?}/conformance-target"
export GOMAD_WASM_CONFORMANCE_HELPER="$QUAL_ROOT/helpers/conformance/gomad-wasmhost"
export GOMAD_WASM_CONFORMANCE_REQUIRED=1
python3 "$driver_dir/helper-files.py" "$installed_helper" "$admitted_helper" "$GOMAD_WASM_CONFORMANCE_HELPER"
sha256sum -- "$GOMAD_WASM_CONFORMANCE_HELPER" > "$QUAL_ROOT/conformance-helper.sha256"
stage ordinary-helper-publication publish_ordinary_helper
cd "$module"
stage ordinary make test-wasm "GOMAD_STOCK_GO=$qualification_go"
cd "$wasm"
named_test retained-durable-restart TestProviderActualCrashDurableRestart 15m ./cluster
# fn-178 in-process cluster gates on the authenticated release helper.
stage cluster-provider "$qualification_go" test -v -tags test_dep -count=1 -timeout=30m ./cluster
stage cluster-functional "$qualification_go" test -v -tags test_dep -count=1 -timeout=30m \
  -run 'TestCluster|TestBrokerOrders|TestWorld|TestClock|TestStorageCrash|TestImplementationIdentity' ./wasi
stage busy-boundaries busy12 "$qualification_go" test -v -tags test_dep -count=1 -timeout=30m \
  -run '^(TestClusterOutputBoundaryParity|TestClusterBrokerDurableRestartParity)$' ./wasi
stage busy-grouped busy12 "$qualification_go" test -v -tags test_dep -count=10 -timeout=30m \
  -run '^TestProviderGroupedAcknowledgmentCrashSourceLessReplay$' ./cluster
export GOMAD_WASM_TEMPORAL_RECOVERY_ROOT="$QUAL_ROOT/network-streams"
named_test network-streams TestBrokerIndependentGuestStreamsReplay 30m ./wasi
unset GOMAD_WASM_TEMPORAL_RECOVERY_ROOT
jq -e --arg helper "$expected_helper" '.network_owner == "rust-cluster" and (.network_owner_implementation_sha256 == $helper or .network_owner_implementation_sha256 == ("sha256:" + $helper)) and
  .reference_network_owner == "go-broker-reference" and .distinct_choice_traces >= 2' "$QUAL_ROOT/network-streams/broker-network-streams-report.json" > /dev/null
stage retained-cluster "$qualification_go" test -v -tags test_dep -count=1 -timeout=30m -run 'Retained|Replay' ./cluster
cd "$wasm"

# The actual ./tests producer supplies authenticated preparation inputs. The
# measured helper is its retained admission copy, byte-identical to installation.
export GOMAD_WASM_TERMINAL_EXIT_PREPARE_ROOT="$QUAL_ROOT/terminal-module"
named_test terminal-module TestPrepareTemporalTerminalExitModule 60m ./backend
unset GOMAD_WASM_TERMINAL_EXIT_PREPARE_ROOT
one_path module_receipt "$QUAL_ROOT/terminal-module" temporal-producer.json
export GOMAD_WASM_TERMINAL_EXIT_MODULE_RECEIPT="$module_receipt"
export GOMAD_WASM_TERMINAL_EXIT_MODULE="$(jq -er '.Prepared.Path' "$module_receipt")"
export GOMAD_WASM_TERMINAL_EXIT_HELPER_RECEIPT="$helper_admission"
export GOMAD_WASM_TERMINAL_EXIT_ROOT="$QUAL_ROOT/terminal-exit"
export GOMAD_WASM_TEST_HELPER="$admitted_helper"
capture_host_cpu terminal-host
named_test terminal-host-admission TestDiagnosticPerformanceCurrentHostAdmission 15m ./internal/measure
jq -n --arg head "$expected_head" --arg source "sha256:$expected_source" \
  --arg helper "sha256:$expected_helper" --arg module "$(digest_file "$GOMAD_WASM_TERMINAL_EXIT_MODULE")" \
  --arg cpu "$(digest_file "$QUAL_ROOT/terminal-host-cpuinfo.txt")" \
  --arg receipt "$module_receipt" --arg admission "$helper_admission" \
  '{source_revision:$head,source_manifest_sha256:$source,helper_sha256:$helper,module_sha256:("sha256:"+$module),
    cpuinfo_sha256:("sha256:"+$cpu),module_receipt:$receipt,helper_admission:$admission,
    required_host:"linux/arm64",cpuinfo_byte_limit:1048576}' > "$QUAL_ROOT/terminal-host-binding.json"
stage terminal-exit clean_measure 60m terminal-exit \
  --stock-go "$qualification_go" --test-helper "$admitted_helper" \
  --terminal-exit-root "$QUAL_ROOT/terminal-exit" \
  --terminal-exit-helper-receipt "$helper_admission" \
  --terminal-exit-module "$GOMAD_WASM_TERMINAL_EXIT_MODULE" \
  --terminal-exit-module-receipt "$module_receipt"
sha256sum -c "$QUAL_ROOT/terminal-host-cpuinfo.sha256" > /dev/null
prune_bulk "$QUAL_ROOT/terminal-module" terminal-module
prune_bulk "$QUAL_ROOT/terminal-exit" terminal-exit
export GOMAD_WASM_TEST_HELPER="$installed_helper"
unset GOMAD_WASM_TERMINAL_EXIT_MODULE_RECEIPT GOMAD_WASM_TERMINAL_EXIT_MODULE
unset GOMAD_WASM_TERMINAL_EXIT_HELPER_RECEIPT GOMAD_WASM_TERMINAL_EXIT_ROOT

export GOMAD_WASM_INTEGRATED_SOURCE_MANIFEST="$QUAL_ROOT/source.sha256"
export GOMAD_WASM_QUALIFICATION_MANIFEST="$QUAL_ROOT/source.sha256"
export GOMAD_WASM_INTEGRATED_SOURCE_SHA256="sha256:$expected_source"
export GOMAD_WASM_QUALIFICATION_ROOT="$QUAL_ROOT/runtime"
for index in $(seq 0 31); do
  export GOMAD_WASM_QUALIFICATION_INDEX="$index"
  named_test "runtime-$index" TestCooperativeRuntimeQualification 10m ./wasi
done
unset GOMAD_WASM_QUALIFICATION_INDEX
named_test runtime-aggregate TestCooperativeRuntimeQualificationAggregate 15m ./wasi
export GOMAD_WASM_TASK4_EVIDENCE_ROOT="$QUAL_ROOT/artifact"
named_test artifact TestIntegratedFailureArtifactFreshObservedReplays 15m ./backend
unset GOMAD_WASM_TASK4_EVIDENCE_ROOT
named_test operator TestInstalledOperatorDiscoversInspectsAndReplaysWithoutBuildInputs 30m ./command
export GOMAD_WASM_TEMPORAL_RECOVERY_ROOT="$QUAL_ROOT/storage"
named_test storage TestTemporalCommittedWorkflowFreshGuestCrashReopen 90m ./wasi
prune_bulk "$QUAL_ROOT/storage" storage
export GOMAD_WASM_TEMPORAL_INSPECTION_ROOT="$QUAL_ROOT/inspection"
named_test temporal-inspection TestTemporalInspectionRetainedFailure 90m ./backend
for owner in shard workflow onebox; do
  rg -q "^[[:space:]]+--- PASS: TestTemporalInspectionRetainedFailure/$owner " "$QUAL_ROOT/temporal-inspection.log"
done
unset GOMAD_WASM_TEMPORAL_INSPECTION_ROOT
prune_bulk "$QUAL_ROOT/inspection" inspection
export GOMAD_WASM_NETWORK_TEMPORAL_ROOT="$QUAL_ROOT/network"
named_test network TestTemporalNetworkNamedResetRetriesAndExactReplay 90m ./wasi
unset GOMAD_WASM_NETWORK_TEMPORAL_ROOT
prune_bulk "$QUAL_ROOT/network" network
export GOMAD_WASM_TEMPORAL_RECOVERY_ROOT="$QUAL_ROOT/clusters"
named_test clusters TestTemporalIndependentClustersPartitionHealWholeClusterCrash 90m ./wasi
unset GOMAD_WASM_TEMPORAL_RECOVERY_ROOT
one_path clusters_owner "$QUAL_ROOT/clusters" two-cluster-owner.json
jq -e --arg helper "$expected_helper" '.network_owner == "rust-cluster" and (.network_owner_implementation_sha256 == $helper or .network_owner_implementation_sha256 == ("sha256:" + $helper))' "$clusters_owner" > /dev/null

# Match the current producer's ordered concatenation, never a historical pin.
sha256sum -- "$wasm/testdata/temporal_storage/recovery_test.go" "$wasm/testdata/temporal_broker/replication_test.go" > "$QUAL_ROOT/broker-inputs.sha256"
fixture_hash="$(cat -- "$wasm/testdata/temporal_storage/recovery_test.go" "$wasm/testdata/temporal_broker/replication_test.go" | sha256sum | cut -d ' ' -f1)"
fixture_relative=".flow/tmp/q1-temporal-staged"
cmp "$QUAL_REPO/$fixture_relative/recovery_test.go" "$wasm/testdata/temporal_storage/recovery_test.go"
cmp "$QUAL_REPO/$fixture_relative/replication_test.go" "$wasm/testdata/temporal_broker/replication_test.go"
printf '%s\n' "$fixture_relative" > "$QUAL_ROOT/broker-source.path"
export GOMAD_WASM_QUALIFICATION_ROOT="$QUAL_ROOT/workloads"
export GOMAD_WASM_TEMPORAL_QUALIFICATION_INPUTS="$QUAL_ROOT/clusters"
export GOMAD_WASM_TEMPORAL_PREPARATION_SOURCE="./$fixture_relative"
export GOMAD_WASM_QUALIFY_WORKLOADS=1
for entry in 'fixture controlled TestActualMatchedFixtureQualification' 'fixture cooperative TestActualMatchedFixtureQualification' \
  'sqlite cooperative TestActualSQLiteDurableQualification' 'onebox cooperative TestActualOneboxQualification' \
  'multi-instance cooperative TestActualTemporalClusterQualification'; do
  read -r workload profile gate <<< "$entry"
  export GOMAD_WASM_QUALIFICATION_PROFILE="$profile"
  for cell in success canary; do
    export GOMAD_WASM_QUALIFICATION_CELL="$cell"
    named_test "workload-$workload-$profile-$cell" "$gate" 90m ./qualification
  done
done
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
readonly primary_sha="sha256:$(digest_file "$GOMAD_WASM_SQLITE_NATIVE_CATALOGUE_RESULT")"
printf '%s\n' "$GOMAD_WASM_SQLITE_NATIVE_CATALOGUE_RESULT" > "$QUAL_ROOT/primary-sqlite.path"
jq -e '[.NativeCrashCatalogues[] | select(.Handle.node == "sqlite" and .Handle.incarnation == 2 and .Volume == "sqlite")] | length == 1 and (.[0].Enumeration.Complete == true) and (.[0].Enumeration.States | length == 4)' "$GOMAD_WASM_SQLITE_NATIVE_CATALOGUE_RESULT" > /dev/null
mkdir -p "$selected_dir"
files=("$selected_dir"/*)
test "${#files[@]}" -eq 0
export GOMAD_WASM_QUALIFICATION_PROFILE=cooperative
for index in 0 1 2 3; do
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
done
unset GOMAD_WASM_SQLITE_SELECTION_INDEX
export GOMAD_WASM_QUALIFICATION_ROOT="$QUAL_ROOT/workloads"
named_test workload-corpus-after-relocation TestActualMandatoryWorkloadCorpus 15m ./qualification
export GOMAD_WASM_QUALIFICATION_ROOT="$QUAL_ROOT/sqlite-selections"
named_test sqlite-catalogue TestActualSQLiteSelectionCatalogue 15m ./qualification
check_primary
files=("$primary_dir"/*)
sha256sum -- "${files[@]}" > "$QUAL_ROOT/primary-after.sha256"
cmp "$QUAL_ROOT/primary-before.sha256" "$QUAL_ROOT/primary-after.sha256"
for index in 0 1 2 3; do sha256sum -c "$QUAL_ROOT/sqlite-selection-$index-after.sha256" > /dev/null; done
check_source
jq -s '[.[] | .Manifest | {ID,Profile,HelperSHA256,ModelSHA256,RuntimeSHA256,EngineSHA256,CompilerSHA256,IntegratedSourceSHA256}]' \
  "${primary_results[@]}" > "$QUAL_ROOT/current-identities.json"
printf '%s\n' 'Matched-release candidate comparison remains separately OPEN; no speedup or whole-task completion asserted. Native execution remains deferred fn-128/fn-149.' > "$QUAL_ROOT/remaining-obligations.txt"
printf '%s PASS-STAGES qualification-driver (bulk pruning follows)\n' "$(date -u +%FT%TZ)" >> "$QUAL_ROOT/stages.log"
prune_bulk "$QUAL_ROOT/runtime" runtime
prune_bulk "$QUAL_ROOT/artifact" artifact
printf '%s PASS qualification-driver module=%s matched-release=OPEN\n' "$(date -u +%FT%TZ)" "$module_name" | tee -a "$QUAL_ROOT/stages.log"

