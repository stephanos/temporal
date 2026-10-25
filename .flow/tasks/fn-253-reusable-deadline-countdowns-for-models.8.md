---
satisfies: [R5]
---
# fn-253-reusable-deadline-countdowns-for-models.8 Docs and full gates for countdowns

Touches: [model/README.md, model/umpire/README.md, model/DESIGN.md, model/temporal/feature/activity/README.md, MILESTONES.md]

## Description
Document how a Model declares its countdowns and deadlines, then run the full gates. The spec closes on this evidence.

**Size:** S
**Files:** `model/README.md`, `model/umpire/README.md` (bounded optional values section), `model/DESIGN.md`, activity README, `MILESTONES.md`.
**Touches:** [model/README.md, model/umpire/README.md, model/DESIGN.md, model/temporal/feature/activity/README.md, MILESTONES.md]

### Approach
- README: a short authoring example (countdown fields, `arm`/`tick`/`expired`, the laws), next to the closure-law paragraph.
- DESIGN: the two-layer split and the independence rule for refinement.
- Run `make model-check`, then `BASH_ENV=/dev/null UMPIRE_BACKENDS=require make model-validate` in the main checkout against the shared target. Run `cargo test --workspace --no-fail-fast` first to surface stale snapshots at once.
- Write the validation-cost review required by spec close.

### Acceptance
- [ ] README and DESIGN describe the abstraction with an example; the activity README matches the migrated Model.
- [ ] `make model-check` and the TLC-required `make model-validate` pass; goldens re-accepted after review.
- [ ] MILESTONES.md updated.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
