---
satisfies: [R3, R4, R16]
---
# fn-141-shrink-the-ir-generator-one-description.4 Declaration-level sugar by definition; the lifter's sugar file and its lint exemption go

## Description
Part A, second half. The claim patterns and the sticky monitors are compositions of core builders, so their definitions can be followed like any helper def. After this task the lifter names no sugar that Part A can reach, and the syntax lint says so.

**Size:** M
**Files:** `model/irgen/{Syntax,Claims,Declarations}.scala`, `model/umpire/Syntax.scala`, `model/check/{SyntaxRule,Gate}.scala`, `model/check/test/SyntaxRule.test.scala`
**Touches:** [model/irgen/**, model/umpire/Syntax.scala, model/check/**]
**Deferred:** see MILESTONES.md, Deferred, fn-141. Planned 2026-10-06 against the tree before the DSL batch. The files and names below are that tree's and carry no line numbers on purpose: when the spec is revived, re-read them and recount before starting, since the DSL batch and fn-140 rewrite them.

### Approach
- Write `once(...).keeps(...)`, `never(...)`, `never(...).from(...)`, `stays(...)` and `stays(...).unless(...)` as bodies over `holds` and `holdsAcross`. Today they construct the Property record directly, which the lifter cannot follow.
- `sticky` and `stickyAcross` already call `monitor`; follow them.
- Names must not move: a predicate passed as a lambda lifts today as a function named after the word that takes it, and a `keeps` projection as a field path. Reproduce those names through the generic binding, so the expected IR does not change.
- Three forms evaluate at run time and cannot be followed as trees: named inputs (`start(x := v)`), the request-scope assignment and the script helpers. Leave their arms in place, list them in the done summary, and name the Part B task that removes each (task 8 for the realization forms, task 9 for named inputs).
- Extend the syntax lint: no lifter file names a sugar definition, apart from the arms just listed, each with the task that removes it. Delete the lifter's `Syntax.scala` if nothing is left in it.
- Record `model/irgen` source lines before and after Part A (R16).

### Investigation targets
**Required:**
- `model/irgen/Syntax.scala` (`patterned`, `pattern`, `stickyMonitor`, `keptPath`, `named`, `requestAssignment`)
- `model/irgen/Declarations.scala` (`stepFunction`, `monitorOf`) for the names a lambda lifts under
- `model/umpire/Syntax.scala` (`Once`, `Never`, `Stays`, `sticky`) and `model/umpire/Claims.scala`
- `model/check/SyntaxRule.scala` (`sugarNames`, the three rules) and its test

## Acceptance
- [ ] No lifter code names a claim pattern or a sticky monitor.
- [ ] The sugar fixture's pattern and monitor pairs lift to the same IR as before, function names included.
- [ ] The syntax lint fails on a sugar name in a lifter file, naming file, line and sugar; its test covers it.
- [ ] The remaining arms are listed with the task that removes each.
- [ ] `scala-cli test model/irgen`, `make lint-model` and `make umpire-check-model` pass; `make umpire-gen-model` leaves `model/ir`, `model/cases` and the lifter fixtures' expected IR byte-identical (R11); any difference stops the task until it is traced.
- [ ] Line counts before and after Part A are in the done summary.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
