---
satisfies: [R1, R4]
---
# fn-185-gomad-wasm-host-side-execution-cost.1 Attribute host imports to exact ops and model owners in the diagnostic profile

## Description
Extend the diagnostic-profile feature in wasmhost: record each host import's exact module.op and the model owner that served it (namespace, volume, network, clock, World, storage transport, cache, scheduler handoff, transcript/evidence) with count, total wall, exclusive wall (nested encoding/hash/remote-write spans subtracted) and a bounded per-call distribution (min, median, p99, max; fixed-size histogram, no allocation in the hot path beyond what the feature already does). Add the reconciliation row (attributed host wall + guest residual versus execute wall) and keep the overlap error counter. Emit `abi_ops` and `owners` in the profile JSON while keeping phases/abi/nested buckets unchanged. Implement the collector on the Rust linked-execution path, using the fn-187 Go internal/measure and cmd/wasmmeasure collectors as semantic references. Read the new fields and fail on a reconciliation miss or any unattributed import; do not add new Go-host collection work. Prove the release helper is unchanged: empty-features build, the moved runtime-cost producer (formerly `TestHelperRuntimeCostProfile`) on it, no Cargo.lock change. Early proof: one single-guest activity run on seed 0 whose tables reconcile within tolerance with the same choice witness as the release run. Follow completed fn-187 and its final helper module ownership; fn-180 must already be integrated. Preserve the corpus contracts while extending the current dispatch modules.

Pending R4 receiving cross-reference: fn-232 R12/task .13's final frozen Rust window receives the actual runtime-cost-equivalent release-behavior and qualified choice/WASI/network/stdout/stderr preservation comparisons, with the original assertions, bounds and actual identity bindings. R4 stays OPEN in this task until that receiving proof exists and is reconciled here; older f172 raw early proofs remain historical and do not close fn-185.1 for the current candidate. This coordination authorizes no new Go-only profiling/full qualification, migrated or relabeled digests, acceptance waiver or status change.

**Size:** M
**Files:** tools/gomad/internal/backends/wasm/wasmhost/src/** (diagnostic owner and dispatch modules after fn-187), tools/gomad/rust/gomad/src/**, tools/gomad/docs/wasm/README.md

**Touches:** tools/gomad/internal/backends/wasm/wasmhost/src/** (diagnostic owner and dispatch modules after fn-187), tools/gomad/rust/gomad/src/**, tools/gomad/docs/wasm/README.md

## Acceptance
Implement request-local diagnostic session and execute spans on the Rust linked execution path, with bounded sinks and current Rust collector/report owners. Go transport/wasmmeasure paths are semantic references only. Prove the linked diagnostic producer emits the required attribution, without losing the existing pending fn-232.13 receiving obligation for R4. **Touches additionally:** tools/gomad/rust/gomad/src/** and helper linked execution/diagnostic modules.

- [x] Every host import appears under an exact module.op and an owner; unattributed imports fail the collector
- [x] Reconciliation within the stated tolerance and zero overlap errors on the proof run
- [x] Choice witness of the profiled run equals the release run at the same seed
- [ ] Release helper behavior unchanged: moved runtime-cost producer passes (formerly TestHelperRuntimeCostProfile), no Cargo.lock change

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

## Handover 2026-10-23 (worker B, branch fn185-attribution-v2)

No source change was needed: the linked attribution source (request-local capture, Execute/Instantiate spans, bounded sinks, `profile::validate` collector, `gomad-profile`, measurement `host-owner-profile` producer) is already on `gomad` at 2fb6e7ceb4. The earlier branch `fn185-rust-attribution` (42a88eb607 received as a09fa72491; de94aaa91b and cac5b294eb are already present as `Sink::retained_bytes` without the 1 GiB cap and `Conditions::DiagnosticProfile`) has nothing left to port. Its uncommitted refactor only moves code around and is superseded by the measurement producer.

Builds: rust 1.99.1, `--release --locked`, isolated target `/home/agent/.cache/codex-build/fn185-v2-target`, `nice -n 19`, 2 jobs.
- Release (empty features) `gomad-measure` built, exit 0, sha256 `1e1910d7...e708f`. Diagnostic (`helper-diagnostic-profile`) `gomad-measure` built, exit 0, sha256 `0a1ffbb7...c219`.
- `cargo test -p gomad --features helper-diagnostic-profile --test profile_corpus` exit 0 (1/1). Its mutations include unattributed, overlap and reconciliation, and the collector refuses all three. `--lib profile::` exit 0 (2/2). `cargo test -p gomad-wasmhost --features diagnostic-profile --lib diagnostic::` exit 0 (11/11).
- `tools/gomad/rust/Cargo.lock` unchanged against `gomad`.

Early proof (`gomad-measure host-owner-profile`, activity `^TestActivityTestSuite$`, seed 0, reference=release helper, warmup+rep0=diagnostic helper, fresh preparations): exit 0, report `accepted: true`. Rerun: `activity-seed0-rerun.json`. Raw root: `/Users/stephan/Workspace/skunkworks/gomad/.build-cache/fn185-v2/host-owner-activity-01`.
- Both diagnostic runs: `unattributed_imports`, `unframed_imports`, `abi_overlap_errors`, `span_errors`, `threads_omitted` and `report_rows_dropped` are all 0. 32 exact ops over 12 owners. Reconciliation difference is 0 ns against a tolerance of 18.2 ms (rep0: execute 3.649 s = attributed host 1.140 s + guest residual 2.509 s). The collector validated each profile on read.
- Witness: the choice stream sha256 `5db87066...f78d`, transcript `39f31382...`, stdout/stderr/network, fuel_remaining 1125879823799633 and module `4c913e06...7477` are identical across reference, warmup and rep0. The producer's role-variant comparison passed. Guest PASS, exit 0.
- Host was not quiet (worker A builds); timings are not R3 measurements.

Release helper: `gomad-measure runtime-cost-profile` on the release helper (reference/candidate seed 7) exit 0, `accepted: true` (`runtime-cost-profile.json`, `runtime-cost-rerun.json`). This is local evidence only. R4 stays OPEN pending fn-232.13's receiving proof on the frozen candidate.

Retained: `.flow/artifacts/fn185-attribution-v2/`. Kept: worktree `/home/agent/wt/fn185-attribution-v2` (until merge), raw root above plus `.build-cache/fn185-v2/{bin,stock-go1.27.1,toolchain,gocache}` (rerun inputs, about 3 GiB on the host volume), Cargo target `/home/agent/.cache/codex-build/fn185-v2-target`. The old worktree `/home/agent/wt/fn185-rust-attribution` holds uncommitted superseded edits and validation files. It is left for the conductor to remove.
