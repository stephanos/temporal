---
satisfies: [R5, R6, R7]
---
# fn-135-effect-and-is-blocks-for-effects-and.3 Lift effect { } blocks with their statement refusals

## Description
Lift `effect { }` section vals to the step function the method form gives, and add the statement refusals the spec lists. Split from .2 because rule-effect lowering and the statement walker are a separate path from predicates.

**Size:** M
**Files:** `model/irgen/Declarations.scala`, `model/irgen/Syntax.scala`, `model/irgen/Expressions.scala`, `model/irgen/testdata/lifts/Blocks.scala` (extend) and a new `BlockRejects.scala` with `expected/` output, `model/irgen/test/Fixtures.test.scala`
**Touches:** [model/irgen/*.scala, model/irgen/testdata/lifts/**, model/irgen/test/Fixtures.test.scala]
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- Rule-effect lowering (`Declarations.scala` ~620-660) today requires `lambda(r.effect)` whose body is a call of a `def` ("the effect of a rule is a call of a def of effects"). Add the `val` member registered by .2's mechanism; leave the `def` branch as is. `givesNoEmpty` (`:693`) does not apply to a block, which never yields no step.
- Walk the block's statements (unwrapped with `contextBody`) once: setter calls, recognized by the fixed setter shape from .1 (any other shape is refused, naming the accessor), become updates of one copy of `s` (`E.Copy` over `E.Var("s")`), emitted in field-declaration order with each field's type as the expected type, as `copyOf` does (`Expressions.scala:689-700`); `record` calls append facts in call order; a lone `reject(o)` becomes `Step(o, Var s, [])`. Otherwise build `Step(ok, copy, facts)` with the `Ok` outcome resolved the way `enter` resolves it (`outcomeOf(ok, "enter")`, `irgen/Syntax.scala:19-60`). An assign-only or empty block lifts exactly as `enter(s.copy(…))` / `enter(s)` does.
- Refuse, with `fail(tree, msg)` naming the statement and position: any statement other than an assignment, `record` or `reject` (a local `val`, a bare expression such as `phase == started`, any other call); an assignment, `record` or `reject` inside `if`/`match`/a loop (`If`/`Match` lift generically today at `Expressions.scala:358-371`, so the walker must look before delegating); a rejecting block with any other statement; a field assigned twice; a field read after its assignment; a nested block.
- Extend the two-field equivalence fixture with effect vals beside their `def` twins (assign+record, partial assignment, assignments written out of field order, two `record`s, assign-only, empty, reject) bound in rules; assert identical IR but for names and positions.
- Placement lint: classify `effect { }` vals as step functions (`Order.scala:276-305`), so the rule that step functions belong in `effects` applies to them.

### Investigation targets
**Required** (read before coding):
- `model/irgen/Declarations.scala:615-700` — rule-effect lowering, `givesNoEmpty`
- `model/irgen/Syntax.scala:19-60` — `sugarCall`, `sugar`, how `enter`/`reject` lift
- `model/irgen/Expressions.scala:340-375` — generic `If`/`Match`/`While`/`Block` lifting
- the .2 fixture `model/irgen/testdata/lifts/Blocks.scala`

**Optional** (reference as needed):
- `model/irgen/testdata/lifts/Rejects.scala`, `expected/rejects.txt`

### Acceptance
- [ ] Effect vals lift to IR identical to their `def` twins but for names and positions, for every case listed in Approach
- [ ] One refusal fixture per refusal in the spec's Edge Cases (including any-other-statement and a malformed setter), each message naming the statement and position
- [ ] The placement lint classifies `effect { }` vals
- [ ] Every existing lifter fixture passes; no change to any checked-in IR (R7) is checked at the batch's regeneration
- [ ] `make lint-model` passes

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
