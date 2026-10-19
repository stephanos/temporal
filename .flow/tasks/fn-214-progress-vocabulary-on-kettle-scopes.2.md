---
satisfies: [R2, R3]
---
# fn-214-progress-vocabulary-on-kettle-scopes.2 Bounded claims and named progress verdicts with hints

## Description
Implement R2 and R3 after .1's explicit-fairness representation is integrated.
This task owns numeric authoring bounds, structured progress results and the shared checking algorithm.

**Size:** M
**Files:** registered progress data, generated LeadsTo, checker progress/report/admission, focused fixtures.
**Touches:** [model/umpire/dsl/src/progress.rs, model/umpire/dsl/macros/src/machine.rs, model/umpire/dsl/macros/tests/fixtures/**, model/umpire/dsl/macros/tests/ui/**, model/umpire/dsl/tests/**, model/umpire/checker/src/progress.rs, model/umpire/checker/src/report.rs, model/umpire/checker/src/refusal.rs, model/umpire/checker/src/explore.rs, model/umpire/checker/src/lib.rs, model/umpire/checker/tests/**, model/umpire/wasm/src/refusal.rs, model/umpire/wasm/tests/dispatch.rs, model/umpire/proto/README.md]

### Approach

- Add numeric `.within(n)` to generated author-facing `LeadsTo` and propagate bound metadata through `ProgressEvaluation` and registered claims. Preserve legacy `Progress<M>::within(Scope)`; use a distinctly named numeric operation on registered claims for checker queries.
- Treat the bound as fixed claim metadata alongside the assumption. Refuse zero; retain Unknown for inconsistent reached metadata. Compile-time refusal is useful only when the macro can prove invalidity without rewriting authored statements/headers; otherwise use the checked runtime refusal.
- Fix explicit `.assuming` overrides on evaluated registrations: an explicit override replaces the evaluator's authored assumption, including an unrestricted override, and discards checks of that replaced metadata. A numeric override similarly replaces only the bound. Absence of an override retains the authored value. Preserve the original predicate callbacks and one evaluation per reached state.
- Reuse region construction, deterministic predecessor paths and SCC analysis in `checker/src/progress.rs`. Preserve the existing weak-fair closed-walk behavior and cut/hole refutation precedence. Compute finite longest-to-first-target lengths without enumerating paths; keep path reconstruction lazy.
- Extend the independent reference oracle for bounded questions where practical, retaining one property test and independently derived expected behavior. Exercise a known refutation alongside a separate reached hole so Unknown cannot erase that proof; fixed assumption/bound inconsistency still makes the whole claim Unknown.
- Carry added checker refusals through the existing WASM mapping and generated protobuf roundtrip controls. Preserve distinct diagnostics and use an existing wire category only when it describes the new error accurately. No new method, schema dependency or ABI change is needed for this task.
- Expose `ClaimResult::verdict: Option<ProgressVerdict>` and `hint: Option<String>`. The progress enum has the five kinds in R3. Ordinary safety/effect results have no temporal classification, and keep their original Outcome. Do not invent Cycle/Deadlock for a failed invariant or a caught predicate defect.
- Preserve the default `Report::spell` bytes. Supplementary query fixtures in .3 render the new fields explicitly using checker-owned output filenames.
- Pin equality boundaries, zero/finite/unbounded longest values, escaped fair cycles, fair actions disabled intermittently, competing verdict classes, cuts, holes and bounded witnesses. The first target at exactly n transitions satisfies a bound of n; the incoming transition into a trigger is outside the obligation length.

### Investigation targets

**Required:**
- `model/umpire/dsl/src/progress.rs` - registered callbacks and legacy scope builder.
- `model/umpire/dsl/macros/src/machine.rs:505` - combined registration and LeadsTo helpers.
- `model/umpire/checker/src/progress.rs` - shared region/SCC checking and predicate recorder.
- `model/umpire/checker/src/report.rs:68` - shared result and canonical report renderer.
- `model/archived/framework/checker/NativeCheck.scala:630` - reference progress ordering and longest calculation, read-only.

**Optional:**
- `.flow/memory/bug/performance/build-counterexample-paths-only-for-2026-10-18.md` - lazy witness construction.

### Quick commands

From `model/`, `mise exec -- cargo test -p umpire-dsl -p umpire-dsl-macros -p umpire-checker -p umpire-tests`.
From the repository root, run `make model-check` and record one `make model-time` output under the 10 s / 30 s soft policy.
## Acceptance
- [ ] Numeric LeadsTo bounds reach registration and checking; zero refuses, a target at exactly n passes, and fixed metadata mismatch remains Unknown. Legacy scope-taking Progress builder and both canonical samples remain byte-identical at source and golden level.
- [ ] Synthetic fixture goldens distinguish Deadlock, Cycle, MissedDeadline, Verified with longest 0/finite/None, and Unknown. Refuting logical results carry deterministic paths, cycle_start where applicable and the specified hint text.
- [ ] Controls pin verdict precedence, explicit escaping fairness with and without bounds, intermittent enabledness, concrete refutations despite later cuts, and no success from reached holes or incomplete exploration.
- [ ] Evaluated claims honor replacement assumptions without losing original trigger/target callbacks or evaluating a progress body more than once per reached state. Safety and predicate/effect defects retain their existing Outcome and do not acquire a fabricated temporal verdict.
- [ ] Canonical report rendering stays byte-identical. Focused tests, `make model-check` and the recorded `make model-time` pass.
## Done summary
Model authors can bound generated LeadsTo claims with within(n), and checker clients can replace registered bounds with within_steps(n) while retaining legacy within(scope). Progress results now expose named verdicts, longest first-target distances and deterministic refutation hints without changing Report::spell.

R2/R3 coverage lives in checker fixture progress_bounds, the generated DSL machine fixture, the existing recorder controls and the single independent brute_force_oracle property test. They cover zero refusal, equality at n transitions, exclusion of the incoming trigger transition, metadata inconsistency, independent assumption/bound replacement, callback counts, verdict precedence, finite/zero/unbounded longest values, explicit escaping fairness, intermittent enabledness, cuts, separate reached holes and caught defects. The finite-distance calculation uses reverse graph traversal and constructs paths only for refutations. The native generated-protobuf roundtrip maps zero bounds to malformed request code 2 with a claim-specific diagnostic.

Both complete sample_kettle and sample_kettle_trait inventories and every file's Git blob hash match base 4047dfaf6135f8fdc77769379dfd6dad1c12fe54. The generated fixture registries under /home/agent/.cache/codex-build/cargo-target point to this checkout. Macro snapshot changes pin the new helper and bound propagation; existing canonical report, state, row and path goldens remain unchanged.

baseline: green. The focused suite, make model-check and make model-time exited 0 before implementation. Logs are .flow/tmp/fn214-2-baseline-focused.log, fn214-2-baseline-model-check.log and fn214-2-baseline-model-time.log. Baseline timing was 2.638 s inner and 3.717 s full. The baseline timing command briefly overlapped the preceding gate's tail; all post-change builds and gates were serialized after both child authors stopped.

RED evidence is retained in .flow/tmp/fn214-2-red-dsl.log, fn214-2-red-checker-api.log and fn214-2-red-wasm-api.log. They observed missing numeric authoring/registration APIs, result fields and the refusal variant before production changes. Initial fixture setup and snapshot mismatches remain in fn214-2-first-focused.log, fn214-2-red-golden.log and fn214-2-snapshots.log. The first full gate's function-size lint failure remains in fn214-2-model-check-initial.log; the final helpers pass the unchanged lint policy.

The final focused suite and full make model-check passed. The full gate includes formatting, denied-warning clippy/docs, native fixtures, stock WASI guest/core checks and CGO-disabled wazero Go-host parity. Post-change make model-time exited 0 at 2.066 s inner and 5.152 s full against the 10 s / 30 s soft targets. Logs are .flow/tmp/fn214-2-final-focused.log, fn214-2-model-check.log, fn214-2-final-fixture.log and fn214-2-model-time.log.

GATE_SKIPPED:unittest:green-receipt ad9aec6f - verification reused the current task's post-gate pass
GATE_SKIPPED:model-check:green-receipt ad9aec6f - verification reused the current task's post-gate pass
GATE_SKIPPED:model-time:green-receipt ad9aec6f - verification reused the current task's post-gate pass

stage: impl-review - ran [2026-10-19T16:21:58Z..2026-10-19T16:24:17Z] (codex:gpt-6.1-sol:high, SHIP on all three axes, zero findings)

Review receipt: /tmp/impl-review-receipt-8f37faba39e2-fn-214-progress-vocabulary-on-kettle-scopes.2.json. The writer and reviewers use the same model family under explicit project routing. Read-only Cargo build-lock failures limited the reviewers to static inspection; the worker's actual gates passed. The original foreground review session 55100 and PID 3967613 completed with exit 0 despite the transient fanout_abandoned journal. The contracts draw's formatting prevented merge-plan parsing, so the documented merged-file repair finalized the same three SHIP results without redispatch. Clean first-pass review required no memory capture.

Tier: session (jev-unavailable(no_key)); explicit project routing preserved

Two child agents implemented the checker algorithm and authored the disjoint bound fixture/oracle extension. Both returned before final staging, verification and review. Tracker inactive; planSync.enabled=false. Task .3 retains the registered query, Kettle question fixtures and emitter documentation. No follow-up work is required for R2/R3.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: ad9aec6ffc23dbc02c343a46f3f98f8b6507132e
- Tests: mise exec -- cargo test -p umpire-dsl -p umpire-dsl-macros -p umpire-checker -p umpire-tests, make model-check, make model-time, bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn214-2-protected.sh, GATE_SKIPPED:unittest:green-receipt ad9aec6f - verification reused the current task’s post-gate pass, GATE_SKIPPED:model-check:green-receipt ad9aec6f - verification reused the current task’s post-gate pass, GATE_SKIPPED:model-time:green-receipt ad9aec6f - verification reused the current task’s post-gate pass
- PRs: