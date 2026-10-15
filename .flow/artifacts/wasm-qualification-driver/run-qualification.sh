#!/usr/bin/env bash
# fn-178 lifecycle/client batch frozen qualification, adapted from the fn-176
# 84-stage driver (Uq7Ppu). Arguments: exact frozen HEAD, explicit lint base,
# absolute fresh evidence root, absolute fresh scratch root.
# No generation is performed: stale generated sources fail the frozen candidate.
set -euo pipefail
set -o noclobber
shopt -s nullglob dotglob
export LC_ALL=C
test "$#" -eq 4
readonly expected_head="$1" lint_base="$2"
[[ "$expected_head" =~ ^[0-9a-f]{40}$ ]]
export QUAL_REPO=/Users/stephan/Workspace/skunkworks/gomad-fn178-frozen-20261015
export QUAL_ROOT="$3"
readonly scratch="$4"
[[ "$QUAL_ROOT" == /* && "$scratch" == /* ]]
test -d "$QUAL_ROOT" && test ! -e "$QUAL_ROOT/stages.log"
module_candidates=()
for candidate in "$QUAL_REPO/tools/gomad3" "$QUAL_REPO/tools/gomad"; do
  [[ ! -f "$candidate/go.mod" ]] || module_candidates+=("$candidate")
done
test "${#module_candidates[@]}" -eq 1
readonly module="${module_candidates[0]}"
readonly module_name="${module##*/}"
readonly wasm="$module/internal/backends/wasm"
readonly installed_helper="$wasm/wasmhost/target/release/gomad3-wasmhost"
readonly primary_dir="$QUAL_ROOT/workloads/wasm-qualifications/v1"
readonly selected_dir="$QUAL_ROOT/sqlite-selections/wasm-qualifications/v1"
export GOMAD3_STOCK_GO="/Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad/.toolchain/stock/878c9b2935e2c9e6/go/bin/go"
export GOMAD_WASM_STOCK_GO="$GOMAD3_STOCK_GO"
export GOENV=off GOTOOLCHAIN=local GOWORK=off GOFLAGS=
export PATH="$(dirname "$GOMAD3_STOCK_GO"):$PATH"
unset GOROOT GOMADSEED GOMAD3_CHILD_SEED GOEXPERIMENT GOOS GOARCH
# Do not inherit opt-in producer roots, selection indices or diagnostic inputs.
while IFS= read -r variable; do unset "$variable"; done < <(compgen -A variable GOMAD_WASM_)
export GOMAD_WASM_STOCK_GO="$GOMAD3_STOCK_GO"
mkdir "$scratch"
export TMPDIR="$scratch" GOCACHE="$scratch/go-cache"
export CARGO_TARGET_DIR="$scratch/rust-check-target" CARGO_INCREMENTAL=0
export RUSTUP_TOOLCHAIN=1.94.1 RUSTC_WRAPPER= RUSTC_WORKSPACE_WRAPPER=
unset RUSTFLAGS CARGO_ENCODED_RUSTFLAGS
export GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD=1
source_manifest_ready=false
helper_ready=false
expected_helper=
expected_source=

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
  git -C "$QUAL_REPO" diff --quiet
  git -C "$QUAL_REPO" diff --cached --quiet
  if [[ "$source_manifest_ready" == true ]]; then
    test "$(digest_file "$QUAL_ROOT/source.sha256")" = "$expected_source"
    sha256sum -c "$QUAL_ROOT/source.sha256" > /dev/null
    sha256sum -c "$QUAL_ROOT/compiler.sha256" > /dev/null
  fi
  if [[ "$helper_ready" == true ]]; then
    test "$(digest_file "$installed_helper")" = "$expected_helper"
    test "$(digest_file "$admitted_helper")" = "$expected_helper"
    sha256sum -c "$QUAL_ROOT/helper.sha256" > /dev/null
  fi
}
stage() {
  local label="$1" status=0
  shift
  check_source
  printf '%s START %s\n' "$(date -u +%FT%TZ)" "$label" | tee -a "$QUAL_ROOT/stages.log"
  printf 'cwd=%q\n' "$PWD" > "$QUAL_ROOT/$label.command"
  printf '%q ' "$@" >> "$QUAL_ROOT/$label.command"
  printf '\n' >> "$QUAL_ROOT/$label.command"
  "$@" > "$QUAL_ROOT/$label.log" 2>&1 || status=$?
  printf '%s EXIT %s %d\n' "$(date -u +%FT%TZ)" "$label" "$status" | tee -a "$QUAL_ROOT/stages.log"
  check_source
  test "$status" -eq 0
}
named_test() {
  local label="$1" name="$2" timeout="$3" package="$4"
  stage "$label" "$GOMAD3_STOCK_GO" test -v -tags test_dep -count=1 \
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
test -x "$GOMAD3_STOCK_GO"
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
"$GOMAD3_STOCK_GO" version > "$QUAL_ROOT/go-version.log"
rg -q 'go1\.27\.1 linux/arm64$' "$QUAL_ROOT/go-version.log"
RUSTC="$(rustup which --toolchain 1.94.1 rustc)"
export RUSTC
"$RUSTC" -vV > "$QUAL_ROOT/rust-version.log"
rg -q '^rustc 1\.94\.1 ' "$QUAL_ROOT/rust-version.log"
cargo +1.94.1 --version > "$QUAL_ROOT/cargo-version.log"
sha256sum -- "$GOMAD3_STOCK_GO" "$RUSTC" > "$QUAL_ROOT/compiler.sha256"

# Retain absolute source paths, including renamed module and shared root consumers.
git ls-files -z -- "tools/$module_name" tools/gomad3sim tests/gomadfunctional \
  go.mod go.sum go.work go.work.sum Makefile cmd/tools/lintcode .github > "$QUAL_ROOT/source-paths.zlist"
while IFS= read -r -d '' path; do
  [[ "$path" != *$'\n'* && "$path" != *$'\\'* ]]
  test -f "$QUAL_REPO/$path" && test ! -L "$QUAL_REPO/$path"
  sha256sum -- "$QUAL_REPO/$path"
done < "$QUAL_ROOT/source-paths.zlist" > "$QUAL_ROOT/source.sha256"
test -s "$QUAL_ROOT/source.sha256"
test "$(wc -l < "$QUAL_ROOT/source.sha256")" -le 20000
test "$(wc -c < "$QUAL_ROOT/source.sha256")" -le 16777216
expected_source="$(digest_file "$QUAL_ROOT/source.sha256")"
source_manifest_ready=true
printf 'sha256:%s\n' "$expected_source" > "$QUAL_ROOT/source.digest"
check_source

cd "$module"
stage validate make validate
stage architecture env GOMAD3_ARCH_ALL_PLATFORMS=1 "$GOMAD3_STOCK_GO" test -v -tags test_dep -count=1 -timeout=15m ./internal/archcheck
stage portable "$GOMAD3_STOCK_GO" test -v -tags test_dep -count=1 -timeout=20m ./internal/model/... ./internal/evidence/... ./internal/host/... ./internal/backends/target/...
stage compile-only "$GOMAD3_STOCK_GO" test -exec /bin/true -tags test_dep -count=1 -run '^$' ./internal/backends/wasm/... ./internal/runner/... ./cmd/...
stage native-profile-inventory "$GOMAD3_STOCK_GO" test -v -tags test_dep -count=1 -timeout=15m \
  -run '^TestDeterministicProfileCompatibilityGolden$' ./internal/backends/native/transcript
# This reviewed function validates both portable inventory/implementation rows
# before its unsupported-host guard. Only that precise native-frame SKIP is valid.
test "$(rg -c '^=== RUN   TestDeterministicProfileCompatibilityGolden$' "$QUAL_ROOT/native-profile-inventory.log")" -eq 1
test "$(rg -c '^--- SKIP: TestDeterministicProfileCompatibilityGolden ' "$QUAL_ROOT/native-profile-inventory.log")" -eq 1
test "$(rg -c '^[[:space:]]+profile_test.go:[0-9]+: no profile golden for linux/arm64$' "$QUAL_ROOT/native-profile-inventory.log")" -eq 1
test "$(rg -c -- '--- SKIP:' "$QUAL_ROOT/native-profile-inventory.log")" -eq 1
if rg -q -- '--- (FAIL|PASS):|^FAIL' "$QUAL_ROOT/native-profile-inventory.log"; then exit 1; fi
printf '%s\n' 'Portable Darwin/arm64 and Linux/amd64 inventory and implementation assertions reached the reviewed Linux/arm64 host guard without failure. Unsupported native-frame branch SKIPPED; no native execution pass.' > "$QUAL_ROOT/native-profile-disposition.txt"
named_test native-bootstrap-pin TestPortableBootstrapPlatformGoldens 15m ./internal/backends/native/transcript
for platform in darwin/arm64 linux/amd64; do
  rg -q "^[[:space:]]+--- PASS: TestPortableBootstrapPlatformGoldens/$platform " "$QUAL_ROOT/native-bootstrap-pin.log"
done
named_test native-semantic-pin TestSemanticInstrumentationIdentityBindsProbeNamesAndStableIDs 15m ./internal/backends/native/transcript
named_test native-minimizer-preservation TestChoiceModePreservesNativeControllerAndStateEncoding 15m ./internal/runner/internal/minimizer
stage native-source "$GOMAD3_STOCK_GO" test -v -tags test_dep -count=1 -timeout=15m \
  ./internal/backends/native/sourceinventory ./internal/backends/native/toolchain/version ./internal/backends/native/generation/...
cd "$QUAL_REPO"
stage lint-router "$GOMAD3_STOCK_GO" test -v -tags test_dep -count=1 ./cmd/tools/lintcode
stage lint-router-vet "$GOMAD3_STOCK_GO" vet -tags test_dep ./cmd/tools/lintcode
actionlint_version="$(sed -n 's/^ACTIONLINT_VER := //p' Makefile)"
test "$actionlint_version" = v1.7.7
mkdir "$QUAL_ROOT/bin"
stage actionlint-build env "GOBIN=$QUAL_ROOT/bin" "$GOMAD3_STOCK_GO" install "github.com/rhysd/actionlint/cmd/actionlint@$actionlint_version"
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
stage lint-root make lint-code-fast "GOLANGCI_LINT_BASE_REV=$lint_base" GOLANGCI_LINT_FIX=false \
  "GOLANGCI_LINT=$QUAL_REPO/.tmp/linuxbin/golangci-lint-v2.13.0" "ERRORTYPE=$QUAL_REPO/.tmp/linuxbin/errortype"
lint_target="lint-code-$module_name"
if ! rg -q "^$lint_target:" Makefile; then lint_target=lint-code-gomad-wasm; fi
sed -n "/^$lint_target:/,+1p" Makefile | rg -q -- "-module tools/$module_name$"
stage lint-module make "$lint_target" "GOLANGCI_LINT_BASE_REV=$lint_base" GOLANGCI_LINT_FIX=false \
  "GOLANGCI_LINT=$QUAL_REPO/.tmp/linuxbin/golangci-lint-v2.13.0" "ERRORTYPE=$QUAL_REPO/.tmp/linuxbin/errortype"

cd "$wasm"
stage rust-fmt cargo +1.94.1 fmt --manifest-path wasmhost/Cargo.toml --check
stage rust-default-test cargo +1.94.1 test --manifest-path wasmhost/Cargo.toml --release --locked --no-default-features
stage rust-diagnostic-test cargo +1.94.1 test --manifest-path wasmhost/Cargo.toml --release --locked --no-default-features --features diagnostic-profile
stage rust-default-clippy cargo +1.94.1 clippy --manifest-path wasmhost/Cargo.toml --release --locked --no-default-features --all-targets -- -D warnings
stage rust-diagnostic-clippy cargo +1.94.1 clippy --manifest-path wasmhost/Cargo.toml --release --locked --no-default-features --features diagnostic-profile --all-targets -- -D warnings
stage terminal-controls "$GOMAD3_STOCK_GO" test -v -tags test_dep -count=1 -timeout=15m -run '^TestTerminalExit' ./wasi
# This producer owns pinned Cargo/rustc provenance, no wrappers, empty features,
# raw build capture and the single-link immutable helper admission receipt.
export GOMAD_WASM_TERMINAL_EXIT_HELPER_BUILD_ROOT="$QUAL_ROOT/helper-build"
named_test helper-build TestBuildTerminalExitReleaseHelper 60m ./wasi
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
install_tmp="$(mktemp "$wasm/wasmhost/target/release/qualification-helper.XXXXXX")"
cp -- "$admitted_helper" "$install_tmp"
chmod 755 "$install_tmp"
cmp "$admitted_helper" "$install_tmp"
mv -f -- "$install_tmp" "$installed_helper"
sha256sum -- "$installed_helper" "$admitted_helper" > "$QUAL_ROOT/helper.sha256"
export GOMAD_WASM_TEST_HELPER="$installed_helper"
helper_ready=true
check_source
cd "$module"
stage ordinary make test-wasm "GOMAD3_STOCK_GO=$GOMAD3_STOCK_GO"
cd "$wasm"
named_test retained-durable-restart TestProviderActualCrashDurableRestart 15m ./cluster
# fn-178 in-process cluster gates on the authenticated release helper.
stage cluster-provider "$GOMAD3_STOCK_GO" test -v -tags test_dep -count=1 -timeout=30m ./cluster
stage cluster-functional "$GOMAD3_STOCK_GO" test -v -tags test_dep -count=1 -timeout=30m \
  -run 'TestCluster|TestBrokerOrders|TestWorld|TestClock|TestStorageCrash|TestImplementationIdentity' ./wasi
stage busy-boundaries busy12 "$GOMAD3_STOCK_GO" test -v -tags test_dep -count=1 -timeout=30m \
  -run '^(TestClusterOutputBoundaryParity|TestClusterBrokerDurableRestartParity)$' ./wasi
stage busy-grouped busy12 "$GOMAD3_STOCK_GO" test -v -tags test_dep -count=10 -timeout=30m \
  -run '^TestProviderGroupedAcknowledgmentCrashSourceLessReplay$' ./cluster
export GOMAD_WASM_TEMPORAL_RECOVERY_ROOT="$QUAL_ROOT/network-streams"
named_test network-streams TestBrokerIndependentGuestStreamsReplay 30m ./wasi
unset GOMAD_WASM_TEMPORAL_RECOVERY_ROOT
jq -e --arg helper "$expected_helper" '.network_owner == "rust-cluster" and (.network_owner_implementation_sha256 == $helper or .network_owner_implementation_sha256 == ("sha256:" + $helper)) and
  .reference_network_owner == "go-broker-reference" and .distinct_choice_traces >= 2' "$QUAL_ROOT/network-streams/broker-network-streams-report.json" > /dev/null
stage retained-cluster "$GOMAD3_STOCK_GO" test -v -tags test_dep -count=1 -timeout=30m -run 'Retained|Replay' ./cluster
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
named_test terminal-host-admission TestDiagnosticPerformanceCurrentHostAdmission 15m ./wasi
jq -n --arg head "$expected_head" --arg source "sha256:$expected_source" \
  --arg helper "sha256:$expected_helper" --arg module "$(digest_file "$GOMAD_WASM_TERMINAL_EXIT_MODULE")" \
  --arg cpu "$(digest_file "$QUAL_ROOT/terminal-host-cpuinfo.txt")" \
  --arg receipt "$module_receipt" --arg admission "$helper_admission" \
  '{source_revision:$head,source_manifest_sha256:$source,helper_sha256:$helper,module_sha256:("sha256:"+$module),
    cpuinfo_sha256:("sha256:"+$cpu),module_receipt:$receipt,helper_admission:$admission,
    required_host:"linux/arm64",cpuinfo_byte_limit:1048576}' > "$QUAL_ROOT/terminal-host-binding.json"
