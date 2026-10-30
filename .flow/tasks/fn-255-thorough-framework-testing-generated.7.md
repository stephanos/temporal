---
satisfies: [R6, R7]
---
# fn-255-thorough-framework-testing-generated.7 Tier wiring through the framework entry point, docs and budgets

Touches: [model/umpire/**, model/umpire/README.md, Makefile]

## Description
Wire every technique into its tier without slowing the inner loop (spec R6).

**Cross-spec gate:** after the committed closures of fn-254.10 (framework entry point in `model/umpire`), fn-254.6 (`model-quick`) and fn-254.12 (per-stage budgets).

**Size:** S
**Files:** framework entry point subcommands (per fn-254.9), `model/umpire/README.md`; the root `Makefile` only if fn-254.10 left a Temporal-side hook.
**Touches:** [model/umpire/**, model/umpire/README.md, Makefile]

### Approach
- Extend the existing native stage DAG and inherited cancellation, admission and work-count authority; add no independent scheduler or competing receipt producer.
- Every fn-255 step is its own named stage with entries in fn-254.12's budget table: work counts (cases, corpus inputs, TLC launches) enforced hard, wall time against its budget.
- `model-check`: fixed-seed native/interpreter checks with explicit TLC skip, plus stable deterministic corpus replay (each input once). Total fn-255 share at most 30 s warm on the fn-254 baseline host.
- `model-validate`: fn-255 share at most 120 s (bounded generator run, `-max_total_time` per fuzz target, Miri).
- Required-TLC generated cases, bounded timed fuzzing and Miri run in `model-validate`; longer campaigns are scheduled-only.
- Scheduled tier only (`scheduled` entry point, never part of `model-check`, `model-validate` or a spec-close requirement): full cargo-mutants/coverage campaigns, long fuzz and long generator runs. Each run has a wall-clock cap and reports partial results when it hits it.
- Re-measure: with fn-255 wired in, `model-check` and `model-validate` still meet fn-254 R6's targets; if not, cut fn-255 counts, never the targets.
- README: what each tier runs, how to reproduce a failure from its seed, and how to pin a finding as a fixture or corpus regression.
- Standalone: everything passes fn-254.11's control on `model/umpire` alone.

## Acceptance
- [ ] Tier wiring in place; `model-quick` unchanged; `model-check` additions measured within budget.
- [ ] A planted corpus regression and a planted fixed-seed disagreement each fail `model-check` with their name and seed.
- [ ] README documents tiers, reproduction and pinning; the fn-254.11 standalone control passes.
- [ ] Every fn-255 stage, including mutation/coverage smoke controls, has budget-table entries; a planted over-budget work count fails the stage.
- [ ] The .3 slow-input controls fail stable replay and timed fuzzing within the outer timeout, with stage, target and input diagnostics.
- [ ] Required-backend cases include a nonzero source-generated batch and strict complete-success comparison; matching refusals and same-state/verdict outcome/fact mutations fail. A Refuted source family uses individual per-claim TLC checks and counterexample replay through compiled/interpreted semantics; a single-backend-refutes control fails and shrinks. Scheduled partial-result controls demonstrate that full mutation/coverage campaigns are unnecessary for closure.
- [ ] Re-measured gates meet fn-254 R6's targets with fn-255 included, or counts are reduced until they do.
- [ ] No open-ended job (mutants, coverage, long fuzz or generator runs) is reachable from `model-quick`, `model-check` or `model-validate`.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
