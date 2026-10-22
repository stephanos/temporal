---
satisfies: [R3]
---
# fn-227-parameterized-models-declared-worlds.16 Named intermediates in the IR so admitted trees stay linear in conditional returns

## Description
From fn-227.15's reviews (`.flow/tmp/fn22715-review.md` L1, `fn22715-review2.md` N2). Emitted TLA+ is linear in conditional returns since fn-227.15, but the front end's admitted tree and the macro-embedded `__lifted` data still grow about 7x per return when a later condition or update reads a field changed after a conditional return (and a run of updates after one early return doubles the tree per update). fn-227.15 added a 10,000-node budget so such steps refuse at the statement instead of exhausting rustc memory (measured: about 24k-64k nodes OOM), which caps the pattern at about four sequential returns.

Introduce named intermediates in the IR (`ExprKind::Local` plus per-branch definitions, emitted as TLA+ `LET`) so the tree, the embedded data and compile time stay linear. Then raise or remove the budget. Run before fn-227.3, whose retries add conditional rejections to the activity.

### Verification
Size tests over the admitted tree and embedded data (not only emitted TLA+) for sequential returns and repeated updates; TLC verdicts, state counts and replay unchanged on every fixture; listed TLA+ text goldens may change, justified. Quick then Full gates; record compile time for a pathological probe before/after.


**Size:** L
**Touches:** `model/umpire/ir/**`, `model/umpire/front/**`, `model/umpire/lift/**`, `model/umpire/backends/tlc/**`, `model/umpire/tests/**`, `model/umpire/dsl/macros/**`, `model/README.md`, `model/DESIGN.md`, `model/temporal/feature/activity/expected/**` (only justified emitted TLA+ text), `model/temporal/hosts/go/umpire_temporal_guest.wasm` (generated refresh).

### Execution and gates

Consume fn-238.9's derived constructor printer before adding the new IR fields and variant. Keep generic IR/front-end/lifter ownership; use the existing emitted LET/name-collision machinery where it fits. Preserve first-return semantics, conditional execution, checked arithmetic, payloads, selected World substitution, phase/rejection and terminal-law behavior. An authored name must not capture a generated intermediate. Source/TLC replay must resolve the same typed values. This changes the admitted data representation, not the authored Model semantics.

Before changing expectations, retain baseline semantic/golden hashes and per-fixture TLC counts/verdicts, and measure the existing pathological source probe. Add discriminating linear-growth tests for both sequential returns and repeated updates over admitted tree size and embedded constructor data, plus actual compilation past the old refusal/OOM range. Emitted-text size alone is insufficient. Raise or remove the obsolete node cap only with measured safe compilation. Preserve canonical Model source, native reports/states/rows, verdicts and replay outputs. List each necessary emitted TLA+ text golden change with its mechanism; do not globally refresh unrelated goldens. A needed path outside Touches must be reported to the conductor before editing.

Quick from model/: `mise exec -- cargo test -p umpire-front -p umpire-ir`; `mise exec -- cargo test -p umpire-lift --test phased`; relevant source/replay controls under required TLC. Include the new growth/collision/compilation controls in focused runs. Full from root: one `UMPIRE_BACKENDS=require make model-time`, whose nested `make model-check` must explicitly show required TLC execution and all stages passing. This satisfies the complete full gate and timing together; preserve the actual command. Use an outer bound of at least 1800s for this full/timing run, and at least 1300s for the gap target whose Go test has a 20m bound. Do not repeat a completed unchanged full gate merely for a second command spelling. If changed closure bytes require the embedded guest refresh, generate through `make umpire-guest` in a separate commit once source settles, then actual `make umpire-check-model-slice` and `make umpire-check-model-gaps`; don't fit expected semantics to the server. Preserve literal timing output, probe before/after, TLC counts/verdicts/replays and golden comparisons in task-owned artifacts. Record soft-target overruns honestly.
## Acceptance
- [ ] Admitted tree and embedded data grow linearly with sequential conditional returns and repeated updates (size tests).
- [ ] Node budget raised or removed with a probe showing no rustc blow-up.
- [ ] TLC verdicts, state counts and replay unchanged; changed goldens justified.

## Done summary
Named intermediates now keep admitted IR, embedded constructor data and emitted TLA+ linear across sequential conditional returns and repeated updates. The obsolete node cap is removed after an actual 32-return/64-update macro compilation, and vector admission follows dependency-ordered facts without expanding shared definitions.

Tier: session (jev-unavailable(no_key)); repository implementer override gpt-6.1-sol at medium.
stage: impl-review - ran (first three-axis round NEEDS_WORK; subsequent single-dispatch rounds 2–4 SHIP at historical source pins; final round 5 SHIP at 28306365be, codex:gpt-6.1-sol:high, fresh review context; same model family, not cross-family independence).

Baseline: green focused phased Quick before edits. The first front/IR observation lost its exit handle and remains inconclusive. The redundant pre-edit full gate was cancelled at the conductor's direction, with a recent required-TLC full pass at 11436f2f and only Flow/MILESTONES changes through task base. All failed, superseded and cancelled observations remain retained and are not green evidence.

R3 controls: admitted node counts for 8/16/32 sequential returns are 266/522/1034; repeated updates 138/234/426; vector updates 129/209/369. Constructor data sizes are recorded in task16-phased-final2.log. The old eight-return probe safely refused at 10,000 nodes (test compile 1.34s); the new, different 32-return/64-update compiled probe completed in 0.82s (wall 0.905s, child RSS 215456KiB). These are different probes and boundaries; no speedup ratio is inferred. Authored/generated-name collision controls include actual TLC state agreement.

Vector validation's repeated Boolean DAG regression was reproduced, then fixed with dependency-ordered facts. Review/audit exposed scalar and sequence identity compatibility cases; red-to-green controls cover saved/chained/conditional lengths, unrelated/nested/other-field conditionals, Boolean guards and finite limits, with stale/cross-field/mixed/unknown and unbounded refusals. Pinned d706 source reconstruction establishes pre-task acceptance; cached-library observations are explicitly superseded. Required TLC compares all admitted variants to the same native three-state set and Verified/Refuted verdict pair. Conditions and checked arithmetic continue to execute.

Verification: UMPIRE_BACKENDS=require make model-time passed with exit0, all17 stages including explicit required TLC. Gate611.465s exceeds the60s soft target; inner7.880s meets10s. Conductor observed unrelated gomad Rust release build overlap, so this is closure validation rather than idle/isolated/steady-state performance. Required focused front/IR/lift tests and clippy pass. All 782 baseline golden/refusal files are compared:776 unchanged, six TLA+ payloads changed only to introduce ordered LET definitions (Inputs, ChangedState, Counter, Job, Kettle, Gate). task16-tla-changes.txt explains each; native reports/states/rows/verdicts/replays, snapshot metadata and refusals are exact. The default activity Model is unchanged; DESIGN remains 893 lines.

Pinned guest refresh checkpoints are retained honestly, including corrections after review/audit. Final bytes equal the baseline SHA256 d544b0b2d8c835034fc0e070e875ad3b90fbf512e9132a79196c64fa7788652d. Actual make umpire-check-model-slice (six controls) and make umpire-check-model-gaps (369.070s) passed against those exact bytes. Conductor-authorized artifact-equivalence reuse applies because Go/host/server source and tests are unchanged. Full raw slice/gap/fixture logs are preserved as verified gzip with hashes and bounded summaries. The derived Vec printer uses vec![..] after a real large_stack_arrays lint; no lint was suppressed. fn238.9 token equivalence remains historical at its own pin.

The first review finding was fixed and captured in memory. Root's periodic correctness and standards audits each found one issue, both resolved and rechecked before the final full gate; root owns their durable aggregation, remaining tasks, parent completion and milestones. No attributable command remains running at handoff.

Durable evidence: .flow/artifacts/fn-227-parameterized-models-declared-worlds/task16-*; final review receipt task16-review-final-receipt.json, final gate task16-accepted-gate-summary.json, golden comparison task16-all-golden-comparison.json, guest equivalence task16-guest-equivalence.json.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 927461a80f1b47a734b3de14b34ab3adef0d1960, 5c26527df4554134d290c8f10e1207be7425fca2, 5e568d1584f9c36905bb57e945e3ad794868555f, 750f1211c824c743f30f27c09375afab8a3415d3, 54cced51cbf038c60e58721bb51cf090a817e1b9, db7f65db5c3f8219aef574c806350dbec7dbeb5c, 3e34b8010e64de36cb9eafdd87c72af92ea29ee1, c06f43e0218aa7e70a1a8762be5d6c3e8e6a3d00, d395724b04a2fa3480ceed17ca98515b403cbee4, 87a5300ba48547eb8ad16d1da45439ca5816d18a, 79fe477d3f690a6ace3f8f0c43ef940a9e6476b2, dd4897437d975fb5cd52aa1470c0fe71c8fac4c2, c632d95e9df9239917dfb0e56a2f42d84373a74f, 36e5611d559e511a23a3b6f47372627044605c63, 4e7e25cbecbd4da3d593e7f1612bab50adbbf5f8, 28306365be80e87fecb2328e831fda559c536bc5, d32057228c537b4d1353857c54a80e44326733f6, 0a87cde3dfd871fd4c435cbd094a31da56eaa6a9
- Tests: cd model && UMPIRE_BACKENDS=require mise exec -- cargo test -p umpire-front -p umpire-ir -p umpire-lift, cd model && mise exec -- cargo clippy -p umpire-front -p umpire-ir -p umpire-lift --all-targets -- -D warnings, make umpire-guest, make umpire-check-model-slice, make umpire-check-model-gaps, UMPIRE_BACKENDS=require make model-time
- PRs: