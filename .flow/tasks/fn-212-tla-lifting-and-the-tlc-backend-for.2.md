---
satisfies: [R2, R3]
---
# fn-212-tla-lifting-and-the-tlc-backend-for.2 Lift, emit TLA+, run TLC; agreement goldens on Kettle and the shared fixtures

## Description
Implement the owner-required TLC scaling backend using `model/design/second-opinion.md` and the parent spec. The owner settled yes on 2026-10-19; the small current corpus supplies agreement controls, not a reason to defer the backend.

**Touches:** [model/umpire/lift/**, model/umpire/backends/tlc/**, model/umpire/testing/**, model/umpire/tests/Cargo.toml, model/umpire/tests/build.rs, model/umpire/tests/fixtures.rs, model/umpire/tests/fixtures/*/expected/**, model/Cargo.toml, model/Cargo.lock, model/.gitignore, model/temporal/testing/Cargo.toml, model/temporal/testing/fixtures.rs, model/temporal/testing/build.rs, model/temporal/features/activity/expected/**, model/scripts/**, Makefile, model/README.md, model/DESIGN.md, model/design/second-opinion.md]

### Approach

- Use a build-time `syn` pass over authored source. Keep `#[machine]` registration-only. Own an internal typed tree, explicit dependency/module/cfg admission, the liftable subset and located warnings/refusals. Do not export the native checker's computed graph as a substitute for independent source lifting.
- Emit TLA+ modules/configs using the author's Rust names, bounded integer semantics, ordered updates, registered safety/postcondition/progress claims and named assumptions. Current formatting is standard Display/Debug (fn-220); do not restore Spell.
- The earlier task wording about fairness on internal actions is superseded by fn-214: only authored `fair(A)` entries emit `WF_vars(A)`. Internal actions add no fairness. Preserve native maximal emitted-edge path semantics, finite scoped dead ends and real-transition bounds; prove the proposed private observer encoding with the decision note's controls before claiming liveness agreement. Do not change Model transitions or add fairness to force agreement.
- Keep source-local reference checks independent of TLC. Disagreement fails with diagnostic evidence rather than fitting either Model or engine to the other. A cut native exploration proves no agreement; scope larger-Model comparisons explicitly.
- Pin the official `tla2tools.jar` version and integrity, fetch it under `model/build/`, and run through the mise JDK. Validate missing/wrong versions, subprocess failures, limits, complete dumps, deterministic cleanup and decoding. Keep deadlock checking enabled while representing intentional completion explicitly.
- Decode typed states/counterexamples and replay through Machine::step. Project only declared private observer fields/steps, refusing name collisions; deduplicate the projected state set. Compare complete sets and verdicts, not just state counts. Preserve action identity, choices, outcomes and facts wherever the admitted source uses them.
- Register a TLC consumer through the existing shared fixture runner. Kettle and the shared fixtures the subset admits own emitted `<Model>.tla`, `<Model>.cfg` and `tlc.txt` goldens. Unsupported expressions must warn/refuse by source location with golden controls, never disappear or turn into passing results.
- Measure actual required TLC execution and protect the 30 s full/10 s inner soft budgets by batching or explicit `UMPIRE_BACKENDS=require`. Default runs must make the absence of TLC evidence clear; required runs fail for missing/failed/unselected backend coverage. Keep native and existing WASM/Go gates intact.

### Quick commands

`make model-check`; `UMPIRE_BACKENDS=require make model-check`; `make model-time`. Add focused lift/emitter/real-TLC controls before the complete gates. Preserve raw command outputs and real version/state/verdict evidence.
## Acceptance
- Kettle: TLC reaches 283 states, verdicts agree; `Kettle.tla`, `Kettle.cfg`, `tlc.txt` goldens.
- An out-of-subset expression is warned on and refuses emission naming it (golden).
- Gate within R30.

## Done summary
Implemented independent authored-source lifting, typed TLA+ emission and pinned real TLC execution. The standalone Module entry point checks registered safety, postconditions and progress without native exploration; native complete-set/verdict agreement is a separate fixture obligation. Source admission, explicit fairness and private observer projection preserve the supported semantics; unsupported syntax refuses by location rather than disappearing or changing the Model.

All 19 admitted shared fixtures agree on complete reachable sets and property verdicts; both Kettles have 283 projected states, and returned counterexamples replay through actual checked Model steps with action, choice, outcome and facts. Six shared source forms retain located refusals. Current system Phased/reject/closable source remains explicitly unsupported with no system TLC evidence: old Activity artifacts were replaced by the current located refusal, without changing authored Model behavior. The independent four-state/native-cut control proves execution decoupling, not a performance benchmark.

The parallel lane was integrated locally, its source commits remain reachable, and its clean inactive worktree/branch were removed after all writers stopped. Main integration added checked-step refusal propagation with a runtime red/green control, handled zero-admission generated bindings, and preserved native/WASM/Go gates. Detailed original and integrated evidence is in summary.md, integrated-summary.md and fix-summary.md beside this file.

The initial correctness/standards audits and their rechecks are preserved verbatim in quality-audits.md. Seven original semantic findings and two structural findings were fixed; an optional fixture-root rendering duplication remains nonblocking. The first formal three-draw review returned NEEDS_WORK for four further source-semantic issues. All four were fixed, including raw-identifier and associated-type/signature variants of the same admission boundaries. Compiled native controls and actual pinned-TLC red/green runs establish narrow signed remainder overflow, checked overrides, guarded world selection and conditional-source refusal. The same-receipt single re-review returned SHIP, all R1–R3 met, zero new blockers. Reviewer cached binaries passed eight controls; fresh reviewer Cargo was read-only blocked, while worker full verification supplied execution evidence.

Final verification on afecc53237 passed make model-check, UMPIRE_BACKENDS=require make model-check (142 seconds), and make model-time. The measured warm default gate is 10.704 seconds and edit/test loop 2.157 seconds, within the 30/10-second soft targets. Both nested Go hosts passed. Workspace Clippy, format and diff checks passed; changed-Go lint reported no affected Go packages. A stale old-worktree generated-protocol cache failure is retained with exact old/new artifact evidence; unchanged current source passed using the confirmed main build cache, without protocol changes or cache deletion.

The conductor's integrated-target verification honored all three receipts at the reviewed source HEAD immediately before done:
GATE_SKIPPED:model:green-receipt afecc532 - make model-check
GATE_SKIPPED:model-required:green-receipt afecc532 - UMPIRE_BACKENDS=require make model-check
GATE_SKIPPED:model-time:green-receipt afecc532 - make model-time

stage: impl-review - ran (three Codex gpt-6.1-sol/high draws, then same-receipt single fix re-review; final SHIP; same model family as writer)
stage: plan-sync - skipped(config: planSync.enabled != true)
Tier: session (jev-unavailable(no_key))
Tracker sync: n/a (bridge inactive)
Owner requested wrapping up in-flight work and stopping; no follow-on task was started.
## Evidence
- Commits: fe17b131d4dfae138ef9a6d816ae4988fe5b4ed3, 1b33ecb31198b3f7b2708d5fbc70213569c18fa2, eba320b1fb6e7cffa9c16daa9896cdf8bf682a18, 6da4094f7a57d243abd9e9655a9255384483f556, c3376c2aede5a32aaf242407adcb62f8eb34d556, 31ae8ec7f31f20c7b314f077d72b3617d7528494, afecc53237befc50b426faa30ece9526f5ab60eb
- Tests: CARGO_INCREMENTAL=0 CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target timeout 600 make model-check: exit 0 at afecc53237; fix-final-model-check.log., CARGO_INCREMENTAL=0 CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target UMPIRE_BACKENDS=require RUST_TEST_NOCAPTURE=1 timeout 600 make model-check: exit 0 at afecc53237, 142s; fix-final-required.log; all 19 shared fixtures agree, both Kettles 283 states, counterexamples replayed., CARGO_INCREMENTAL=0 CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target timeout 600 make model-time: exit 0; fix-final-model-time.log; gate 10.704s, inner 2.157s., Actual pinned-TLC i8/i16 MIN%-1 controls: red native overflow versus old TLC success, then green TLC Assert2132 typed overflow refusal; fix-remainder-* logs and raw outputs., Compiled manual admission controls: checked override, guarded world selection, cfg/cfg_attr and raw spellings, associated types/signatures; red then green; 12 original plus 6 new controls pass; fix-manual-* logs., Final workspace all-target/all-feature Clippy, cargo fmt --all --check and git diff --check passed; original 19-source admission inventory retained., GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=5c1e8cd19c make lint-code-fast: exit 0; no changed Go packages; integrated-lint.log. Subsequent fixes contain no Go changes., Source integration checked-step replay regression: integration-checked-step-red.log fails unchecked panic, integration-checked-step-green.log passes named refusal propagation., Formal three-draw review NEEDS_WORK finalized, all four retained findings fixed; same-receipt Codex gpt-6.1-sol/high re-review SHIP, zero new blockers, all R1-R3 met; impl-rereview.log and /tmp/impl-review-receipt-8f37faba39e2-fn-212-tla-lifting-and-the-tlc-backend-for.2.json., GATE_SKIPPED:model:green-receipt afecc532 - integrated-target gate check honored make model-check immediately before done., GATE_SKIPPED:model-required:green-receipt afecc532 - integrated-target gate check honored UMPIRE_BACKENDS=require make model-check immediately before done., GATE_SKIPPED:model-time:green-receipt afecc532 - integrated-target gate check honored make model-time immediately before done., Old isolated-target generated-protocol mismatch retained as failed evidence in fix-model-check.log and fix-stale-target-evidence.log; not counted as passing. Current source/protocol unchanged to fit it.
- PRs: