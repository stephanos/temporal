---
satisfies: [R5, R11, R12]
---
# fn-123-declare-faults-as-the-environments.5 Budgets: budgetedBy, the four table rules in Go, LostStartAnswer bound to lossAvailable

## Description
Budgets (R5, R12): `f.budgetedBy(_.b)` on a binding, recorded in the IR, the four table rules of spec Part C checked in Go, the durable-budget rule in the lifter, and `LostStartAnswer` converted to name its existing `lossAvailable` field. Nothing in its step functions changes.

**Size:** M
**Files:** `model/framework/Machine.scala` or `Syntax.scala` (`budgetedBy`), `model/irgen/Declarations.scala` (record and refusals), `proto/internal/temporal/server/api/umpire/v1/machine.proto` (budget field on `StepBinding`) + regenerated Go, `tools/umpire/internal/engine/table.go` or `tools/umpire/ir/validate.go` (the four rules, + tests), `model/temporal/features/activity/standalone/system/DispatchRaces.scala`, lifter fixtures, regenerated `model/ir/**`, `model/cases/**`
**Touches:** [model/framework/**, model/irgen/**, proto/internal/temporal/server/api/umpire/v1/**, api/umpire/**, tools/umpire/internal/engine/**, tools/umpire/ir/**, model/temporal/features/activity/standalone/system/DispatchRaces.scala, model/ir/**, model/cases/**]
**Order:** After task 4 (one regeneration at a time, and the durable-budget rule needs task 2). Re-read `DispatchRaces.scala` first: fn-139.5 moved `LostStartAnswer` onto the shared `Outcome` and fn-140 rewrote its Query as a witness after these targets were verified.

### Approach
- DSL and lifter: `budgetedBy` takes a field selector of the machine's state type. Refuse at the line: a field that is neither Boolean nor an `Int` with catalog `0..n`, a second budget on one binding, and a budget field not classified `durable` on a machine with a crash (cover this with a fixture, since no Model exercises it).
- IR: the budget field's name on the binding (next free number in `StepBinding`, `machine.proto:76-83`), default-empty.
- Go: check the four rules over every catalog state (spec Part C): no row raises the field, each row of `f` lowers it by one (`true` to `false` for a Boolean), no other action's row lowers it, and `f` has no row from an exhausted state. Refuse at the binding with the first offending state and action. One test per rule, plus one showing a guard and an `end` that read the field pass. Keep state identity injective (memory `finite-search-requires-injective-2026-09-30`).
- `LostStartAnswer` (`DispatchRaces.scala:122-166`): bind `ackLoss.budgetedBy(_.lossAvailable)`. `history.dispatch` reads the field and never lowers it, and both `loseResponse` arms set it `false`, so the rules hold. If Go refuses the binding, stop for the owner rather than edit a step function (R12).
- Regenerate. The diff for `LostStartAnswer` holds only the budget field, and `StandaloneActivityPins` passes unchanged.

### Investigation targets
**Required** (read before coding):
- `model/temporal/features/activity/standalone/system/DispatchRaces.scala:101-166` - state, choices, the machine
- `tools/umpire/internal/engine/table.go:160, 369-430` - rows and table checks
- `proto/internal/temporal/server/api/umpire/v1/machine.proto:76-83` - `StepBinding`
- `model/temporal/features/activity/standalone/StandaloneActivityPins.test.scala:7-25` - the pin that must not change

**Optional** (reference as needed):
- `tools/umpire/ir/validate.go:733` - step admission
- `model/ir/activity-standalone-race.json` - where `LostStartAnswer` lifts

## Acceptance
- [ ] `budgetedBy` lifts onto the binding. Refusal fixtures cover the wrong type, a second budget and a non-durable budget field on a crashing machine.
- [ ] Go refuses each broken budget rule at the binding with the offending state and action (one test each). A guard and an `end` that read the field pass.
- [ ] `LostStartAnswer` binds `ackLoss.budgetedBy(_.lossAvailable)`. Its state type, step functions, `end`, tables, state keys, catalogs, Definition IDs and Case bytes are unchanged, and `StandaloneActivityPins` passes.
- [ ] The regenerated diff holds only the budget metadata and positions.
- [ ] `make umpire-check-model`, `make umpire-check-cases` and the Go tests of the changed packages pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
