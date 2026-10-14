---
satisfies: [R6, R8, R9, R11, R12, R13]
---
# fn-141-shrink-the-ir-generator-one-description.12 Export claims: Properties, Scenarios, Queries and progress; the lifter's fold goes

## Description
The builder chains already build these values. Export them, and delete the lifter's evaluator of declaration expressions. This is where declaration-level Scala becomes free.

**Size:** M
**Files:** `model/umpire/{Claims,Machine,Assume,Syntax}.scala`, `model/irgen/{Claims,Lifting,Constants}.scala`, `model/temporal/**` (helper defs)
**Touches:** [model/umpire/**, model/irgen/**, model/temporal/**]
**Deferred:** see MILESTONES.md, Deferred, fn-141. Planned 2026-10-06 against the tree before the DSL batch. The files and names below are that tree's and carry no line numbers on purpose: when the spec is revived, re-read them and recount before starting, since the DSL batch and fn-140 rewrite them.

### Approach
- Properties take their predicate as a recorded function; Scenarios and Queries are data with references by name.
- A helper def that declares takes the naming context, and its function-valued parameter takes the recorded-function type. These are the two Model edits R13 allows. List every edited def.
- Unnamed Queries keep their derived name; two declarations that derive one ID are refused naming both positions.
- Query totals: the lifter computes the static count where the author gives none, and Go recomputes it. Follow the ledger on which side keeps the check.
- Add the lift fixture R13 names: Queries yielded from a comprehension, each with its own name. Add the reject fixture R9 names: a comprehension inside a function body.
- Delete `fold`, `declaring`, `bodyOf`, the claim bundles and the root dispatch that only served them.

### Investigation targets
**Required:**
- `model/irgen/Claims.scala` (`fold`, `declaring`, `bodyOf`, `query`, `scenario`, `register`, `captured`)
- `model/irgen/Lifting.scala` (`liftRoot`, `irFiles`) and `Capabilities.scala` (`countedTotal`)
- `model/umpire/Claims.scala` and `Machine.scala` (`Declares.property`, `scenario`)
- Helper defs that declare under `model/temporal` (shared laws, per-design Query lists)

## Acceptance
- [ ] Properties, Scenarios, Queries and progress claims in the IR come from the exporter; the lifter has no fold.
- [ ] The comprehension fixture lifts; the function-body comprehension is refused with today's text.
- [ ] Every edited Model def is listed and is one of the two allowed kinds.
- [ ] The ledger's claim rows have their outcomes.
- [ ] `make umpire-gen-model` leaves `model/ir`, `model/cases` and the lifter fixtures' expected IR byte-identical (R11); any difference stops the task until it is traced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
