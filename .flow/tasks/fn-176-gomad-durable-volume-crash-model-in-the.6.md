---
satisfies: [R1, R4]
---
# fn-176-gomad-durable-volume-crash-model-in-the.6 Broker split ownership: helper-owned namespace and volume under the Go broker

## Description
Make Broker runs use the Rust namespace and volume while scheduling, runtime decisions and the shared clock stay with the Go Broker (R1, R4). The committed-workflow crash/reopen gate and the SQLite durable cells run under a Broker (`CrashOnOutput`, `Restart`), and the Broker disables the helper local model (process.go:324), so the Rust volume cannot reach those gates without this split mode. fn-178 later moves the Broker in-process; this mode stays as its Go reference path.

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/wasi/process.go`, `tools/gomad/internal/backends/wasm/wasi/helper_model.go`, `tools/gomad/internal/backends/wasm/wasi/broker_process.go`, `tools/gomad/internal/backends/wasm/wasmhost/src/model.rs`, `tools/gomad/internal/backends/wasm/wasmhost/src/engine.rs`, `tools/gomad/internal/backends/wasm/wasmhost/src/protocol.rs`, `tools/gomad/internal/backends/wasm/wasmhost/protocol.md`, `tools/gomad/internal/backends/wasm/wasi/process_test.go`, `tools/gomad/internal/backends/wasm/wasi/broker_test.go`
**Touches:** [tools/gomad/internal/backends/wasm/wasi/process.go, tools/gomad/internal/backends/wasm/wasi/helper_model.go, tools/gomad/internal/backends/wasm/wasi/broker_process.go, tools/gomad/internal/backends/wasm/wasi/broker.go, tools/gomad/internal/backends/wasm/wasmhost/src/model.rs, tools/gomad/internal/backends/wasm/wasmhost/src/engine.rs, tools/gomad/internal/backends/wasm/wasmhost/src/protocol.rs, tools/gomad/internal/backends/wasm/wasmhost/protocol.md, tools/gomad/internal/backends/wasm/wasi/process_test.go, tools/gomad/internal/backends/wasm/wasi/broker_test.go, tools/gomad/internal/backends/wasm/wasmhost/tests/engine.rs, tools/gomad/internal/backends/wasm/wasi/integration_test.go, tools/gomad/internal/backends/wasm/cluster/provider.go, tools/gomad/internal/backends/wasm/cluster/provider_test.go, tools/gomad/internal/backends/wasm/wasi/identity_test.go]

### Approach
- New ownership mode (`filesystem-owned`, decision):
  - The helper owns namespace, volume and the WASI transcript.
  - `runtime_*`, `clock_time_get`, `sched_yield` and `poll_oneoff` clock parts stay remote to Go, and the Broker keeps the choice/runtime session and clock.
  - Remote replies carry `local_state` (`now`, `advances`) and canonical transcript bytes exactly as helper-owned runs do (protocol.md). The volume clock therefore follows Broker time.
- fd 1/2 writes stay remote in Broker runs, so `CrashOnOutput`, `StopOnOutput` and `CrashAfterOutputs` (broker.go:306-352, ~560) still observe output in order. Everything else in the namespace is local.
- Revocation-after-reply invariant: the durable effects at a Broker crash/stop must equal the Go-owned path's.
  - Today a revocation detected in `Broker.Delivered` after the reply interrupts the next call (process.go:534 `revokedAfterReply`, 703-742). With local storage, the guest could perform local durable effects before its next remote call.
  - Approach: Go marks a reply `revocable` when Delivered has not yet been evaluated. Before the guest's next local effect (any namespace/volume mutation or read), the helper sends a lightweight `sync` frame and waits for `continue` or the typed `interrupt`.
  - Only remote replies trigger this, so the steady state stays callback-free.
- On a Broker `crash`/`stop` interrupt, the helper flushes storage evidence and the image as fn-176.5 does for `storage_crash`. A pending storage crash and a Broker revocation are ordered by which one the guest reaches first; add a test for each order.
- Call accounting: local calls consume the global call IDs in the helper. Broker `Limits.Calls` and `nextCall` adopt helper state from `local_state` on remote frames.
- Parity: run Broker fixtures with the Go-owned path (`remoteModel` extended to Broker runs) and the split mode, and require identical transcripts, storage evidence and images:
  - `TestBrokerCrashRecoveryFreshGuest`
  - `TestBrokerCrashRetainsInterruptedRuntimePrefix`
  - a CrashOnOutput SQLite fixture
- Extend `TestTransportRevocationAfterReplyInterruptsTheCrossingCall` (process_test.go:607) to the split mode, including a local storage write between reply and next remote call.

### Investigation targets
**Required:**
- `tools/gomad/internal/backends/wasm/wasi/process.go:300-560,600-808`
- `tools/gomad/internal/backends/wasm/wasi/broker_process.go`, `tools/gomad/internal/backends/wasm/wasi/broker.go:239-360,488-640`
- `tools/gomad/internal/backends/wasm/wasi/process_test.go:259,607-664`
- `tools/gomad/internal/backends/wasm/wasi/broker_test.go:87-200`
## Acceptance
- [ ] Broker storage runs answer every non-fd-1/2 namespace and volume op in the helper (Go fs-served counter equals the number of fd 1/2 writes).
- [ ] Go-owned vs split-mode parity passes on the Broker fixtures above, including images and recovered history.
- [ ] Revocation-after-reply tests pass in split mode for crash and stop, with and without an intervening local storage effect, under 12 busy loops (as fn-156.8 measured).
- [ ] protocol.md documents the mode, the `revocable` reply flag and the `sync`/`continue` frames.


## Done summary
Qualified Broker split ownership and fixed two review-discovered revocation defects. `sync` now checks the current session and clock under the broker lock, and canceled Crash/Stop actions cancel the helper during the revocation grace and return the context error while execution retains cleanup ownership. Real-helper controls cover crash and stop with a local volume write/sync or a direct next output; broker parity fixtures require exact output-only Go filesystem callback counts.

Acceptance and command evidence is retained in `.flow/artifacts/fn176-task6-acceptance-2026-10-15.md`. The implementation originated in merge cd2cf0c9f6; this task preserves the coordinated offered-output correction 2668af049e and its descriptor-identity routing, cumulative local/remote budget and terminal-bound rules. The historical implementation review base is 2237e0276f50f3b4b4b244f32a564ec17caca110. The lifecycle base is 41efa72b189ec43c4e901240ef4f47e1756c3172.

baseline: green. The three parent Quick commands passed before edits. Fresh broker parity passed for clock read steps 0 and 1000, including transcripts, choices, storage evidence/image and recovered history. SQLite crash output and both storage-crash/revocation orders passed; NetworkV2 storage restart passed for crash and stop. The real-helper control verifies unchanged storage log, structured evidence/history and image across the reply crossing; successful continue controls prove the local write/sync executes when allowed.

Fresh stress ran the three Go transport/split revocation tests 50 times each and the real-helper matrix 50 times under twelve busy loops. It exited zero and all twelve exact spawned PIDs were reaped. `.flow/tmp/fn176.6-stress.log` retains the observations. Conductor-approved Touches additions for wasmhost/tests/engine.rs and wasi/integration_test.go were persisted through flowctl before commit.

Final source gates passed. Rust ran 196 unit and 151 integration tests with two existing ignored tests; canonical volume-filter and volume conformance gates passed. The expanded Go gate after the fixes passed in 105.571 seconds, actual cluster cleanup/restart controls in 41.162 seconds and backend identity/storage/evidence consumers in 6.578 seconds. Configured native lint reported zero issues. Rust sources remained unchanged by the Go fixes, preserving the earlier cargo fmt, conformance clippy and full Rust pass. Generated validation passed before those fixes; its commands and the final source logs are retained in the acceptance artifact.

Both stock compiler environment variables used tools/gomad/.toolchain/stock/878c9b2935e2c9e6/go/bin/go with TMPDIR=/tmp. Initial shell wrappers that inherited the forced wrong working directory were inconclusive and corrected before their commands executed. The new helper fixture initially included an empty interrupt record without chunks; protocol refusal exposed the fixture error, corrected by omitting that absent optional record. Helper SHA256 remains 7ccd46787a123e3e7d7b108d31d778a4223b258de4f606fe9c66fcdb935bdb64. The deliberate Go source fixes rotate its model identity to sha256:33a4d4ff606754aec5d82664d2c5360c34ae1a7a43e420e314241d02e037f2db, with the exact static pin updated and tested.

The first review merged two findings, deduplicating correctness/contracts/integration reports about live revocation state and correctness/integration reports about action cancellation. Failing reproductions are committed at 7e75cccd3e; fixes are cfa09430a7. The final committed timing/cleanup controls each passed 50 repetitions under twelve busy loops in `.flow/tmp/fn176.6-final-timing-stress.log`, including the explicit Reaped transition and mandatory Shutdown. All exact load PIDs were reaped.

Defect route:
- prior fixes: memory and local branch/history searches found no existing correction for either interval; unchecked: GitHub PRs/issues (HTTP 401 Bad credentials); tracker inactive.
- diagnosis: after-successful-Delivered crash/stop reproductions observe `continue` at the pending sync; a blocked-node canceled action exceeds the one-second control bound and never calls helper cancel. Reading current session state at sync and observing operation cancellation eliminate those failures.
- introduced by: cached sync check at 0ed1631451 and context-blind grace at 629e735ca6; bisect skipped because there is no known-good split implementation.
- base: 7e75cccd3e fails both regressions in `.flow/tmp/fn176.6-review-red.log`; head: cfa09430a7 passes focused, actual helper, consumer and stress controls.
- live: no interactive application surface; real helper parity/cleanup/restart and deterministic transport/blocked-node tests provide execution proof.

The accepted local clock/yield/poll deviation and split ownership for NetworkV2 remain recorded in the parent spec. Final frozen qualification, Temporal committed-workflow gates, workload success/canary cells and all four WASM SQLite selected states/catalogue remain with task 7 and the conductor. Native execution stays deferred under fn-128/fn-149. All 18 external fn-180/fn-181/fn-182/fn-183 planning files retain their initial hashes and remain unstaged.

Tier: session (jev-unavailable(no_key))

GATE_SKIPPED:unittest:green-receipt cfa09430 - expanded task gate pass reused after review
GATE_SKIPPED:smoke:green-receipt cfa09430 - volume conformance pass reused after review
GATE_SKIPPED:rust-volume:green-receipt cfa09430 - canonical volume filter pass reused after review

stage: impl-review - ran [2026-10-15T07:23:31Z..2026-10-15T07:37:32Z] (three-axis NEEDS_WORK, same-receipt re-review SHIP; historical base 2237e0276f50f3b4b4b244f32a564ec17caca110 through cfa09430a7; Codex gpt-6.1-sol at high; fresh first-round reviewer contexts, same GPT/Codex family as writer; receipt /tmp/impl-review-receipt-657da2bc4466-fn-176-gomad-durable-volume-crash-model-in-the.6.json)

stage: memory-capture - ran (bug/integration/check-live-broker-revocation-at-local-2026-10-15; bug/runtime-errors/observe-action-cancellation-during-2026-10-15)

stage: plan-sync - skipped(config: planSync.enabled != true)

Tracker sync: n/a (bridge inactive)
## Evidence
- Commits: cbd7ae3b285976ee242efc16a28bb09b9ff1f50c, 7e75cccd3e0d9cce6ae6b69a3ad01ed0ebfdfc4e, cfa09430a701eeb5e7da62e93fd280be4f262c40
- Tests: cd tools/gomad && go test -tags test_dep -count=1 -run 'TestVolumeConformance|TestStorage|TestSQLiteStorage|TestBroker|TestSplit|TestTransport.*(Revocation|Split)|TestHelperBridgeSplit|TestImplementationIdentity' ./internal/backends/wasm/wasi, cd tools/gomad && go test -tags test_dep -count=1 -run '^(TestProviderCanceledSplitRevocationCancelsAndReaps|TestProviderMandatoryCleanupReapsWithoutLifecycleActions|TestProviderUnboundRevocationOfSplitStorageNodeRestarts|TestProviderActualCrashDurableRestart)$' ./internal/backends/wasm/cluster, cd tools/gomad && go test -tags test_dep -count=1 -run 'Identity|Storage|Evidence' ./internal/backends/wasm/backend, cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked, cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked volume, make -C tools/gomad/internal/backends/wasm test-conformance, timeout 600 bash /Users/stephan/Workspace/skunkworks/gomad/temporal/.flow/tmp/fn176.6-stress.sh, cargo +1.94.1 fmt --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --check, cargo +1.94.1 clippy --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked --features conformance --all-targets -- -D warnings, make lint-code-fast GOLANGCI_LINT=/Users/stephan/Workspace/skunkworks/gomad/temporal/.tmp/linuxbin/golangci-lint-v2.13.0 ERRORTYPE=/Users/stephan/Workspace/skunkworks/gomad/temporal/.tmp/linuxbin/errortype GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=41efa72b189ec43c4e901240ef4f47e1756c3172, make -C tools/gomad validate, GATE_SKIPPED:unittest:green-receipt cfa09430 - expanded task gate pass reused after review, GATE_SKIPPED:smoke:green-receipt cfa09430 - volume conformance pass reused after review, GATE_SKIPPED:rust-volume:green-receipt cfa09430 - canonical volume filter pass reused after review
- PRs: