---
satisfies: [R8, R9]
---
# fn-161-facets-action-owning-behaviour-areas-of.4 Wrap remaining machines into facets; switch facets-only and vocabulary rules on globally

## Description
Wrap every remaining declared machine into facets, then switch on the facets-only rule (R8) and the facet-vocabulary rule (R9) for all models. Most machines get a single facet, or the few their rules naturally fall into. The goal is to move existing sections, not redesign anything.

**Size:** M–L
**Files:** the remaining declared machines (survey 2026-10-10): worker `Polling` (`actors/worker/Worker.scala`); activity `ActivityRecord` (`system/Dispatch.scala`), `HeldDispatch` and `LostStartAnswer` (`system/DispatchRaces.scala`); `NexusProduct` (`nexus/product/Product.scala`); `NexusSystem` (nexus standalone and nexus workflow `system/System.scala`); `RejectAfterClose` (`nexus/workflow/system/ClosePolicy.scala`); `TrustingCaller` (`nexus/workflow/system/TrustingCaller.scala`); `TaskQueueProduct` and `TaskQueueSystem` (`foundations/taskqueue/{product,system}/*.scala`). Also their feature files (each declares its facet vocabulary), the lifter's allow-list from fn-161.1, and regenerated `model/ir/*`.
**Touches:** [model/temporal/actors/worker/*.scala, model/temporal/features/**/*.scala, model/temporal/foundations/**/*.scala, model/irgen/*.scala, model/ir/*, model/ir/*.lint.json]

### Approach
- **Re-survey** the declared `Machine` objects first; fn-160.3 and fn-159 may have moved or renamed some.
- **For each machine:** declare its feature's facet vocabulary in the feature file (for a foundation or actor package, its root file). Move its effects, rules and capabilities into facets: one facet when the rules form one area (the worker's polling, the task queue's queue behaviour), several when they clearly split (e.g. nexus scheduling, completion, cancellation). Apply ownership by action and the same-owner-across-levels rule between each product and its system.
- **Design and adversarial machines** (`ActivityRecord`, `HeldDispatch`, `LostStartAnswer`, `RejectAfterClose`, `TrustingCaller`) usually take a single facet named after the concern they model. Their derived variants stay derived.
- **Switch the rules on globally:** empty fn-161.1's allow-list so R8 and R9 hold for every declared machine. Update the README to drop any mention of machines without facets.
- **IR:** regenerate. Rows, outcomes, facts, Cases and Query answers must stay the same; differences limited to function bodies, hashes and source positions. Record them per IR file.

### Prerequisite
- fn-161.1 (the construct, checks and allow-list). Can run beside fn-161.2 and fn-161.3, which cover the standalone activity's main machines.

### Quick commands
```sh
make umpire-gen-model
make umpire-check-model
make umpire-check-lint
```

## Acceptance
- [ ] Every declared machine defines its rules, effects and capabilities only in facets named from its feature's vocabulary; the allow-list is empty and R8/R9 hold for all models.
- [ ] Each product and system pair owns shared actions by same-named facets.
- [ ] Regenerated IR keeps every row, outcome, fact, Case and Query answer; `make umpire-check-model` and `make umpire-check-lint` pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
