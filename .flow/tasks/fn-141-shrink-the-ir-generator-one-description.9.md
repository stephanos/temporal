---
satisfies: [R4, R6, R7, R11, R12]
---
# fn-141-shrink-the-ir-generator-one-description.9 Export the signature: actions, inputs, entities, observations, channels, assumptions, holes, Limits

## Description
The declarations everything else refers to. An action gets its name at run time, which ends the empty run-time name and the `codeOf` workaround.

**Size:** M
**Files:** `model/umpire/{Action,Channel,Assume,Domain,Claims,Syntax}.scala`, `model/irgen/{Declarations,Claims,Constants,Syntax}.scala`
**Touches:** [model/umpire/**, model/irgen/**, model/temporal/**]
**Deferred:** see MILESTONES.md, Deferred, fn-141. Planned 2026-10-06 against the tree before the DSL batch. The files and names below are that tree's and carry no line numbers on purpose: when the spec is revived, re-read them and recount before starting, since the DSL batch and fn-140 rewrite them.

### Approach
- Each of these records its name, Definition ID and position through the naming context. An action declared with `action(actor)`, `timer` or `internal` is named after its `val`.
- `Rules.on` names the action from the action itself; `codeOf` and `writtenAction` go, and `on` need not be `inline` for that reason.
- Action classes, including named inputs (`start(x := v)`), export from the constructed class. Remove the named-input arm task 4 left.
- Uniqueness rules (action, channel, assumption and hole names; an input token used twice) follow the ledger: enforced once, with positions.
- Aliases (`val a = other.on(e)`) are plain values now; the alias-following code goes.

### Investigation targets
**Required:**
- `model/irgen/Declarations.scala` (`declaredAction`, `actionOf`, `inputToken`, `channelOf`, `assumptionOf`, `holeOf`, `distinctName`)
- `model/irgen/Claims.scala` (`classOf`, `limitsOf`) and `Syntax.scala` (`named`)
- `model/umpire/Action.scala` (`ActionDecl`: the fields the lifter fills in today) and `Syntax.scala` (`Rules.on`, `block`)

## Acceptance
- [ ] Actions, inputs, channels, assumptions, holes and Limits in the IR come from the exporter.
- [ ] An action's run-time name equals its IR name; the overlap message names it without `codeOf`.
- [ ] The ledger's rows for these kinds have their outcomes; kept refusals name the author's file and line.
- [ ] Model edits, if any, are of the two kinds R13 allows.
- [ ] `make umpire-gen-model` leaves `model/ir`, `model/cases` and the lifter fixtures' expected IR byte-identical (R11); any difference stops the task until it is traced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
