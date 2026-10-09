---
satisfies: [R7]
---
# fn-155-name-the-standalone-activitys-repeated.6 Regenerate, prove the mapping and close fn-155

## Description
Closes the spec: one regeneration, the projection proof, Go test and fixture updates, a docs check, gates and review (spec R7, Decision Context batch nature). It hands the mapping to fn-140.4/.6 and fn-129.3.

**Size:** M
**Files:** `model/ir/activity-standalone*.json`, `model/cases/**`, `model/irgen/testdata/lifts/expected/hints*.json`, affected Go tests, `model/README.md` if a cited example changed, `.flow/tmp/fn-155/mapping.md`; conductor owns `MILESTONES.md` updates
**Touches:** [model/ir/**, model/cases/**, model/irgen/testdata/lifts/expected/**, tools/umpire/**/*_test.go, model/README.md]

### Approach
- **Regenerate.** Run `make umpire-gen-model`, then `project.py` against the task 1 baseline. Every non-position difference must appear in `mapping.md`, either in the identity mapping or with a reason in the structural-review section. Re-dump the step tables and compare them with the baseline.
- **Fixtures.** Refresh the irgen hints fixtures through the gate's `--update` path (`model/check/Gate.scala:495-500`).
- **Go tests.** This task alone updates Go tests that read renamed effects, rule shapes or state-expression shapes: `lower/withholding_test.go:44` (reads `retryCompletes` as a `construct`) and `:47-50`, `export/quint_test.go:258` and `:1188-1191`, `export/open_test.go:154`, `lint/holes_test.go:73,163`, `interp/decisions_test.go:74,91`.
- **Gates.** Run the model gate, `make lint-model`, `make umpire-check-cases` and the Go tooling suite (`-tags test_dep -p 2 -timeout 30m`, under the shared flock). Also run the glossary gate on the new names.
- **Docs.** Check `model/README.md` (Recorded, `toProduct`, `deadlines`, and the reset-override example near the Retries section). Edit it only where a cited example changed.
- **Handoff.** Give `mapping.md` to fn-140.4/.6 and fn-129.3 as their re-anchor. Return milestone-row evidence to the conductor, who alone edits MILESTONES.md.

### Investigation targets
**Required:**
- `MILESTONES.md` — Verification instructions and Batches
- `.flow/tmp/fn-155/mapping.md`

### Acceptance
- [ ] Step tables equal the baseline; projection reports only positions, mapped identities and reasoned structural-review entries
- [ ] Model gate, model lint, `umpire-check-cases` and the Go tooling suite pass, or inherited batch 5 failures are shown unchanged from the baseline
- [ ] Independent review of the diff and mapping is recorded
- [ ] MILESTONES.md is updated and the mapping is linked for the downstream specs
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
