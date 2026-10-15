---
satisfies: [R1, R2]
---
# fn-176-gomad-durable-volume-crash-model-in-the.3 Rust crash enumeration, selection, lifecycle and self-contained recovery image

## Description
Port crash enumeration, crash selection, lifecycle advance and durable cloning, and define the helper-owned recovery image (R1, R2). This is the semantic core that the committed-workflow crash/reopen and SQLite durable gates depend on.

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/wasmhost/src/volume/crash.rs` (new), `tools/gomad/internal/backends/wasm/wasmhost/src/volume/image.rs` (new), `tools/gomad/internal/backends/wasm/wasmhost/src/volume/mod.rs`, `tools/gomad/internal/backends/wasm/wasmhost/src/conformance.rs`
**Touches:** [tools/gomad/internal/backends/wasm/wasmhost/src/volume/**, tools/gomad/internal/backends/wasm/wasmhost/src/conformance.rs]

### Approach
- Port `EnumerateCrashStates` (volume_generated.go:291):
  - `CrashEnumerationLimits{States, Operations, Depth, Bytes, Work}`
  - the resumable `CrashExploration` cursor with `pendingSHA256` validation and stale-cursor rejection
  - `enumerationBudget.Take()` charging Work per visited cursor (`internal/model/volume/runtime.go`)
  - `Complete` and `Capacity` results
  - state identities (`crashStateIdentity`, `pendingIdentity`)
- Port `CrashVolumes` (:477), `AdvanceVolumeLifecycle` (:567; handles revoked, fs ESTALE, descriptors cleared before any image is exposed), `AdvanceVolumeLifecycleAfterDivergence` if reachable from wasi, and `SelectCrashOperations(seed)` (:628).
- Recovery image (decision; format change allowed): a versioned binary blob with magic, a version and length-prefixed sections, carrying everything recovery uses:
  - the durable tree
  - per-volume persisted state, history, `nextOperation` and `nextInode`
  - fault occurrence counters
  - the storage-model identity supplied by Go
- Image identity = SHA-256 over the blob. The old identity (SHA-256 of Go `json.Marshal({Snapshots, Occurrences})`) is retired, because it omitted recovery-relevant state.
- Recovery (`NewRecoveredEnvironment` and `loadStorageTree` parity):
  - the model identity in the image must equal the request's, else `invalid` before execution
  - lock-database paths inside an image are refused
  - capacity overflow on reload keeps Go's classification
  - occurrence counters continue from the image
- The Go reference image digest used by the harness becomes this blob's digest computed on the Go side. Add a Go encoder for the same blob, test-only, in the conformance test package, so the harness compares images structurally and by digest.

### Investigation targets
**Required:**
- `tools/gomad/internal/model/volume/volume_generated.go:291-760` (enumerate, crash, lifecycle, select)
- `tools/gomad/internal/model/volume/runtime.go` (`CloneDurable`, Work budget)
- `tools/gomad/internal/backends/wasm/wasi/volume.go:328-392` (CrashStorage, EnumerateStorageCrashStates, NewRecoveredEnvironment)
- `tools/gomad/internal/model/volume/work_test.go`, `tools/gomad/internal/model/volume/volume_test.go` (enumeration goldens)

## Acceptance
- [ ] The full volume conformance corpus passes, including crash, enumeration (with resume and capacity), selection, lifecycle and recovery scripts, with identical identities.
- [ ] Recovery refuses an image whose model identity differs from the request's, an image containing lock-database paths, and a truncated or digest-mismatched image (Rust unit tests and conformance scripts).
- [ ] Recovered runs continue occurrence counting from the image, as Go does (conformance script).


## Done summary
Verified Rust crash enumeration, selection, lifecycle and recovery against the Go reference. Added four raw-image conformance scripts for model mismatch, persisted SQLite locks, rehashed truncation and digest mismatch. Fixed the review-discovered acceptance of multiple persisted paths for one inode before recovery rebuild; file and directory aliases now refuse, while six Go-versus-Rust cross-directory rename cases preserve valid crash/recovery behavior with no sync or either parent synced.

Implementation provenance is merge `cd2cf0c9f644345a0cdf5807dd5d8f4d52a5474d`, original core `b2dcd21f5ac797a41ae166dd739c956575ad0283`, inode guards `62229035045da928e3487f160108a92e7cab757d`, review fixes `629e735ca6ef1777d238b156e52600fe5cabe69c`, conformance coverage `79362d2ece0dd266e5a2db402de97aadb4dc39b7`, failing alias reproduction `e6b912b441f0d48fb8b197171632772f0971e9fb` and alias fix `6476da1cd18f4ac58d202f891a15419f3b2a14f7`. Lifecycle baseline is `e58a342491714c48cda2d05d5613cc5daa093358`; independent review uses historical base `2237e0276f50f3b4b4b244f32a564ec17caca110` to inspect actual implementation.

baseline: green. All three Quick commands passed pre-edit. Final Rust volume checks passed 54 unit tests plus one integration test; the complete 46-script differential corpus, four invalid-image scripts and six rename-recovery cases passed; the focused Go storage suite passed after the production fix in 64.153 s. Existing tests cover all enumeration capacities/cursors, lifecycle revocation, durable cloning and recovery-state continuation. `recover-occurrence-continuation` proves counters 2 and 3 survive repeated recovery. The raw-image scripts failed against the missing driver input before support was added (`.flow/tmp/fn176.3-red.log`); the aliased-image regression failed with `accepted aliased recovery paths` before the production fix (`.flow/tmp/fn176.3-alias-red.log`). Model, lock, truncated and aliased blobs are rehashed so digest rejection cannot mask their validator.

Rust fmt, conformance clippy and native lint passed. Initial lint was inconclusive because the inherited `.bin` executable had the wrong architecture (make exit 2); the unchanged lint policy passed with `.tmp/linuxbin` tools, fixes disabled and the lifecycle baseline. Final logs are `.flow/tmp/fn176.3-rust-alias-post.log`, `.flow/tmp/fn176.3-conformance-final.log`, `.flow/tmp/fn176.3-go-alias-post.log`, `.flow/tmp/fn176.3-clippy-alias-post.log` and `.flow/tmp/fn176.3-lint-final.log`. The rebuilt production helper now has SHA-256 `ccee6cd032ac079851a9f5612589de4b01040e46467c0926845e3e934873075c`; the earlier unchanged-helper observation applies only to the conformance-only commit. The conductor stopped the frozen `500e4d921d` run on discovery of the production defect. Full frozen qualification and whole-spec acceptance remain open. Broader inherited source checks are retained in `.flow/artifacts/fn176-source-gates-2026-10-15.md`.

Defect route:
- prior fixes: local image history contains the earlier cross-volume/reserved-inode guard, with no existing within-volume path guard; memory search found no entry; unchecked: GitHub PRs/issues (HTTP 401 Bad credentials).
- diagnosis: an authenticated Go image with rehashed duplicate file paths was accepted before the fix; the per-volume path map admitted duplicate inode references and rebuild created separate nodes. Go records rename removal/insertion and descendants in one atomic persistence operation; either parent sync selects the whole operation. No volume hard-link API exists. Six differential rename/recovery scripts confirm valid states still recover; existing tests retain cross-volume/reserved-node alias checks. Go-owned FS images originate internally; helper-bound raw blobs use the Rust decoder before execution.
- introduced by: `b2dcd21f5ac797a41ae166dd739c956575ad0283` Prepare pure Rust volume persistence crash and recovery core; bisect skipped because no known-good decoder revision exists.
- base: `e6b912b441f0d48fb8b197171632772f0971e9fb` regression fails with accepted aliased recovery paths; head: `6476da1cd18f4ac58d202f891a15419f3b2a14f7` file/directory alias regression and all affected gates pass.
- live: no interactive app surface; library decoder, conformance process and storage-helper integration tests provide execution proof.

All 18 external fn-180/fn-181/fn-182/fn-183 planning files retain their original hashes and remain outside this task's commits.

Tier: implementer gpt-6.1-sol at medium (role default/project pin)

stage: impl-review - ran [2026-10-15T05:54:22Z..2026-10-15T06:05:14Z] (Codex gpt-6.1-sol at high; first fanout NEEDS_WORK, same-receipt re-review SHIP; reviewer and writer are from the same Codex family; receipt `/tmp/impl-review-receipt-657da2bc4466-fn-176-gomad-durable-volume-crash-model-in-the.3.json`)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: cd2cf0c9f644345a0cdf5807dd5d8f4d52a5474d, b2dcd21f5ac797a41ae166dd739c956575ad0283, 62229035045da928e3487f160108a92e7cab757d, 629e735ca6ef1777d238b156e52600fe5cabe69c, 79362d2ece0dd266e5a2db402de97aadb4dc39b7, e6b912b441f0d48fb8b197171632772f0971e9fb, 6476da1cd18f4ac58d202f891a15419f3b2a14f7
- Tests: baseline: green (all three Quick commands passed pre-edit), TMPDIR=/tmp cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked volume (exit 0; 54 unit tests and 1 integration test), TMPDIR=/tmp make -C tools/gomad/internal/backends/wasm test-conformance (exit 0; 46 volume corpus scripts, 4 invalid raw recovery scripts and 6 cross-directory rename/recovery cases), cd tools/gomad && TMPDIR=/tmp go test -tags test_dep -count=1 -run 'TestVolumeConformance|TestStorage|TestSQLiteStorage' ./internal/backends/wasm/wasi (exit 0 after production fix; 64.153s), TestVolumeConformanceInvalidRecoveryImages: RED missing raw-image conformance control, then GREEN model/lock/truncated/digest; occurrence continuation corpus GREEN, cargo +1.94.1 fmt --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --check (exit 0), cargo +1.94.1 clippy --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked --all-targets --features conformance -- -D warnings (exit 0), make lint-code-fast (inconclusive: inherited .bin golangci-lint Exec format error; make exit 2), make lint-code-fast GOLANGCI_LINT=/Users/stephan/Workspace/skunkworks/gomad/temporal/.tmp/linuxbin/golangci-lint-v2.13.0 ERRORTYPE=/Users/stephan/Workspace/skunkworks/gomad/temporal/.tmp/linuxbin/errortype GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=e58a342491714c48cda2d05d5613cc5daa093358 (exit 0; zero issues), cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked rehashed_images_refuse_multiple_paths (RED exit101 at e6b912b441; GREEN in final 54-test volume run), make -C tools/gomad/internal/backends/wasm helper (exit 0; production alias fix changes helper SHA256 to ccee6cd032ac079851a9f5612589de4b01040e46467c0926845e3e934873075c; frozen500 qualification stopped by conductor), sha256sum --check .flow/tmp/fn176.3-external-before.sha256 (18 external planning files unchanged)
- PRs: