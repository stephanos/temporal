# fn-142-rename-modeltemporalshared-to.1 Move shared/ to foundations/ and Bounds.scala to the root; regenerate; paths-only proof; docs

## Description
**Size:** S
**Touches:** [model/temporal/**, model/irgen/**, model/ir/**, model/cases/**, model/check/**, tools/umpire/**, model/README.md, model/SEMANTICS.md, .plans/UMPIRE_MODULES.md, MILESTONES.md]

`git mv` `shared/taskqueue` to `foundations/taskqueue`, `shared/worker` to `actors/worker`, `Client.scala` to `actors/client/Client.scala` and `shared/Bounds.scala` to `Bounds.scala`; update packages and imports, the structure lint's folder classes and their fixtures, and the docs. Regenerate once and prove the IR and Case diff is paths and positions only (R3).
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