named_test terminal-exit TestTemporalTerminalToProcessExit 60m ./wasi
sha256sum -c "$QUAL_ROOT/terminal-host-cpuinfo.sha256" > /dev/null
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
export GOMAD_WASM_NETWORK_TEMPORAL_ROOT="$QUAL_ROOT/network"
named_test network TestTemporalNetworkNamedResetRetriesAndExactReplay 90m ./wasi
unset GOMAD_WASM_NETWORK_TEMPORAL_ROOT
export GOMAD_WASM_TEMPORAL_RECOVERY_ROOT="$QUAL_ROOT/clusters"
named_test clusters TestTemporalIndependentClustersPartitionHealWholeClusterCrash 90m ./wasi
unset GOMAD_WASM_TEMPORAL_RECOVERY_ROOT
one_path clusters_owner "$QUAL_ROOT/clusters" two-cluster-owner.json
jq -e --arg helper "$expected_helper" '.network_owner == "rust-cluster" and (.network_owner_implementation_sha256 == $helper or .network_owner_implementation_sha256 == ("sha256:" + $helper))' "$clusters_owner" > /dev/null

# Match the current producer's ordered concatenation, never a historical pin.
sha256sum -- "$wasm/testdata/temporal_storage/recovery_test.go" "$wasm/testdata/temporal_broker/replication_test.go" > "$QUAL_ROOT/broker-inputs.sha256"
fixture_hash="$(cat -- "$wasm/testdata/temporal_storage/recovery_test.go" "$wasm/testdata/temporal_broker/replication_test.go" | sha256sum | cut -d ' ' -f1)"
fixture_relative=".tmp/wasm-backend-temporal/temporal_broker-${fixture_hash:0:16}"
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
printf '%s PASS frozen-qualification module=%s\n' "$(date -u +%FT%TZ)" "$module_name" | tee -a "$QUAL_ROOT/stages.log"
