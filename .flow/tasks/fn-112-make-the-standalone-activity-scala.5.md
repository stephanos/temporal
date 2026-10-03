---
satisfies: [R9, R14, R16]
---
# fn-112-make-the-standalone-activity-scala.5 Add captured typed action inputs and bounded counters

Touches: [model/umpire/Action.scala, model/umpire/Domain.scala, model/lifter/Declarations.scala, model/lifter/Expressions.scala, model/lifter/Types.scala, model/lifter/test/**, model/lifter/testdata/**]

## Description
Implement the settled input-token syntax and field-local finite counter bounds without changing state keys.

**Size:** M
**Files:** model/umpire Action/input/Finite APIs; model/lifter declaration/expression/finite lifting; focused fixtures.

### Approach
- Add captured `input[A]` tokens, Action declarations that consume them and `token := value` calls. Reorder supplied tokens into declaration order and default omissions to the first finite value.
- Refuse duplicate and foreign tokens, wrong value types and an input absent from the Action; compiler errors count where invalid Scala cannot produce TASTy.
- Add UpTo[N] finite enumeration and use a fixture to pin `0,1,2` order for UpTo[2]. Keep Active as its current enum.
- Preserve explicit `results("Delivery")` IR metadata while separating removal of the dead Scala enum.

## Acceptance
- [ ] Named-token calls support partial and reordered inputs while lifting exactly the original positional argument list and defaults.
- [ ] The positive/negative matrix covers repeated input types, duplicate/foreign tokens, wrong value types and missing inputs.
- [ ] UpTo[2] enumerates exactly 0,1,2 and Protocol-state fixture keys match the original baseline; Active is unchanged.
- [ ] No action input-name string literal or new IR field is required.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
