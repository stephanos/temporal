---
satisfies: [R4, R5]
---
# fn-214-progress-vocabulary-on-kettle-scopes.3 Kettle's progress questions as fixtures; fairness and emitter contract in DESIGN.md

## Description
Implement R4 and R5 with supplementary Kettle question fixtures and the narrow query seam they need.
This task consumes .1/.2; it does not introduce the future suite's Space, Run or starts DSL.

**Size:** M
**Files:** checker registered-progress query, shared fixture adapters/questions and current documentation.
**Touches:** [model/umpire/checker/src/explore.rs, model/umpire/checker/src/progress.rs, model/umpire/checker/src/lib.rs, model/umpire/checker/src/refusal.rs, model/umpire/checker/tests/**, model/umpire/tests/fixtures/kettle_progress*/**, model/umpire/tests/support/**, model/umpire/wasm/src/refusal.rs, model/umpire/wasm/tests/dispatch.rs, model/umpire/proto/README.md, model/README.md, model/DESIGN.md]

### Approach

- Add a narrow `Result::verify_progress(&self, world, &registered_claim) -> Result<ClaimResult, Refusal>` operation over an existing exploration. Share one borrowed evaluator with explore-time checking, returning a local result rather than mutating cached claims. Do not re-explore, clone retained rows or duplicate the progress algorithm.
- Retain the original ordered action catalog and registered progress descriptors. Reject changed ordered catalogs before indexing completed-action columns. Resolve a unique original registration by name and execute its canonical callbacks, applying only the supplied explicit assumption/bound overrides. Refuse absent/ambiguous registration. A public descriptor does not establish provenance; do not treat function addresses or names as proof of same-world identity. Document the caller requirement to use the same unchanged world. A fabricated same-name selector must never execute fabricated predicates.
- Use precise refusal diagnostics for catalog/registration misuse. Existing action-admission and budget categories do not describe these errors. Update the exhaustive WASM refusal bridge and native generated-protobuf roundtrip using the existing malformed-request category where accurate; no schema, operation, generated Go or ABI change is needed.
- Import the canonical attribute Kettle by its existing fixture source path. Use a fixture-side generic Machine adapter to select room/cold starts, delegate actions/internal/step to the original world and forward the original registered progress callbacks. Construct starts through the existing public Model fields. Do not duplicate transition or boils predicate bodies, expose private methods by rewriting the sample, or change either canonical sample directory.
- Select the original `boils` registration from `Machine::progress()`. The future `Model::boils.assuming(...)` suite notation is not yet an existing descriptor API; use the registered claim now and preserve the future obligation for fn-213.
- Produce supplementary checker-owned `report.txt` and `paths/*.txt` with a fixture `run()` using existing Files/consumer discovery and deterministic double rendering. Do not add arbitrary output filenames or a second fixture runner.
- Pin query invariants: unchanged cached reports/counts/states/rows, no extra step calls, one combined evaluator call per reached state per query, no evaluator calls with zero starts, registration/catalog refusals, independent override replacement and existing hole/cut/panic/refutation behavior. Validate available metadata even with zero starts; do not fabricate a state.
- Compare the entire two canonical sample directories with the committed task baseline, including source and snapshot metadata. Preserve them byte-for-byte while pinning all R4 questions in separate fixture directories.
- Update README and DESIGN for explicit fairness, numeric authoring bounds, query operation and legacy compatibility. Record that fn-212 emits `WF_vars(A)` only for explicit fair entries. Keep the generic hint text without claiming that fair(heat) repairs Kettle's press/depress cycle; heat is disabled while the element is off.

### Investigation targets

**Required:**
- `model/umpire/checker/src/explore.rs` - retained graph, registration and cached report seam.
- `model/umpire/checker/src/progress.rs` - .2's shared progress algorithm and local result boundary.
- `model/umpire/tests/fixtures/sample_kettle/model.rs` - canonical world, public fields and private authored predicates.
- `model/umpire/testing/src/lib.rs` - fixture producer and Files API.
- `model/umpire/testing/src/runner.rs` - recognized output ownership and deterministic comparison.
- `model/archived/framework/samples/Kettle.test.scala:255` - reference questions, paths and off-by-one boundaries, read-only.

**Optional:**
- `model/DESIGN.md` - current compatibility and follow-up sections.
- `.flow/tmp/fn214-query-seam-findings.md` - read-only design advice, not an implementation or task record.

### Quick commands

From `model/`, `mise exec -- cargo test -p umpire-checker -p umpire-tests -p umpire-testing`.
From the repository root, run `make model-check` and record one `make model-time` output under the 10 s / 30 s soft policy.

## Acceptance
- [ ] Supplementary deterministic goldens pin all six R4 questions with original registered predicates: longest 81 from 20, longest 141 from -40, unrestricted depress/press cycle, press-only deadlock in the specified switched-on state, bound-80 missed deadline, and budget-10 Unknown.
- [ ] The new query operation reuses an existing exploration and one progress checker, honors replacement assumptions/bounds, refuses catalog/registration misuse and does not duplicate Kettle behavior. Query path/hint fixtures pin actual deterministic witnesses; cached results/rows remain unchanged and no transition is re-executed.
- [ ] Both complete canonical sample directories are byte-identical to the task baseline; no aliases, duplicated Models, new runner, unregistered expected filenames or future suite DSL are added.
- [ ] README/DESIGN describe explicit fairness, bound/query syntax, legacy compatibility and the future `fair(A)` to `WF_vars(A)` contract; no implicit internal fairness remains in the current or future emitter description.
- [ ] Focused tests, `make model-check` and recorded `make model-time` pass; fn-213's ergonomic tests.rs presentation remains a named downstream obligation.

## Done summary
Checker clients can ask another registered progress question over a retained exploration with `Result::verify_progress`, and Kettle's supplementary fixtures pin all six required answers. README and DESIGN now describe explicit fairness, positive numeric bounds, query overrides, legacy compatibility and fn-212's explicit `fair(A)` to `WF_vars(A)` emitter contract.

The query resolves a unique retained registration by name, runs canonical callbacks and applies only explicit assumption/bound overrides. It retains the original ordered action catalog, refuses catalog/registration misuse before column indexing and shares one borrowed evaluator with exploration. Cached reports, counts, states and rows remain unchanged; query checking executes no transitions. Public documentation requires the same unchanged world because descriptors cannot establish provenance.

R4 coverage lives in `umpire/tests/fixtures/kettle_progress`, its generic `support/progress_starts.rs` adapter, checker `progress_bounds` and `support/progress_evaluation.rs`. The questions pin longest 81/141, the depress/press cycle, press-only deadlock in the switched-on room state, bound-80 missed deadline and budget-10 Unknown. An additional fair-heat question preserves the cycle because heat is disabled while off. A generic cycle regression and the Kettle lasso both failed with cycle_start 2 before the SCC witness rotated to its earliest outstanding obligation; both now pin cycle_start 1. The fixture imports the existing canonical source module and delegates its original transitions and registered predicates through public state fields. Existing Files/consumer discovery double-renders report and path goldens.

Query regressions cover repeated cached rendering and counts, canonical manual and combined callbacks despite fabricated selectors, one combined evaluation per reached state, no evaluations with zero starts, catalog additions/removals/replacements/reordering, missing/ambiguous names, independent override replacement and consistency, available metadata validation, holes, cuts and caught defects. The native generated-protobuf dispatch test covers all three new refusals as malformed requests with distinct diagnostics. No protocol schema, generated Go or ABI changed.

Both complete `sample_kettle` and `sample_kettle_trait` filesystem inventories and every file's Git blob hash match task baseline `6f22a6f7c76a8f4c67673746998d3ce0f9825720`. The comparison includes directories, source, snapshot metadata and payloads, and detects ignored/untracked entries. The generated registry under `/home/agent/.cache/codex-build/cargo-target` points to this checkout. `spec_base` remains `ef61dcbee323abbe7dc3049d3d1ea25517d9a3fc`.

baseline: green. The focused suite, `make model-check` and `make model-time` exited 0 before edits. Logs are `.flow/tmp/fn214-3-baseline-focused-correct.log`, `fn214-3-baseline-model-check.log` and `fn214-3-baseline-model-time.log`. Baseline timing was 2.528 s inner and 4.151 s full. An initial shell-startup directory reset observed the wrong surface and is inconclusive, preserved in `fn214-3-baseline-focused.log`; the corrected invocation explicitly entered the checkout inside its nested shell.

RED evidence is preserved in `.flow/tmp/fn2143-query-red.log` (missing API), `fn214-3-red-wasm.log` (missing exhaustive bridge), `fn2143-query-cycle-red.log` and `fn2143-kettle-cycle-red.log` (wrong witness entry). Initial new-golden failures remain in `fn2143-kettle-red.log` and `fn2143-kettle-cycle-fixed.log`. The first full gate's two lint failures remain in `fn214-3-model-check.log`; the final helpers and borrowed expectation pass the unchanged lint policy.

Final focused tests, native WASM dispatch and the full `make model-check` passed. The full gate covers formatting, denied-warning clippy/docs, native fixtures, stock WASI guest/core checks and CGO-disabled wazero Go-host parity. Logs are `.flow/tmp/fn214-3-final-focused.log`, `fn214-3-wasm.log` and `fn214-3-model-check-correct.log`. `make model-time` passed at 3.498 s inner and 4.931 s full against the 10 s / 30 s soft targets, recorded in `fn214-3-model-time.log`. Phase-5 classification required full gates; unchanged post-gate source commit 42d96f4b was receipted and honored after review.

GATE_SKIPPED:unittest:green-receipt 42d96f4b - verification reused the current task's post-gate pass
GATE_SKIPPED:model-check:green-receipt 42d96f4b - verification reused the current task's post-gate pass
GATE_SKIPPED:model-time:green-receipt 42d96f4b - verification reused the current task's post-gate pass

stage: impl-review - ran [2026-10-19T16:35:48Z..2026-10-19T16:37:51Z] (codex:gpt-6.1-sol:high, SHIP on all three axes, zero findings)

Review receipt: `/tmp/impl-review-receipt-8f37faba39e2-fn-214-progress-vocabulary-on-kettle-scopes.3.json`. Writer and reviewers use the same model family under explicit project routing. Reviewers' fresh Cargo runs were blocked by read-only build locks; one correctness draw ran cached test binaries successfully. Worker gates supply the fresh dynamic evidence. Original foreground session 43171 and PID 4080058 exited 0 despite transient dispatch_interrupted metadata. The original three SHIP draws finalized without redispatch, reset or refund. Clean first-pass review required no memory capture.

Tier: session (jev-unavailable(no_key)); explicit project routing preserved

Requested implementer was gpt-6.1-sol at medium; host execution metadata did not expose the executed writer model. Two native child agents implemented the disjoint checker and Kettle fixture surfaces, then stopped before final gates, staging and commit. The changed client route is `Checker::explore` followed by `Result::verify_progress`; the conductor owns feature-map updates. Fn-213's ergonomic Kettle `tests.rs` presentation remains a named downstream obligation. The conductor owns spec completion review and closure.

stage: plan-sync - skipped(config: planSync.enabled=false)
## Evidence
- Commits: 42d96f4b6b5b64b525b42bb210b4cfe17498b4d0
- Tests: CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target mise exec -- cargo test -p umpire-checker -p umpire-tests -p umpire-testing, CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target mise exec -- cargo test -p umpire-wasm --test dispatch, make model-check, make model-time, GATE_SKIPPED:unittest:green-receipt 42d96f4b - verification reused the current task's post-gate pass, GATE_SKIPPED:model-check:green-receipt 42d96f4b - verification reused the current task's post-gate pass, GATE_SKIPPED:model-time:green-receipt 42d96f4b - verification reused the current task's post-gate pass
- PRs: