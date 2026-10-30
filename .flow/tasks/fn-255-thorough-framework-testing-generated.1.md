---
satisfies: [R1, R7]
---
# fn-255-thorough-framework-testing-generated.1 Spike: generate admitted Models and run them through every backend

Touches: [model/design/generated-differential.md, .flow/tmp/fn255-spike/**]

## Description
Settle how to generate Models for the differential test, at mandatory source level and optionally tree level (spec Architecture R1), before building the harness. The throwaway probe is retained as reproducible evidence; the verdict is committed. No prototype implementation is merged.

**Size:** M
**Files:** `model/design/generated-differential.md` (verdict); probe, README and `run.sh` under `.flow/tmp/fn255-spike/`.
**Touches:** [model/design/generated-differential.md, .flow/tmp/fn255-spike/**]

### Approach
- Tree level: write a small proptest strategy for admitted trees (two enum fields, one bounded int, one `Option<Finite>`, one guarded handler with `reject!`, one invariant). Check whether interpreter and TLC can consume a tree built without the front end, and whether that breaks the independence constraint (spec Edge Cases).
- Source level is required: print the same shapes as `#[machine]` source. Compile N per scratch crate; measure rustc time per batch and per Model. Choose a nonzero bounded source count for required-backend runs; tree-only coverage cannot replace macro/front-end coverage.
- Demonstrate bounded extraction/comparison of complete starts and labeled transitions (action, successor, outcome and facts) from TLC. Existing state-set/verdict and counterexample-replay helpers do not establish complete edge agreement. Account for extraction launches/work in R7 budgets.
- Require successful backend completion before comparing results; probe matching refusals and confirm they fail rather than pass as zero-state agreement.
- Measure cases per minute at each level, with TLC per-run cost from fn-254's baseline, and test batching several generated Models into one TLC module.
- Seed a swapped comparison and an outcome/fact-only mutation in scratch copies. Confirm both are caught and shrunk, including the latter when reached states and claim verdicts remain unchanged.
- Propose the bounds for R7: maximum states/edges/depth per generated Model, cases per tier, TLC launches per case (with batching), and a per-case timeout, each derived from the measurements.
- Answer the spec's three Parked unknowns, or name them as owner questions.

### Investigation targets
**Required:**
- `model/umpire/ir/src/lib.rs` — admitted tree types
- `model/umpire/backends/native/src/interpret.rs` — interpreter entry
- `model/umpire/backends/tla/` — emitter entry
- `model/umpire/backends/native/tests/main/oracle.rs` — existing proptest use
- `model/umpire/dev/testing/src/backend.rs::{check_exploration, agreement}` and `interpreted.rs::compare` — current agreement coverage and matching-refusal behavior

## Acceptance
- [ ] Verdict retains mandatory source coverage, chooses any additional tree layer, harness home, nonzero required-mode source count and TLC batching, with measurements.
- [ ] Probe demonstrates bounded complete starts/labeled-transition extraction; same-state/verdict outcome/fact mutations fail, and matching refusals fail.
- [ ] Seeded lowering bug caught and shrunk at the chosen level.
- [ ] Probe reproducible from `.flow/tmp/fn255-spike/run.sh`.
- [ ] Proposed R7 bounds (Model size caps, cases per tier, TLC launches per case, per-case timeout) with the measurements behind them.

## Done summary
Measured and reviewed the mandatory generated-source spike without registering a production harness or adding gate work. The printer admits two enums, a bounded integer, Option<Finite>, guarded arithmetic, rejection, facts, starts and an invariant. Compiled handlers, interpreted tree and separately source-lifted TLC agree on complete starts, states and labeled transitions; same-state/verdict fact/outcome mutants and a swapped comparison are caught, three source reductions retain detection, and matching cap refusals fail.

The selected two-case family has 2 starts each, 10/6 states, 30/16 edges and 96 augmented assignments. Same-seed individual checks observed 6 model + 2 discovery JVMs; a disjoint-union batch observed 2 model + 1 discovery. Exact-current checked batch measured 4.109 s compile + 1.477 s runtime; other measured rows retain their exact earlier-adapter provenance. Source-family caps, per-process limits and proposed one/default and two/required counts are justified in model/design/generated-differential.md. Default differential share is proposed at 11 s warm (10 compile + 1 native), required at 45 s; authoritative budget/gate activation remains .7/fn254.12. Optional tree-only generation is declined. Batching is explicitly same-schema and Verified-or-fail; refutable-claim families need per-case operators or individual fallback in .2.

Recoverable archive and manifest verify 141 members; 106 historical run files and prior archive are retained unchanged. Coordinator freshly checked all archive hashes, six native/TLC/batch graph equalities and full-framework source projection, and reran isolated Python nonzero/timeout/missing-jar/source-only controls without altering historical evidence. The facade/suite/catalog/protocol/manifests/locks match the pinned integrated source; only named cache/quick tool deltas differ, with cache bypass explicit. The initial malformed preflight regression's own compiler group was stopped and its 408.8 MiB cache cleaned; no JVM ran in that attempt. Idle spike build caches were cleaned after preserving the exact proof binary and source/measurement evidence.

stage: impl-review - ran (three lenses SHIP at 4285914b1b; six scope/provenance/preflight follow-ups addressed at dddeead063; fresh scoped SHIP, model: claude-fable-5-1 high). Maintained harness, dropped-guard mutation/fixture pinning and final gate admission remain later tasks, not claimed here.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 2901123c82, fa9b20fa31, d9a378401b, 4285914b1b, dddeead063
- Tests: retained final-pair, final-batch and exact-current checked-batch real compiled/interpreted/TLC runs, three compiled source shrinking controls, matching-refusal and label-only mutation controls, fresh coordinator archive/source projection/hash/graph equality audit (141 members,6 graph pairs), fresh isolated-copy python3 check-runner.py (no Cargo/Java), python3 -m py_compile and bash -n on runner sources, strict scratch-probe clippy and rustfmt recorded in verification evidence
- PRs: