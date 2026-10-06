---
satisfies: [R1, R20]
---
# fn-126-read-each-feature-top-to-bottom-one.11 Move standalone activity realizations into the System folder

## Description
Move model/temporal/features/standaloneactivity/Realization.scala to system/Realization.scala (or the corresponding activity/standalone path if fn-132 has landed). It realizes StandaloneActivity, HeldDispatch and LostStartAnswer, all System subjects. Update the package and explicit imports, root exports, feature headers, layout rules/templates and docs. Keep the file's existing realization structure until fn-133 introduces one typed object per realization; this task is a placement change. Do not create a Product realization file without a Product-level executable realization.

## Acceptance
- [ ] Standalone activity's executable realization file lives at system/Realization.scala; the obsolete root Realization.scala is absent.
- [ ] All three existing realizations remain available to the same exported IR files/Queries; imports, exports, headers, README and layout/structure checks reflect the new location.
- [ ] Fully qualified realization/script IDs, source paths and positions are accounted for in a rename ledger; regeneration preserves machine behavior, Query answers, existing Case execution and verdicts with the declared map applied.
- [ ] Focused layout/lifter tests and generated-artifact checks pass. Share the preceding task's applicable evidence and run the full required validation once at the fn-126 batch boundary after this move.
- [ ] fn-133's typed realization work uses this System-folder placement after fn-132 moves the enclosing feature.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
