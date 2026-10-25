---
satisfies: [R6, R9]
---
# fn-252-quick-consolidation-wins-in-model-code.6 Activity TLC replay through umpire_testing::replay_trace

## Description
Implement **Activity TLC replay (B9)**: expose `umpire_testing::replay_trace` (or fn-250's replay core if landed) and remove `proof_checks.rs` `state_value`/`matches_state`/`replay_tlc`, comparing on `Values`.

## Acceptance
- [ ] The activity proof checks replay through the shared function.
- [ ] Re-accept goldens after reviewing their diffs; `make model-check` passes.


## Done summary
Completed locally after conductor acceptance.

Activity proof checks now call `umpire_testing::replay_trace`, sharing the existing typed native/TLC traversal and retaining exact native edge projections in `Divergence` for the global-budget mutants. Removed the activity `state_value`, `matches_state` and `replay_tlc`; the conductor-requested fn-250 R3 follow-up also centralizes every observer key in activity proof, policy and rejection trace checks without changing the authored Model.

Task: fn-252-quick-consolidation-wins-in-model-code.6
Status: in_progress (parallel lane; conductor owns integration, review and completion)
Tier: explicit medium implementer routing (conductor dispatch)
Workspace: /Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn2526-activity-replay
Range: 4d07b017f3d66112d958270765f57708e2c42b7a..e0d4262694215beaa26a8b7d48e56e062b04e2b5

baseline: green — scoped testing library + Temporal fixtures, 45 passed / 2 existing ignored; full `make model-check` explicitly reserved for conductor join.

Verification (all commands run from `model/`, except Git):
- `mise exec -- cargo test -p umpire-testing -p umpire-temporal-tests --all-features`: exit 0, 72 passed / 2 existing ignored. Includes native-only unsupported refusal before producer execution, typed hook selection, activity goldens and backend replay controls.
- `UMPIRE_BACKENDS=require mise exec -- cargo test -p umpire-temporal-tests --test fixtures activity_tests -- --nocapture`: exit 0, 33 passed, 403.16s. Covers all three changed activity test files, complete native/interpreted/TLC agreement in thirteen selected Worlds, fairness/starvation lassos, retry/rejection payloads, reset/broken-expiry/missing-tick source mutants, exact canceled-attempt facts and source-only fact mutations.
- Earlier required `proof_checks` selection: exit 0, 7 passed, 90.03s; the final 33-test selection supersedes this for the observer-name cleanup and new regression test.
- `UMPIRE_BACKENDS=require mise exec -- cargo test -p umpire-testing --lib actual_tlc_fact_traces_replay_exact_vectors -- --nocapture`: exit 0, 1 passed, 20.99s. Real TLC fact witnesses reject changed, omitted and duplicate facts and a source-only fact mutant through the shared traversal.
- New public-entry mutation regression: healthy expiry edge plus forged state/outcome/facts/input/choice/tick and unselected starts; confirms precise transition/refusal and native state/outcome/facts. Intentionally dropping the native fact projection failed on `Seq([])` versus the exact ScheduleToClose fact (exit 101); restoring the source made the same test pass (exit 0).
- Strict all-target/all-feature Clippy, docs with `RUSTDOCFLAGS=-Dwarnings`, scoped cargo fmt and explicit rustfmt checks of the three activity files: all exit 0. `git diff --check`: exit 0.
- No authored Model, dependencies, protocols, snapshots, guest bytes or lifecycle records changed. No pending activity snapshots. Initial required-tool setup found the worktree jar absent; copied the root checkout's pinned jar and passed hash admission before all recorded successful TLC runs.

Build environment: `CARGO_TARGET_DIR=/home/agent/.cache/codex-build/umpire-milestones-20261024/fn2526 CARGO_PROFILE_DEV_DEBUG=0 CARGO_PROFILE_TEST_DEBUG=0 CARGO_INCREMENTAL=0 CARGO_BUILD_JOBS=2`.

Preserved proof: `/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/milestones-resume-20261024/fn2526-proof/manifest.json` contains SHA-256 hashes for exact proof binaries, source copies, pinned jar and logs. The archived running activity executable and restored final executable match byte-for-byte: `4756e14014338d6572982a4eccc46f40a1ed13d834fcf086cdd2df72bf12827a`. Jar SHA-256: `936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`; Java Temurin 27+35, rustc 1.99.1. The workspace and solely owned build cache remain present; no owned command is running.

Conductor obligations: integrate this commit with the landed tooling lint attributes, run the integrated full model gate/current guest checks, obtain independent task review and fn-250 supplemental R3 requirement review, then complete lifecycle records. No full gate, guest regeneration, independent impl-review or `flowctl done` ran in this lane.

stage: impl-review - skipped(policy: parallel-wave - conductor owns the gate)

Conductor final acceptance: source and regenerated current guest integrated in umpire; actual gpt-6.1-sol high implementation review reached SHIP (including the guest refresh supplement for fn2514). Independent correctness and standards quality audits found zero issues. The full TLC/Apalache-required UMPIRE_APALACHE=require make model-validate passed on frozen model tree abe4560501278986bc8ab74853d07b5b7128f30e at02751bfef5: full1188.040s, inner6.155s, exit0. This includes every workspace/native-tool, wasm/native-control, current guest freshness and all three Go host stage. Final source-boundary, snapshot ownership and strict workspace all-target/all-feature clippy also passed. Durable final receipt: .flow/artifacts/consolidation-20261025/final-validation-02751bfef5/receipt.json.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: e0d4262694215beaa26a8b7d48e56e062b04e2b5, 0db59f2f50, 02751bfef5
- Tests: mise exec -- cargo test -p umpire-testing -p umpire-temporal-tests --test fixtures --lib, mise exec -- cargo test -p umpire-testing -p umpire-temporal-tests --all-features, UMPIRE_BACKENDS=require mise exec -- cargo test -p umpire-temporal-tests --test fixtures proof_checks -- --nocapture, UMPIRE_BACKENDS=require mise exec -- cargo test -p umpire-temporal-tests --test fixtures activity_tests -- --nocapture, UMPIRE_BACKENDS=require mise exec -- cargo test -p umpire-testing --lib actual_tlc_fact_traces_replay_exact_vectors -- --nocapture, mise exec -- cargo test -p umpire-temporal-tests --test fixtures shared_replay_checks, mise exec -- cargo clippy -p umpire-testing -p umpire-temporal-tests --all-targets --all-features -- -D warnings, RUSTDOCFLAGS=-Dwarnings mise exec -- cargo doc -p umpire-testing -p umpire-temporal-tests --all-features --no-deps, mise exec -- cargo fmt -p umpire-testing -- --check, mise exec -- rustfmt --edition 2024 --check temporal/feature/activity/proof_checks.rs temporal/feature/activity/policy_checks.rs temporal/feature/activity/tests.rs, git diff --check, Actual Codex gpt-6.1-sol high SHIP implementation review; same-family independent CLI, Independent correctness and standards audits: zero findings, UMPIRE_APALACHE=require make model-validate (02751bfef5 frozen current model tree; exit0;1188.040s full,6.155s inner; TLC/Apalache/workspace/guests/3Go hosts green)
- PRs: