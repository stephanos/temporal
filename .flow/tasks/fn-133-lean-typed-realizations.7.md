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
Closed the typed-realization migration with the README forms, projection proofs, and milestone cleanup. Realization line counts were 333/514/101 at baseline; recorded batch checkpoints were 296/357/88, 308/358/88, 303/305/83, and 303/305/83; the final activity-system/Nexus-workflow/Nexus-standalone counts are 310/305/83. The generated IR and Cases, lifter checks, Go consumers, and live runs provide the linked projection proof for each part.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 9b248a6019, 1a5ad6b702, 96a7dd3ba9, a58929556b, f275ba75ed, 46008b3322
- Tests: make umpire-gen-model, make lint-model, make umpire-check-cases umpire-check-fixtures canary-check-case umpire-check-lint, mise exec -- go test -json ./tools/umpire/..., make umpire-check-live-tests
- PRs: