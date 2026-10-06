---
satisfies: [R10]
---
# fn-133-lean-typed-realizations.7 Close: line counts, docs, MILESTONES

## Description
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.
R10. Record the three realizations' line counts before the spec and after each part, with each part's projection proof linked. Check that `model/README.md`'s realization section shows the new forms (scopes, kit modules, typed objects, class-pattern rule). Remove fn-133 from `MILESTONES.md` and from fn-128's gate. Close the spec.

## Acceptance
- [ ] Line counts and proofs are in the done summary.
- [ ] README's realization section matches the code.
- [ ] `MILESTONES.md` updated and the spec closed.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
