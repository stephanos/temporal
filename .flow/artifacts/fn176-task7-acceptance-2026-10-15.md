# fn-176.7 source and durable qualification evidence

The lifecycle base is `363941f7cd1566029fd6f04b04ec68ed63301625` on `gomad`.
The inherited task 7 implementation comes from merge `cd2cf0c9f6`; the historical
implementation-review base is `2237e0276f50f3b4b4b244f32a564ec17caca110`.
Frozen qualification and implementation review were pending at this source checkpoint.
The completed current frozen run is now recorded in
`fn176-frozen-qualification-2026-10-15.md`; implementation review remains separate.

## Refused-run parity correction

Actual-helper comparisons reproduce the cross-task advisory: Go retains the
storage transcript, snapshots and operations when final image creation exceeds
its independent bound or post-run enumeration rejects zero limits; the helper
discarded them. The first image fixture hit transcript capacity instead, so that
observation is inconclusive for image overflow. The corrected fixture raises
only its transcript allowance to 8 MiB. Failing reproduction checkpoints are
`4ac1cd7eb9` and `e21d275a9a`; the corrected red log is
`.flow/tmp/fn176.7-parity-red-corrected.log`.

`TestStorageEvidenceRefusedRunOwnerParity` checks the exact classified terminal
before comparing evidence. Seventeen 64-KiB overwrites under a 64-KiB filesystem
bound yield `capacity` / `WASI capacity: storage-image-bytes` for both owners;
the collected storage log is 3,627 bytes. A single write with zero enumeration
limits yields `invalid` / `crash enumeration limits must be nonzero`, with a
211-byte storage log. Both owners now retain identical nonempty snapshots and
operations, equal enumeration prefixes and no image. Go serves zero filesystem
callbacks for the helper-owned executions.

Rust always queues the already-collected prefix, then applies the unchanged
collection failure terminal. Successful collection still emits complete v1
envelopes. Explicit partial v2 envelopes add one failed-stage byte, omit image
identity/blob and require either an exact configured-order enumeration prefix
with an invalid terminal, or all requested enumerations with the documented
image failure terminal. Go validates the volume set, log binding, every section,
version/count and digest before publishing the prefix.

`TestStorageEvidencePartialCollectionAdmission` covers positive enumeration,
capacity-image and infrastructure-image failures, success/wrong terminals,
nonempty image identity or bytes, missing/complete/non-prefix enumerations,
unknown stages/versions, invalid count/stage length, truncation, corruption and
trailing bytes. Existing complete-v1 tests still refuse missing requested images
and inconsistent identities; independent state/image limits are unchanged.
`.flow/tmp/fn176.7-parity-green.log` retains the first red-to-green observations;
the final expanded Go gate includes the strengthened terminal and version controls.

The separate verbatim correctness and standards audit reports, with conductor
dispositions, are retained in
`fn176-task7-cross-task-quality-2026-10-15.md`. No broad file or vector split is
included. Memory/history found no prior correction; GitHub PR checks are
unavailable (HTTP 401), and tracker sync is inactive. There is no known-good
helper implementation for bisect. This library has no interactive application
surface; the actual helper provides the execution proof.

## Source verification

Baseline: green. Rust volume and conformance reused green receipts `cfa09430`.
The canonical Go Quick selection passed before edits in 89.563 seconds. A first
shell wrapper inherited the wrong directory and was inconclusive; its command
did not execute, and the corrected absolute-directory invocation passed.

All commands below exited zero on the corrected source. Stock compiler variables
`GOMAD3_STOCK_GO` and `GOMAD_WASM_STOCK_GO` point to
`tools/gomad/.toolchain/stock/878c9b2935e2c9e6/go/bin/go`; `TMPDIR=/tmp`.

| Command | Observation | Log under `.flow/tmp/` |
| --- | --- | --- |
| `cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked` | 196 unit + 151 integration; 2 existing ignored | `fn176.7-rust-full.log` |
| `make -C tools/gomad/internal/backends/wasm test-conformance` | volume differential corpus passed | `fn176.7-conformance.log` |
| `cd tools/gomad && go test -tags test_dep -count=1 -run 'TestVolumeConformance\|TestStorage\|TestSQLiteStorage\|TestImplementationIdentity\|TestBrokerCrashRecoveryFreshGuest\|TestTransportSplit\|TestHelperBridgeSplit' ./internal/backends/wasm/wasi` | 88.539 seconds; storage, split, strict evidence and identity controls passed | `fn176.7-go-final.log` |
| `cd tools/gomad && go test -tags test_dep -count=1 -run 'TestWASMArchitecture\|TestArchitecture\|TestStorage\|Test.*Storage\|Test.*Evidence\|Test.*Identity' ./internal/backends/wasm ./internal/backends/wasm/backend ./internal/backends/wasm/cluster ./internal/archcheck` | all four packages passed, including actual storage consumers and dead crash-after boundary test | `fn176.7-consumers.log` |
| `make lint-code-fast GOLANGCI_LINT_BASE_REV=a1f0be7e96e2746cb9c51598766a5ae96d8552c2 GOLANGCI_LINT_FIX=false GOLANGCI_LINT=/Users/stephan/Workspace/skunkworks/gomad/temporal/.tmp/linuxbin/golangci-lint-v2.13.0 ERRORTYPE=/Users/stephan/Workspace/skunkworks/gomad/temporal/.tmp/linuxbin/errortype` | zero issues; initial two De Morgan style findings corrected | `fn176.7-lint-final.log` |
| `cargo +1.94.1 clippy --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked --all-targets --features conformance -- -D warnings` | passed | `fn176.7-clippy.log` |
| `make -C tools/gomad validate` | generated source, compiler tests, pins, scripts, compatibility and qualification manifest passed | `fn176.7-validate.log` |

Cargo fmt, git diff checks and targeted fn176 Flow validation passed. A global
Flow validation separately reports an existing closed fn151 with task .5 still
in_progress; this task does not alter that unrelated lifecycle state. All 18
external fn180/fn181/fn182/fn183 planning files match
`.flow/tmp/fn176.6-external-before.sha256` and remain unstaged.

Later conductor reconciliation completed the historical fn151.5 receipt through
normal Flow lifecycle operations, committed only its tracking file at
`3aeea7936d`, and verified global Flow validation as Valid True. Historical
unreachable-evidence warnings remain. This supersedes the earlier global-status
observation without changing the frozen source or any native qualification claim.

Current release helper SHA256:
`42ef0937b29b88fc67274167e8b0a2388d89f7058bbd8747f9c26ed01a0e5a95`.
Deliberately rotated and tested Go model identity:
`sha256:37a249b97a6fdd3c716032769fa0836a64957cea585633e071be92d9ae46efe4`.

## Frozen acceptance ownership

The conductor owns the one-shot driver, frozen worktree, execution and retained
producer evidence. The fresh run at source `bfb53b2c45` completed exit 0 at
09:02:41Z, with all 84 stage exits zero. Temporal crash/reopen, newly produced
retained replays, all ten primary workload cells and aggregate, four actual WASM
SQLite selected states, authenticated catalogue and final preservation passed.
Five primary successes qualify; five canaries retain expected unqualified
outcomes. The detailed frozen checkpoint binds exact commands, host, compiler,
source and admitted helper. Task review and verified Flow completion remain
separate gates. Actual native execution remains deferred under fn128/fn149.

Tier: session (jev-unavailable(no_key))
