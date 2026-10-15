---
satisfies: [R3]
---
# fn-178-gomad-multi-instance-clusters-in-one.8 In-process incarnation lifecycle: boundaries, revocation, restart holds and recovery-image restart

## Description
Model the incarnation lifecycle in-process (R3): declared crash/stop boundaries, storage crashes, revocation, restart holds and restart from the Rust-held recovery image. The revocation-after-reply race class fixed in fn156.8 must not reappear.

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/wasmhost/src/broker.rs`, `tools/gomad/internal/backends/wasm/wasmhost/src/cluster.rs`, `tools/gomad/internal/backends/wasm/wasmhost/src/protocol.rs` (control frames: restart, live commands), `tools/gomad/internal/backends/wasm/wasmhost/tests/cluster.rs`
**Touches:** [tools/gomad/internal/backends/wasm/wasmhost/src/broker.rs, tools/gomad/internal/backends/wasm/wasmhost/src/cluster/**, tools/gomad/internal/backends/wasm/wasmhost/src/protocol.rs, tools/gomad/internal/backends/wasm/wasmhost/src/engine.rs, tools/gomad/internal/backends/wasm/wasmhost/src/model/**, tools/gomad/internal/backends/wasm/wasmhost/src/storage_transport.rs, tools/gomad/internal/backends/wasm/wasmhost/src/network/cluster.rs, tools/gomad/internal/backends/wasm/wasmhost/tests/cluster.rs]

### Integrated batch amendment (2026-10-15)

Tasks .8–.11 form one coupled lifecycle/client/measurement batch under MILESTONES.md's batch-first strategy. One implementation owner may develop dependent surfaces provisionally against committed fn176 `458c0be2d35ecaf401d8ecd6e176f6be497c5a23` while fn176 qualifies in a separate frozen checkout. This permits development, not acceptance: all original spec dependencies and R1–R8 remain binding before completion. Root owns integration, independent batch review and Flow completion; the worker owns implementation checkpoints in an isolated worktree. Focused checks run during development; integrated qualification runs on the joined candidate, with heavy performance gates alone. No PR, push, CI or native qualification is authorized.

The batch handover maps .8/R3, .9/R1/R3/R7, .10's measured decisions and .11/R4/R5/R8 to exact candidate, commands, exits, regression results and review. Reuse the existing design record and the read-only lifecycle/measurement preflights under `.flow/tmp/fn178-*-preflight-2026-10-15.md`; these are navigation aids, not acceptance evidence. Correct stale planned paths: `src/cluster.rs` is now `src/cluster/mod.rs`, and State lives in `src/engine.rs`. The declared surfaces above admit lifecycle wiring and incarnation-aware transport, not unrelated refactoring. Preserve actual Go semantics (stdout-only cumulative boundaries and Finished releasing restart holds) over stale line references or the fd1/2 wording below. Settle delayed/never-started Open/Start semantics with .9; sealing boot configurations must not start guests early.

### Approach
- Boundaries, validated as in Go (marker ≤ 4096 bytes, one `CrashAfterOutputs` group, members are incarnation 1 with ≥ 2 members):
  - `CrashOnOutput`/`StopOnOutput` evaluated on fd 1/2 output in `finishCall` order (broker.go:306-352)
  - the `CrashAfterOutputs` barrier (broker.go:~560)
  - storage `crash-before` from the instance's Rust volume (fn-176)
- Revoke (broker.go:488):
  - register a restart-hold participant in the clock
  - remove the instance's clock participant
  - revoke its network endpoint (stop is graceful, crash is not)
  - cancel its pending World request
  - set the instance state to revoked with a typed `crash`/`stop` boundary
  - flush its evidence, including storage evidence and image (fn-176.5/.6 semantics)
  - drop its store
- Pending cross-instance deliveries at a crash follow the Go broker's classification. Capture golden cases (graceful stop vs crash, in-flight vs queued, either end of a link) from the Go broker and assert them.
- Revocation-after-reply analogue (invariant): a revocation decided by the effects of an instance's host call takes effect before that instance performs any further effect. In-process, the broker evaluates boundaries before returning from the host call, so the "reply already sent" window does not exist. Port the cases of `TestTransportRevocationAfterReplyInterruptsTheCrossingCall` (wasi/process_test.go:607-664) as in-process tests, plus a crash landing between a durable write and the next call.
- Restart: a control frame `restart{node}` is accepted only when the incarnation is finished. It creates incarnation+1 with slot `old.slot + len(order)` (capacity-checked at `1<<16`), builds a fresh store from the Rust-held recovery image, re-registers with the network and clock, and releases the hold (broker.go:598). Go supplies only the new incarnation's boot config; the image never leaves Rust unless requested.
- Live unbound commands (`Crash`, `Stop`, `Link`) are polled at scheduler handoffs per fn-178.1. They stay non-replayable, as today.
- Slot, incarnation and marker limits keep their `capacity` versus `invalid` classes.

### Investigation targets
**Required:**
- `tools/gomad/internal/backends/wasm/wasi/broker.go:270-692`
- `tools/gomad/internal/backends/wasm/wasi/process.go:516-560,700-760` (modeledReapedCancellation, revokedAfterReply)
- `tools/gomad/internal/backends/wasm/wasi/process_test.go:259,607-664`, `tools/gomad/internal/backends/wasm/wasi/broker_test.go:87-200`
- `tools/gomad/internal/backends/wasm/cluster/provider.go:482-780` (Start/Restart/Crash/Stop)
- Restarted incarnations carry an `incarnation` field in their instance-tagged frames and in `cluster_result.instances` entries (reserved by the fn-178.1 protocol draft).

## Acceptance
- [ ] Instance frames and `cluster_result.instances` entries of restarted incarnations carry `incarnation`.
- [ ] In-process crash, stop, storage-crash and restart fixtures produce the same per-incarnation evidence, boundaries and recovered history as the Go broker reference (recorded cases until fn-178.9's live comparison).
- [ ] Revocation-after-reply analogue tests pass for crash and stop, with and without an intervening durable write, under 12 busy loops.
- [ ] A trap in one instance during another's pending delivery leaves the other's evidence intact and follows the Go classification for the pending delivery.


## Done summary
In-process incarnation lifecycle (R3): incarnation-tagged frames and `cluster_result.instances` entries; declared crash/stop/CrashAfterOutputs boundaries on cumulative stdout with Go precedence; storage crash-before with the fn-176 StorageCrashContext forwarded through sealed cluster boots (10fa86629b: forged context refused like Go); revocation registers restart holds, revokes network endpoints and cancels pending World requests before any further effect; Finished releases holds; restart builds a fresh store from the Rust-held recovery image; live Crash/Stop/Link polled at handoffs; manual controller turns serialize delayed Start/Restart.

Evidence: Go-broker parity cells TestClusterOutputBoundaryParity (stderr/split crash+stop, no durable write) and TestClusterBrokerDurableRestartParity (crash/stop/open-streams/storage-crash x seeds 7/19/42, record+replay) with explicit no-further-effect assertions (2970ccfdd6) passed under 12 busy loops (driver stage busy-boundaries) and in cluster-functional; pending-delivery trap classification vs Go golden; Rust full suite (rust-default/diagnostic test+clippy). Integrated batch .8–.11 (MILESTONES batch-first). Frozen qualified candidate `e75ac7df25fe59934630510dd3bc235f47557c92` (branch `gomad-fn178-batch-20261015`, joined with gomad and fn176-close 3d9c60813f). Fast frozen driver `/home/agent/wt/fn178-tmp/qual/run-qualification-fast.sh` (same 89 stages as the serial driver, asserted) PASSED: root `/Users/stephan/Workspace/skunkworks/gomad-fn178-window5-20261015/qual`, 3226 s, all stages EXIT 0 except 3 keyed REUSED phase-0 passes (lint-router, lint-router-vet from window 1; rust-fmt from window 2). Source digest `sha256:c80a0428…4d53`; authenticated release helper sha256 `5278fab3…0161`. Independent batch impl review (flowctl claude, claude-opus-5-5 high, base gomad): NEEDS_WORK -> SHIP; focused re-reviews of post-SHIP commits: SHIP (27c51cc28a..27822d5e86, P3s fixed in b4aad2b1c7) and NEEDS_WORK -> SHIP (b4aad2b1c7..e75ac7df25). Handover: `.flow/tmp/handovers/fn178-batch-summary.md`. Native execution remains deferred (fn-128/fn-149).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 7c3b767300, 10fa86629b, 2970ccfdd6, f0b44e5a43, e32eb70eea, afe6ada070, 851df2a32f, fdf3ca5825, d18f40916e
- Tests: frozen driver /Users/stephan/Workspace/skunkworks/gomad-fn178-window5-20261015/qual PASS (busy-boundaries, cluster-functional, rust-default/diagnostic-test, rust-*-clippy), go test -tags test_dep -run '^(TestClusterOutputBoundaryParity, TestClusterBrokerDurableRestartParity)$' ./wasi under 12 busy loops (exit 0), cargo +1.94.1 test --release --locked (exit 0)
- PRs:
### Detail (re-appended after flowctl done, 2026-10-15)
- Qualified frozen candidate e75ac7df25; driver stages log `/Users/stephan/Workspace/skunkworks/gomad-fn178-window5-20261015/qual/stages.log`, per-stage `.command`/`.log`, lanes A/B/C 1319/2291/2165 s, total 3226 s. Toolchains: stock Go 1.27.1 (sha256 1675694e…), rustc 1.94.1. Host Linux/aarch64, boot e1418388.
- Storage crash context red/green: `.flow/tmp/fn178-resume2/storage-context-{red,green,divergence}.log` (red: 'cluster frame is not addressed to the waiting instance'; green incl. forged-PendingSHA256 refusal).
- Revocation-after-reply analogues: `/Users/stephan/Workspace/skunkworks/gomad-fn178-window5-20261015/qual/busy-boundaries.log` (12 busy loops, exit 0). Earlier v9 busy-loop and Rust manual-controller evidence remains in the handover.
- Gates: `/Users/stephan/Workspace/skunkworks/gomad-fn178-window5-20261015/qual/cluster-functional.log`, `/Users/stephan/Workspace/skunkworks/gomad-fn178-window5-20261015/qual/rust-default-test.log`, `/Users/stephan/Workspace/skunkworks/gomad-fn178-window5-20261015/qual/rust-diagnostic-test.log`, clippy logs.
