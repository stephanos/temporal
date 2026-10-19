---
satisfies: [R2]
---
# fn-221-rejecting-outcomes-reject-steps-phased.2 Phased, Phase and closable laws; the mini activity Model rejects cancel after closure

## Description
Add `Phase`, `Phased` and `closable::{terminal_remains, closed_rejects}` to `umpire_dsl`, carrying the archived closure laws and their caveats. Convert the current mini activity Model at `model/temporal/features/activity/model.rs`: `Phase::closed` names the terminal set once, closed `Client::Cancel` emits `Rejected(NotFound)`, and delegating postconditions register both shared laws. Regenerate only the activity goldens; retain the existing six phases and atomic Scheduled/Started cancellation.

**Size:** S
**Touches:** [model/umpire/dsl/src/**, model/umpire/dsl/tests/**, model/temporal/features/activity/model.rs, model/temporal/features/activity/expected/**, model/temporal/testing/**, model/temporal/realization/tests/realization.rs, model/temporal/hosts/go/parity_test.go, tests/testpilot_model_slice_test.go]

### Approach
- Hand-write `impl umpire_dsl::Phase for Phase` and `impl Phased for Model` inside the module; no new attributes or capability registry. Use the current standard formatting contract: `Phase: Copy + Eq + Hash + Debug`, not removed `Spell`. `Phased::closed` delegates to `self.phase().closed()`.
- `terminal_remains(after, before)` checks that an emitted step from a closed phase preserves that phase, including rejecting moves between different closed phases. `closed_rejects(after, before, outcome, expected)` additionally requires identical state and `Rejected(expected)`. Keep the unit-returning `check!` registration style; callers evaluate these laws inside claim frames.
- Preserve the archived caveats: terminal preservation alone promises neither rejection nor whole-state immutability; uniform closed rejection is entity-specific and does not model repeated-request-ID deduplication. Generic docs must not mention Temporal-specific entities or numeric RPC codes.
- Enumerate phases in the activity controls, deriving the terminal set by the single `Phase::closed` classification instead of duplicating a hand-maintained terminal list. Retain positive checks for all six phases and every action, and exercise the registered callbacks directly where actions are disabled so the laws are not merely vacuously green.
- Test genuine emitted rejected cancel self-loops from all four terminal phases. Hypothetical controls refute reopening and transitions between closed phases; accepted, wrong-rejection and state-changing edges from closure refute `closed_rejects`. Add a generic multi-field control distinguishing phase preservation from whole-state preservation.
- Kettle remains unphased and existing generic goldens unchanged. This task changes the independent activity Model contract according to R2; later tasks carry outcomes through the protocol and judge real cancel errors. Do not fit these laws to runtime behavior.
- Existing native realization, WASM/Go parity and live-slice controls assert no expected actions after terminal contradictory Describe. The newly specified rejected cancel self-loop makes `Client::Cancel` enabled even there. Capture those assertion failures after the Model change, then require exactly `["Client::Cancel"]` instead of empty while retaining the freeze checks. Only those assertion migrations are in scope in the three added test paths; protocol/binding changes and new live cancel scenarios remain .3/.4. Do not remove or weaken the existing controls.
- Verify the parent Quick commands using the existing Make-provided `UMPIRE_TEMPORAL_WASM` guest path for the direct Go integration command; preserve evidence that an unconfigured invocation is not an implementation regression. No protocol or realization changes in this task.
## Acceptance
- [ ] `closable` laws registered by delegating postconditions; both refuted by the hypothetical controls.
- [ ] Activity goldens show the four rejecting self-loops and the new claim; Kettle untouched.
- [ ] `make model-check` passes.

## Done summary
Model authors can share phase-preservation and entity-specific closed-rejection laws through `Phase`, `Phased` and unit-returning `closable` checks. The activity Model registers both laws and emits `Rejected(NotFound)` cancel self-loops from all four closed phases, while Scheduled and Started cancellation stays atomic.

R2 is covered by the generic multi-field `closure_checks_separate_phase_state_and_entity_specific_rejection` control and six activity controls. They distinguish phase preservation from whole-state preservation, check every phase/action pair through the registered callbacks, exercise emitted rejected cancels, and refute reopening, moves between terminal phases, accepted outcomes and wrong rejections. Only activity goldens changed. Three authorized downstream test assertions now require exactly `["Client::Cancel"]` after a contradictory terminal Describe and retain the freeze checks. No feature-entry route changed.

baseline: green for `make model-check`, `make umpire-check-model-slice`, and the configured direct Go selector. The initial direct invocation failed before edits because it used a nonexistent default guest path; `baseline-live.log` retains that failure. `baseline-live-configured.log` records the corrected green run using the inherited `CARGO_TARGET_DIR` guest. The preceding task's `fn2211/baseline-live.log` retains the unconfigured-selector failure. Baseline logs are beside this summary.

The new cancel reproduction failed with zero emitted steps before the Model change (`red-terminal-cancel.log`); the generic API control failed on missing phase APIs (`red-closable.log`). Native, live and Go-host parity controls then failed at the exact obsolete empty expected-action assertions (`red-native-expectation.log`, `red-live-expectation.log`, `verify-model-final.log`) before their authorized migration. Final model verification, all four live slice scenarios, the configured direct Go selector and nonfixing Go lint passed (`verify-model-parity.log`, `verify-slice.log`, `verify-live.log`, `final-lint.log`). The model gate includes formatting, clippy, documentation, generic boundary checks, native fixtures and wasm/Go parity. Kettle and all existing generic goldens remain unchanged.

The first review attempt failed before dispatch while publishing sidecars and was refunded. Publication probes succeeded; one normal retry produced three SHIP draws with no findings. Correctness, contracts and integration all ran `codex:gpt-6.1-sol:high`, from the writer's model family. Reviewer test attempts were blocked by read-only caches; worker gate logs above supply executable evidence. The merged receipt is `/tmp/impl-review-receipt-8f37faba39e2-fn-221-rejecting-outcomes-reject-steps-phased.2.json`; draw evidence is `.flow/review-fanout/afe4cf3a6c804de0a467111a0d3e0487/`.

GATE_SKIPPED:unittest:green-receipt bdaab350 - verified post-change model gate reused at final verification
GATE_SKIPPED:smoke:green-receipt bdaab350 - verified post-change slice gate reused at final verification
GATE_SKIPPED:live:green-receipt bdaab350 - verified configured direct Go selector reused at final verification

stage: impl-review - ran [2026-10-19T21:21:43Z..2026-10-19T21:24:40Z] (one refunded pre-dispatch transport failure, then three SHIP draws; model: gpt-6.1-sol at high)
stage: plan-sync - skipped(config: planSync.enabled != true)
Tier: session (jev-unavailable(no_key))
Tracker sync: n/a (bridge inactive)
## Evidence
- Commits: bdaab35025b7bef6c95fd22ad4c15e20996bb385
- Tests: mise exec -- cargo test -p umpire-dsl --test closable, mise exec -- cargo test -p umpire-temporal-tests --test fixtures, make model-check, make umpire-check-model-slice, UMPIRE_TEMPORAL_WASM=/home/agent/.cache/codex-build/cargo-target/wasm32-wasip1/release/umpire_temporal_guest.wasm CGO_ENABLED=0 mise exec -- go test -tags 'test_dep integration' ./tests -run '^TestTestpilotModelSlice$' -count=1, GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=2487b1f7b9 make lint-code-fast, GATE_SKIPPED:unittest:green-receipt bdaab350 - verified post-change model gate reused at final verification, GATE_SKIPPED:smoke:green-receipt bdaab350 - verified post-change slice gate reused at final verification, GATE_SKIPPED:live:green-receipt bdaab350 - verified configured direct Go selector reused at final verification
- PRs: