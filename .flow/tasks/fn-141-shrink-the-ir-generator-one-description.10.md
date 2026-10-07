---
satisfies: [R6, R8, R11, R12]
---
# fn-141-shrink-the-ir-generator-one-description.10 Export machines: header, rules, monitors, refinement and derivations

## Description
A machine's table of rules already exists at run time. Export it, and ask the lifter only for the functions: guards, effects, `end`, the refinement map, evidence and monitor functions.

**Size:** M
**Files:** `model/umpire/{Machine,Syntax,Monitor,Refine}.scala`, `model/irgen/{Declarations,Expressions,Context}.scala`
**Touches:** [model/umpire/**, model/irgen/**, model/temporal/**]
**Deferred:** see MILESTONES.md, Deferred, fn-141. Planned 2026-10-06 against the tree before the DSL batch. The files and names below are that tree's and carry no line numbers on purpose: when the spec is revived, re-read them and recount before starting, since the DSL batch and fn-140 rewrite them.

### Approach
- A rule holds its action or class, its heading, its guard and its effect. Guards and effects are recorded functions. The exporter lowers the rules of one action to the step function the lifter's `lowered` builds today, with the same function names.
- The sections the machine does not read at run time (`states`, `effects`, `monitors`, `refinement`) need a run-time handle. Decide how the machine finds them without reflection by name, and record it.
- Derivations (`restrict`, `rebind`, `extend`, `refining`, `assuming`, `unmonitored`) export from the built machine; the lifter's `Derivation` reader goes.
- Default evidence, the inferred entity and `unobservable` follow today's results exactly.
- This is the largest step by IR volume. Split it if it does not fit one iteration: header and rules first, then monitors, refinement and derivations.

### Investigation targets
**Required:**
- `model/irgen/Declarations.scala` (`objectMachine`, `ruleSteps`, `cases`, `lowered`, `derivedMachine`, `evidenceDefaults`, `monitorOf`)
- `model/umpire/Machine.scala` (`Rule`, `Bound`, `stepFunction`, `table`, `Built`) and `Syntax.scala` (`Rules`, `Case`, `Firing`)
- `model/umpire/Refine.scala` (`declaredBy`: the reflection used today)

## Acceptance
- [ ] Machines in the IR come from the exporter; the lifter's machine, rule and derivation readers are deleted.
- [ ] Rule-lowered step functions keep their names and bodies.
- [ ] The ledger's machine rows have their outcomes; the rules' overlap check still runs at construction.
- [ ] `make umpire-gen-model` leaves `model/ir`, `model/cases` and the lifter fixtures' expected IR byte-identical (R11); any difference stops the task until it is traced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
