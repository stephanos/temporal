---
satisfies: [R2, R3]
---
# fn-212-tla-lifting-and-the-tlc-backend-for.2 Lift, emit TLA+, run TLC; agreement goldens on Kettle and the shared fixtures

## Description
Implement the owner-required TLC scaling backend using `model/design/second-opinion.md` and the parent spec. The owner settled yes on 2026-10-19; the small current corpus supplies agreement controls, not a reason to defer the backend.

**Touches:** [model/umpire/lift/**, model/umpire/backends/tlc/**, model/umpire/testing/**, model/umpire/tests/Cargo.toml, model/umpire/tests/build.rs, model/umpire/tests/fixtures.rs, model/umpire/tests/fixtures/*/expected/**, model/Cargo.toml, model/Cargo.lock, model/scripts/**, Makefile, model/README.md, model/DESIGN.md, model/design/second-opinion.md]

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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
