---
satisfies: [R7]
---
# fn-246-realization-as-a-checked-refinement-a.6 DESIGN.md 'Realization as a checked refinement', README authoring paragraph, counts and model-time; the workflow Model named as the re-measurement

## Description
After .3, .4 and .5. Write the DESIGN.md section: the five abstractions, the protocol Model and refinement semantics (starts, preserved edges, stutters, gaps, violations), strategy derivation, the shared/feature layout, per-feature and shared line counts before and after against the twice-the-Model target, and `make model-time`. Update `model/README.md` with the authoring paragraph (registry attributes, entity, read, refines) and the crate map for `temporal/shared`. Name the workflow Model as the re-measurement of the ratio. MILESTONES remains root-owned. Full gates: `make model-check`, `UMPIRE_BACKENDS=require make model-check`, `make model-time`, `make umpire-check-model-slice`, `make umpire-check-model-gaps`.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
