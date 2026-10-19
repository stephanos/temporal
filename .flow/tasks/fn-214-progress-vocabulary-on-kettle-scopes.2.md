---
satisfies: [R2, R3]
---
# fn-214-progress-vocabulary-on-kettle-scopes.2 Bounded claims and named progress verdicts with hints

## Description
Implement R2 and R3 after .1's explicit-fairness representation is integrated.
This task owns numeric authoring bounds, structured progress results and the shared checking algorithm.

**Size:** M
**Files:** registered progress data, generated LeadsTo, checker progress/report/admission, focused fixtures.
**Touches:** [model/umpire/dsl/src/progress.rs, model/umpire/dsl/macros/src/machine.rs, model/umpire/dsl/macros/tests/fixtures/**, model/umpire/dsl/tests/**, model/umpire/checker/src/progress.rs, model/umpire/checker/src/report.rs, model/umpire/checker/src/refusal.rs, model/umpire/checker/src/explore.rs, model/umpire/checker/src/lib.rs, model/umpire/checker/tests/**]

### Approach

- Add numeric `.within(n)` to generated author-facing `LeadsTo` and propagate bound metadata through `ProgressEvaluation` and registered claims. Preserve legacy `Progress<M>::within(Scope)`; use a distinctly named numeric operation on registered claims for checker queries.
- Treat the bound as fixed claim metadata alongside the assumption. Refuse zero; retain Unknown for inconsistent reached metadata. Compile-time refusal is useful only when the macro can prove invalidity without rewriting authored statements/headers; otherwise use the checked runtime refusal.
- Fix explicit `.assuming` overrides on evaluated registrations: an explicit override replaces the evaluator's authored assumption, including an unrestricted override, and discards checks of that replaced metadata. A numeric override similarly replaces only the bound. Absence of an override retains the authored value. Preserve the original predicate callbacks and one evaluation per reached state.
- Reuse region construction, deterministic predecessor paths and SCC analysis in `checker/src/progress.rs`. Preserve the existing weak-fair closed-walk behavior and cut/hole refutation precedence. Compute finite longest-to-first-target lengths without enumerating paths; keep path reconstruction lazy.
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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
