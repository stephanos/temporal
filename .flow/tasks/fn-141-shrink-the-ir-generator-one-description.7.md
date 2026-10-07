---
satisfies: [R5]
---
# fn-141-shrink-the-ir-generator-one-description.7 Realization factories build complete values

## Description
The realization DSL discards part of what an author writes, because the lifter reads it from source. Make every factory keep everything, so the values can be exported.

**Size:** M
**Files:** `model/umpire/realize/{Realize,Typed,Scripts,Exploration}.scala`, `model/temporal/realize/*.scala`, new munit tests beside them
**Touches:** [model/umpire/realize/**, model/temporal/realize/**]
**Deferred:** see MILESTONES.md, Deferred, fn-141. Planned 2026-10-06 against the tree before the DSL batch. The files and names below are that tree's and carry no line numbers on purpose: when the spec is revived, re-read them and recount before starting, since the DSL batch and fn-140 rewrite them.

### Approach
- A request scope (`rpc { … }`, `readUntil { … }`, `withFields { … }`) keeps its assignments; today the block runs and its result is dropped.
- An evidence value keeps its operation and fields. `Condition.equal`, `greater`, `not` and `all` keep their operator; today two pairs build identical values.
- `Field(_.a.b)` records its path, and a factory that takes a protobuf type argument records the message name, as `Action.schema` already does through the companion.
- `family` and a command's id come from the naming context of task 5 if it has landed; otherwise leave the two fields and say so. Do not block on it.
- The lifter still reads realizations from source in this task. Nothing it reads may change.

### Investigation targets
**Required:**
- `model/umpire/realize/Scripts.scala` (the scope helpers), `Typed.scala` (`Field`, `Condition`), `Realize.scala` (`Evidence`, `Observed`, `Proto`, `Command`, `family`)
- `model/irgen/Realizations.scala`: for each construct, what it reads from the tree that the value lacks
- `model/temporal/realize/Kit.scala` and one realization, such as the standalone Nexus one

## Acceptance
- [ ] A test constructs one value per factory and reads every argument back; it fails naming the factory and the argument.
- [ ] No factory of `model/umpire/realize` or the kit drops an argument.
- [ ] `scala-cli test model/irgen` and `make umpire-check-model` pass; `make umpire-gen-model` leaves `model/ir`, `model/cases` and the lifter fixtures' expected IR byte-identical (R11); any difference stops the task until it is traced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
