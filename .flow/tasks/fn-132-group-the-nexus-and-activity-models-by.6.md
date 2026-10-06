---
satisfies: [R3]
---
# fn-132-group-the-nexus-and-activity-models-by.6 General activity declarations in activity/Activity.scala

## Description
**Size:** M
**Touches:** [model/temporal/features/activity/**, model/irgen/**, model/ir/**, model/cases/**, tools/umpire/**, common/testing/testpilot/**, tools/canary/**, tests/*.go, tests/testcore/testpilot/**, Makefile, model/README.md, .plans/UMPIRE_MODULES.md, MILESTONES.md]

**Required investigation:** current `activity/standalone` header/product/System/Realization; task 4 binding decision; exact projection helper/type ledger; scoped action/entity typing and lifting tests. Task 2 must precede this extraction; task 5 is logically independent but cannot run concurrently while both regenerate shared artifacts.

Part C. Move into `features/activity/Activity.scala`: `Timeout`, `TimeoutType`, `AttemptResult`, the worker's `poll` and `respond` on an activity (with task 4's entity binding), `timers` and `deadline`. The standalone form reads them from there.

`ActivityProduct`, `ActivitySystem`, `Control`, `client` and the `activity` entity stay in `standalone/`; the worker actions move as R3 requires, with task 4's proved binding. Include System-owned `TimeoutType` in the exact type identity map. Keep `product.{Phase,State,Fact}` and `system.{Phase,State,Fact}` in their levels, and the System realizations under `standalone/system/`.

Independent of task 5.

## Acceptance
- [ ] `features/activity/Activity.scala` declares every listed type, worker action, timer and deadline; no duplicate remains in `standalone/`. Types-only extraction does not discharge R3.
- [ ] A before/after projection differs only in the paths of moved declarations.
- [ ] The spec's Verification gates pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
