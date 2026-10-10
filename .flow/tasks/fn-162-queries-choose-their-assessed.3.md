---
satisfies: [R4, R5]
---
# fn-162-queries-choose-their-assessed.3 Migrate the standalone activity off claim-container subjects

## Description
Migrate the standalone activity off claim-container subjects. Move each Property, declared once, to the facet or machine that states its promise. Run each Query over the base machine, with its assessment set, and point each realization at the base machine and its Queries.

**Size:** L
**Files:** the activity's system subjects (`Completion`, `RetryFailures`, `Cancellation`, `Pausing`, `Timeouts`, `Heartbeat*`, `ByID*`, `Reset*`, `DispatchEligibility`, … per fn-162.1's list), their realizations, `ActivitySubjectRegression` and other tests that pin subjects, `Standalone.scala` exports (or fn-160's collection), regenerated `model/ir/*`, `model/cases/*`, and the pinned fixture trees
**Touches:** [model/temporal/features/activity/standalone/**/*.scala, model/ir/*, model/cases/*, tests/testcore/testpilot/testdata/generated/*, tools/canary/casebinding/testdata/*]

### Approach
- **For each claim container** in fn-162.1's list: move its Properties to their facet (fn-161), or to `ActivitySystem`'s properties if facets haven't landed. Collapse duplicates into one declaration, e.g. one `completes` reused by the completion, heartbeat, pausing and dispatch Queries. Move its Queries next to them, unchanged in scenario, limits, total and expectation, with an assessment set equal to today's subject Properties for that Case. Delete the derivation.
- **Realizations:** each realizes `ActivitySystem` (or the behaviour-changing derivation it needs) and runs its Queries. Keep their controller, worker and evidence bodies unchanged. If fn-160.4 has landed, rename them after the Query or facet they run, following its naming rule.
- **Tests:** replace the subject regression test with one pinning that each Case's assessment set is exactly what it was before (same Properties by predicate), and update tests that name subjects.
- **Regenerate** the IR and Cases. Every Query answer and total and every Case's expected outcome stays the same. Differences are limited to moved Property and Query ids, the assessment sets and derived Case names. List them, and update the pinned fixture trees.
- **Sequencing:** ideally before fn-160.3 splits these subjects into one file each, so files aren't moved twice. If fn-160.3 lands first, this task deletes those files.

### Quick commands
```sh
make umpire-gen-model
make umpire-check-model
make umpire-check-lint
make umpire-check-fixtures
```

## Acceptance
- [ ] The standalone activity has no claim-container derivations; each Property is declared once; each Query carries the assessment set its Case had before.
- [ ] Every Query answer and total and every Case expected outcome is unchanged; IR and Case differences are limited to moved ids, assessment sets and derived names, as listed in the done summary.
- [ ] `make umpire-check-model`, `make umpire-check-lint` and `make umpire-check-fixtures` pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
