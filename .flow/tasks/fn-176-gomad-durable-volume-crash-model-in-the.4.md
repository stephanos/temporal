---
satisfies: [R1, R2]
---
# fn-176-gomad-durable-volume-crash-model-in-the.4 Rust storage fault engine, storage-model validation and namespace-to-volume routing

## Description
Port the storage fault engine, storage-model validation and the namespace-to-volume routing into Rust (R1, R2), so that a storage-configured Rust namespace answers every storage operation, including faults and `crash-before`.

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/wasmhost/src/volume/faults.rs` (new), `tools/gomad/internal/backends/wasm/wasmhost/src/volume/config.rs` (new: validation), `tools/gomad/internal/backends/wasm/wasmhost/src/namespace.rs` (storage routing branches), `tools/gomad/internal/backends/wasm/wasmhost/src/conformance.rs`
**Touches:** [tools/gomad/internal/backends/wasm/wasmhost/src/volume/**, tools/gomad/internal/backends/wasm/wasmhost/src/namespace.rs, tools/gomad/internal/backends/wasm/wasmhost/src/conformance.rs, tools/gomad/internal/backends/wasm/wasmhost/src/abi.rs, tools/gomad/internal/backends/wasm/wasmhost/src/engine.rs, tools/gomad/internal/backends/wasm/wasmhost/src/model.rs, tools/gomad/internal/backends/wasm/wasmhost/src/protocol.rs, tools/gomad/internal/backends/wasm/wasmhost/protocol.md, tools/gomad/internal/backends/wasm/wasi/helper_model.go, tools/gomad/internal/backends/wasm/wasi/process.go, tools/gomad/internal/backends/wasm/wasi/process_test.go, tools/gomad/internal/backends/wasm/wasi/helper_model_test.go]

Conductor-authorized integrated-batch correction (2026-10-15): address the task .4 integration review's split-output accounting defect now, ahead of its task .6 owner. A remote stdout write followed by local stdout close and an invalid write must share Go's offered-output capacity and terminal count; later remote writes must adopt prior local charges. Preserve descriptor-based routing. Add focused Go-owner versus split regressions, retain the same review receipt, and cite this correction in task .6 evidence. Original acceptance remains unchanged.

### Approach
- Port `storageDecision` (wasi/volume.go:233):
  - `resource\0op` occurrence counting
  - first-match fault lookup
  - action mapping: `io-error`/`sync-error` → 29, `no-space` → 51, `short-write`/`torn-write` shorten the count and torn also gives 29, `crash-before` → modeled storage crash
  - the runtime `invalid("storage write fault must shorten the matched write")`
- `StorageEvent` lines: render the exact Go JSON line, `json.Marshal(StorageEvent)` field order with omitempty, so the storage transcript bytes and the `capacity("storage-transcript")` boundary against `Limits.TranscriptBytes` match Go byte for byte.
- Port `validateStorageModel` (:71) completely: volume/fault/lock-database bounds, writable-preopen and capacity checks, the fault action/operation matrix, ambiguity checks, and lock-path rules. `crash-after` stays rejected. Rust re-validates in addition to Go, and either side's refusal is `invalid` before execution.
- SQLite lock exception: `sqliteLockPath`/`storagePath` (:392-414). `.lock`, `-shm` and `-shm/…` nodes stay namespace-only and never reach the volume.
- Namespace routing: add the storage branches the Go namespace has at namespace.go 335-388 (close, sync), 457-465 (read), 538 (`storageWrite`), 586-598 (truncate), 731-757 (open), 806-816 (mkdir), 836-842 (remove) and 943-953 (rename), with `refreshStorageNode` and `storageResizeAllowed` semantics. Keep namespace filesystem-bytes and volume capacity as distinct classifications.
- `crash-before` produces a typed Rust error that fn-176.5 maps to a helper-originated `storage_crash` termination.

### Investigation targets
**Required:**
- `tools/gomad/internal/backends/wasm/wasi/volume.go` (whole file)
- `tools/gomad/internal/backends/wasm/wasi/namespace.go:335-388,425-610,674-960` (storage branches)
- `tools/gomad/internal/backends/wasm/wasi/volume_test.go:23-280`

## Acceptance
- [x] The full volume conformance corpus passes through the Rust namespace with storage routing (not only through the volume module directly), including every fault action, storage-transcript capacity and the lock exception.
- [x] Invalid storage models (each `validateStorageModel` rule, including `crash-after`) are refused by the Rust validator with `invalid` (Rust unit tests mirroring `wasi/volume_test.go` cases).


## Done summary
Verified the inherited Rust fault engine, all storage-model validation rules and namespace-to-volume routing against Go, adding four missing validator predicate cases. Integration review found a split-output accounting defect; conductor-authorized early task .6 correction `2668af049e` now shares offered-byte accounting across local/remote writes and reconciles Go callback totals, with a committed failing reproduction `53320ca608` and passing real-Run parity controls.

Acceptance/coverage record: `.flow/artifacts/fn176-task4-acceptance-2026-10-15.md`. Implementation provenance: merge `cd2cf0c9f644345a0cdf5807dd5d8f4d52a5474d`, original fault/config/routing `6318cad3683cdca433ce624a9e2b17c23e6cbf6c`, and original review corrections `629e735ca6ef1777d238b156e52600fe5cabe69c`. Lifecycle baseline is `ab39e8ec3c406c4181c293388929b0e06ce23168`; review used historical base `2237e0276f50f3b4b4b244f32a564ec17caca110` through `2668af049eea818280063d115969d5d44e8669ea`.

baseline: green (Rust Quick command executed pre-edit; source-identical task .3 conformance/Go observations reused with conductor authorization, then all canonical Quick commands rerun after the production correction).

GATE_SKIPPED:unittest:green-receipt 6fa6bbd8 - pre-correction verify reused its Rust pass; superseded by production correction and final gate runs

Final gates: Rust 196 unit + 150 integration tests (2 existing ignored); exact volume Quick command 54 unit + 1 integration; conformance 46 scripts + 4 invalid recovery controls + 6 cross-directory recovery cases; canonical Go storage 61.657s; explicit-compiler transport/helper/split/broker controls 13.345s; fmt, default/conformance clippy and configured fast lint pass. Final default helper SHA256: `896a19842b7b5db8edf87e0fdf201927ced30ea011ba65ee18e6f9bf83e43660`.

`TestStorageSplitOutputBudgetAfterClose` covers local closed-write capacity, below-limit EBADF followed by remote stderr, and subsequent remote capacity, checking offered totals, actual stdout/stderr, call/transcript/storage history and remote callback counts across both owners. `TestHelperBridgeSplitOutputStateAdmission` and `TestTransportSplitOutputTerminalBounds` reject regressing/over-limit/incorrectly owned counters. Rust partial/errno tests now intentionally assert offered-byte charging, matching Go and the public protocol, while preserving exact actual-count, errno and buffer assertions. Namespace-image admission already binds its output limit to execute output_bytes; mismatch controls passed in the full Rust suite. Descriptor routing remains unchanged; allocation is monotonic and fd_renumber is unsupported.

The first broader transport/broker invocation was inconclusive because explicit GOMAD3_STOCK_GO was missing; the same selection passed with the required stock compiler. The first conformance clippy run caught an unused production-only setter; scoping it out of the conformance build resolved that source warning. No gate policy was changed.

Defect route:
- prior fixes: recent local engine history and bug memory showed no matching fix; unchecked: GitHub PRs/issues (HTTP 401); tracker inactive.
- diagnosis: confirmed separate remote/namespace counters with the real-Run failing reproduction; confirmed remote ABI errno refunds conflict with Go offered-byte accounting via Go-owner terminal refusal.
- introduced by: skipped bisect because no known-good split revision was identified; inherited split implementation predates this lifecycle baseline.
- base: regression fails at `53320ca6084b065301cbad4cec8bf9df36d145fc`; head: same table passes after `2668af049eea818280063d115969d5d44e8669ea`.
- live: no live app surface; actual helper processes run through Go Run in both ownership modes.

Task .6 must retain the early coordinated fix and its evidence. Whole-spec frozen qualification, Temporal committed-workflow gate, workloads and WASM SQLite selected-state/catalogue remain pending with the conductor. Native qualification remains deferred. All 18 external planning files retain their initial SHA256 hashes and remain unstaged.

Tier: implementer gpt-6.1-sol at medium (project pin)

stage: impl-review - ran [2026-10-15T06:11:13Z..2026-10-15T06:32:50Z] (Codex gpt-6.1-sol at high; first round correctness/contracts SHIP, integration NEEDS_WORK; same-receipt re-review SHIP. Reviewer and writer are from the same GPT/Codex family. Receipt: `/tmp/impl-review-receipt-657da2bc4466-fn-176-gomad-durable-volume-crash-model-in-the.4.json`.)

stage: plan-sync - skipped(config: planSync.enabled != true)

Tracker sync: n/a (bridge inactive)
## Evidence
- Commits: cd2cf0c9f644345a0cdf5807dd5d8f4d52a5474d, 6318cad3683cdca433ce624a9e2b17c23e6cbf6c, 629e735ca6ef1777d238b156e52600fe5cabe69c, 6fa6bbd8a25824580e1ae8d0ad100bb928d2df34, 53320ca6084b065301cbad4cec8bf9df36d145fc, 2668af049eea818280063d115969d5d44e8669ea
- Tests: baseline: green (pre-edit Rust 54 unit + 1 integration; source-identical task3 Go/conformance evidence reused per conductor, then rerun after production fix), TMPDIR=/tmp cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked volume (exit0; 54 unit +1 integration; .flow/tmp/fn176.4-rust-final.log), TMPDIR=/tmp cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked (exit0; 196 unit +150 integration, 2 existing ignored; .flow/tmp/fn176.4-rust-full.log), TMPDIR=/tmp make -C tools/gomad/internal/backends/wasm test-conformance (exit0; 46 volume corpus scripts +4 invalid raw recovery controls +6 cross-directory rename cases; .flow/tmp/fn176.4-conformance.log), cd tools/gomad && TMPDIR=/tmp go test -tags test_dep -count=1 -run 'TestVolumeConformance|TestStorage|TestSQLiteStorage' ./internal/backends/wasm/wasi (exit0; 61.657s final helper; .flow/tmp/fn176.4-go-storage-final.log), cd tools/gomad && TMPDIR=/tmp go test -tags test_dep -count=1 -run '^TestStorageSplitOutputBudgetAfterClose$' ./internal/backends/wasm/wasi (RED exit1 at53320ca608; same table GREEN in final canonical suite; .flow/tmp/fn176.4-output-red.log), cd tools/gomad && TMPDIR=/tmp go test -tags test_dep -count=1 -run 'TestStorageSplitOutputBudgetAfterClose|TestHelperBridgeSplitOutputStateAdmission|TestTransportSplitOutputTerminalBounds|TestSplit|TestHelperBridgeModel' ./internal/backends/wasm/wasi (exit0; .flow/tmp/fn176.4-output-controls.log), cd tools/gomad && TMPDIR=/tmp go test -tags test_dep -count=1 -run 'TestTransport|TestHelper|TestSplit|TestBrokerCrashRecoveryFreshGuest' ./internal/backends/wasm/wasi (inconclusive exit1: explicit GOMAD3_STOCK_GO missing; .flow/tmp/fn176.4-transport-broker.log), cd tools/gomad && TMPDIR=/tmp GOMAD3_STOCK_GO=/Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad/.toolchain/stock/878c9b2935e2c9e6/go/bin/go GOMAD_WASM_STOCK_GO=/Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad/.toolchain/stock/878c9b2935e2c9e6/go/bin/go go test -tags test_dep -count=1 -run 'TestTransport|TestHelper|TestSplit|TestBrokerCrashRecoveryFreshGuest' ./internal/backends/wasm/wasi (exit0; 13.345s; .flow/tmp/fn176.4-transport-broker-final.log), cargo +1.94.1 fmt --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --check (exit0), TMPDIR=/tmp cargo +1.94.1 clippy --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked --all-targets -- -D warnings (exit0; .flow/tmp/fn176.4-clippy-default.log), TMPDIR=/tmp cargo +1.94.1 clippy --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked --all-targets --features conformance -- -D warnings (exit0 after production-only setter cfg; .flow/tmp/fn176.4-clippy.log), TMPDIR=/tmp make lint-code-fast GOLANGCI_LINT=/Users/stephan/Workspace/skunkworks/gomad/temporal/.tmp/linuxbin/golangci-lint-v2.13.0 ERRORTYPE=/Users/stephan/Workspace/skunkworks/gomad/temporal/.tmp/linuxbin/errortype GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=ab39e8ec3c406c4181c293388929b0e06ce23168 (exit0; zero issues; .flow/tmp/fn176.4-lint.log), TMPDIR=/tmp make -C tools/gomad/internal/backends/wasm helper (exit0; final SHA256896a19842b7b5db8edf87e0fdf201927ced30ea011ba65ee18e6f9bf83e43660; .flow/tmp/fn176.4-helper-final.log), GATE_SKIPPED:unittest:green-receipt 6fa6bbd8 - pre-correction verify reused its Rust pass; superseded by production correction and final gate runs, sha256sum --check .flow/tmp/fn176.3-external-before.sha256 (exit0; all18 external planning files unchanged), flowctl validate --spec fn-176-gomad-durable-volume-crash-model-in-the --json (exit0)
- PRs: