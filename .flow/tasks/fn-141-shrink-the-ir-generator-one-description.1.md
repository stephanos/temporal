---
satisfies: [R1, R16]
---
# fn-141-shrink-the-ir-generator-one-description.1 Move the order lint, the structure lint and the marker checks out of model/irgen

## Description
Part C. Three files of the lifter (1,479 lines on 2026-10-06) emit no IR. Give them a home of their own and a gate step of their own, with no change to what they refuse.

**Size:** M
**Files:** `model/irgen/{Order,Structure,Markers,Lift,Lifting,Context}.scala`, a new lint module beside `model/check`, `model/check/Gate.scala`, `model/irgen/test/Fixtures.test.scala`, the lint fixtures under `model/irgen/testdata/{initOrder,sectionOrder,layout,grouping,retiredNames,objectForms}`
**Touches:** [model/irgen/**, model/check/**, model/lint/**, model/README.md]
**Deferred:** see MILESTONES.md, Deferred, fn-141. Planned 2026-10-06 against the tree before the DSL batch. The files and names below are that tree's and carry no line numbers on purpose: when the spec is revived, re-read them and recount before starting, since the DSL batch and fn-140 rewrite them.

### Approach
- The two lints read whole typed trees through the lifter's `Index`. Keep one TASTy read: either the lint module owns the index and the lifter reuses it, or both build on a small shared reader. Record the choice.
- The marker checks read what a file lifted (machines, Queries, fault actions) and the marker traits, which the IR does not carry. Keep them after the lift and before anything is written, behind an interface that takes the index and the lifted Models.
- A lint refusal still stops the lift: `Lifter.inspect` lifts nothing when the order lint refuses.
- Move the lint fixtures and their tests with the code. Refusal texts and positions do not change.
- Record `model/irgen` source lines before and after (R16).

### Investigation targets
**Required:**
- `model/irgen/Lift.scala` (`Lifter.inspect`, `liftRoots`: where the lints and marker checks run)
- `model/irgen/Order.scala`, `Structure.scala`, `Markers.scala` and their use of `Index` and `Context`
- `model/check/Gate.scala` (the steps that compile and run the lifter) and `model/irgen/project.scala`
- `model/irgen/test/Fixtures.test.scala` (the lint fixture tests)

## Acceptance
- [ ] `model/irgen` holds no lint: `Order.scala`, `Structure.scala` and `Markers.scala` are gone from it.
- [ ] Every lint and marker reject fixture refuses with the same text at the same position from its new home.
- [ ] A source a lint refuses writes no IR file; the gate reports the refusal as before.
- [ ] `make umpire-check-model` and `make lint-model` pass; `make umpire-gen-model` leaves `model/ir`, `model/cases` and the lifter fixtures' expected IR byte-identical (R11); any difference stops the task until it is traced.
- [ ] Line counts before and after are in the done summary.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
