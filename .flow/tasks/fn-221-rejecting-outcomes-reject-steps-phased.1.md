---
satisfies: [R1]
---
# fn-221-rejecting-outcomes-reject-steps-phased.1 Outcomes and reject!: Rejection, Outcome, the outcome frame, macro edge rules and outcome-taking postconditions

## Description
Add `Rejection`, step `Outcome`, `outcome::{evaluate, reject}` and statement-form `reject!` to `umpire_dsl`, mirroring scoped `claim::{evaluate, fail}` and `check!`. The macro applies the parent spec's accepted/disabled/rejected/invalid edge table and permits an outcome-taking postcondition. Only modules using `reject!` switch their associated outcome from unit. The checker refuses rejection after mutation, naming the action and before state.

**Size:** M
**Touches:** [model/README.md, model/umpire/dsl/src/**, model/umpire/dsl/tests/**, model/umpire/dsl/macros/src/machine.rs, model/umpire/dsl/macros/tests/**, model/umpire/tests/fixtures/**, model/umpire/checker/src/**, model/umpire/checker/tests/**, model/umpire/wasm/src/refusal.rs, model/umpire/wasm/src/runtime.rs, model/umpire/wasm/tests/**]

### Approach

- Current fn-220 formatting supersedes the spec's Spell references: states/actions use Display, outcomes/facts use Debug. Derive the new outcomes' canonical Debug spellings rather than restoring Spell.
- The DSL already exports an aggregate `Outcome::{Verified,Refuted,Unknown}`. Reserve `Outcome` for the requested step outcome and rename that aggregate to `CheckOutcome`, updating its existing live consumers. Keep rendered aggregate variants and all pre-existing golden bytes unchanged. Do not combine these two concepts or keep a misleading old alias.
- New shared fixtures cover accepted, disabled, rejected self-loop, rejecting helper and rejecting-then-mutating refusal. Preserve all existing Model behavior and Kettle goldens; add a separate macro snapshot for rejecting modules while leaving nonrejecting snapshots unchanged.
- Follow claim's frame restoration on nested evaluation and unwinding. The R1 requirement that rejection outside an active frame is inert governs the parent text's contradictory optional debug-assert wording; direct out-of-frame use must not panic. Pin that behavior and helper return semantics.
- Add the minimal generic checker refusal path and its guest refusal-code mapping if needed. Do not invent a Model edge to represent the invalid implementation or allow changed-state rejection to pass as disabled. Root approves the listed test/caller/error-mapping scope expansions required for a green complete model gate.
- Explain any new checked-step/error interface to the conductor before committing a broad Machine API change, so the parallel TLC lane can adapt. Its worktree does not modify DSL/checker/Model source.
- Root approved `Machine::checked_step` with a default `Ok(self.step(..))` and a typed `StepError`, preserving manual implementations and nonrejecting macro snapshots. All checker/guest replay and Answer response-validation callers must use it; changed-state rejection is a typed refusal before publication, not a disabled edge, unexplained observation or WASM trap. The runtime/test scope expansion above covers these maintained callers and controls.

### Quick commands

Focused native fixtures/macro/negative controls first, then `make model-check`. The task acceptance and parent edge table remain authoritative. Preserve red/green evidence and existing golden identities.
## Acceptance
- [ ] R1 fixtures pass natively; rows/report spell outcomes; rejecting-then-mutating is a named checker refusal.
- [ ] Kettle and every pre-existing fixture golden and the existing macro snapshot are byte-identical.
- [ ] `make model-check` passes.

## Done summary
Model authors can record accepted or rejected step outcomes with statement-form `reject!` and inspect them in postconditions. Checker exploration and every runtime replay or enabledness path use `checked_step` to refuse rejected mutations with the action and source state before publishing changes.

R1 is covered by the shared `rejecting_steps` rows/report fixture, the checker `rejecting_mutation` refusal fixture, five scoped-outcome/activation tests, and the runtime refusal/rollback table covering Explore, Judge replay, expected-action enumeration, Answer delivery and response validation. Existing goldens and the nonrejecting macro snapshot remain byte-identical to the base. The aggregate result is now `CheckOutcome`; rendered aggregate variants remain unchanged.

baseline: green for `make model-check` and `make umpire-check-model-slice`. baseline: red (the raw Go selector failed pre-edit because `UMPIRE_TEMPORAL_WASM` was unset). Supplying the built guest path repaired the invocation and passed before edits. Baseline observations are retained in `baseline-model.log`, `baseline-slice.log`, `baseline-live.log` and `baseline-live-configured.log` beside this summary.

Verification passed after the final code change: five outcome tests, workspace clippy with warnings denied, `make model-check`, `make umpire-check-model-slice`, and the configured raw `TestTestpilotModelSlice` selector. Final formatting and clippy passed; Go lint reported no changed Go packages. Raw observations are in `green-reject-comparison.log`, `verify-model-fix.log`, `verify-slice-fix.log`, `verify-live-fix.log` and `final-lint.log`.

The integration review found that `reject != value` incorrectly activated typed outcomes. The associated-type control failed with E0271 before commit `e1fae10f8a`, then passed after requiring a delimited macro argument group. The same-receipt second review returned SHIP. Codex reviewer `gpt-6.1-sol` at high is from the writer's model family. Reviewer Cargo reruns were blocked by its read-only cache; worker gate runs above supplied executable verification. Review receipt: `/tmp/impl-review-receipt-8f37faba39e2-fn-221-rejecting-outcomes-reject-steps-phased.1.json`.

Fresh correctness/standards audits and the correctness recheck of the fix are preserved in `quality-audits.md`. The review fix was captured as `bug/integration/match-complete-macro-invocations-when-2026-10-19`. The conductor's task .2 description-only preparation was included in the implementation checkpoint. No feature-entry route changed.

GATE_SKIPPED:unittest:green-receipt e1fae10f - verified post-change model gate reused at final verification
GATE_SKIPPED:smoke:green-receipt e1fae10f - verified post-change slice gate reused at final verification

stage: impl-review - ran [2026-10-19T21:04:35Z..2026-10-19T21:10:19Z] (NEEDS_WORK then SHIP; two rounds)
Tier session (jev-unavailable(no_key))

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: d368e1ce6a9248fa4745e19a5ba268d35a3185bc, e1fae10f8ae5342e6063bdabcf78e0d2a2553186
- Tests: make model-check, make umpire-check-model-slice, mise exec -- cargo test -p umpire-dsl --test outcome, mise exec -- cargo clippy --workspace --all-targets -- -D warnings, UMPIRE_TEMPORAL_WASM=/home/agent/.cache/codex-build/cargo-target/wasm32-wasip1/release/umpire_temporal_guest.wasm CGO_ENABLED=0 mise exec -- go test -tags "test_dep integration" ./tests -run "^TestTestpilotModelSlice$" -count=1, GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=7c89c8855e make lint-code-fast, GATE_SKIPPED:unittest:green-receipt e1fae10f - verified post-change model gate reused at final verification, GATE_SKIPPED:smoke:green-receipt e1fae10f - verified post-change slice gate reused at final verification
- PRs: