# fn-142-split-modeltemporalshared-into.1 Move shared/ to foundations/ and actors/, Bounds.scala to the root; regenerate; ID-mapping proof; docs
# fn-142-rename-modeltemporalshared-to.1 Move shared/ to foundations/ and Bounds.scala to the root; regenerate; paths-only proof; docs

## Description
**Size:** S
**Touches:** [model/temporal/**, model/irgen/**, model/ir/**, model/cases/**, model/check/**, tools/umpire/**, model/README.md, model/SEMANTICS.md, .plans/UMPIRE_MODULES.md, MILESTONES.md]

`git mv` `shared/taskqueue` to `foundations/taskqueue`, `shared/worker` to `actors/worker`, `Client.scala` to `actors/client/Client.scala` and `shared/Bounds.scala` to `Bounds.scala`; update packages and imports, the structure lint's folder classes and their fixtures, and the docs. Regenerate once, as one batch, and prove the IR and Case diff is only the package mapping of Definition IDs (`temporal.shared.taskqueue` → `temporal.foundations.taskqueue`, `temporal.shared.worker` → `temporal.actors.worker`), source paths and positions (R3).
## Acceptance
- [ ] R1 and R2: the folders, packages and imports follow the new layout, and no `temporal.shared` or `shared/` Model path remains outside dated history.
- [ ] R3: one regeneration batch; Definition IDs change exactly by the package mapping; Query answers and receipts are unchanged apart from those renamed IDs; a diff check shows only that mapping, source paths and positions.
- [ ] The structure lint, the module map and the docs name the new layout; `make lint-model`, `make umpire-check-model` and the Go tooling suite pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
