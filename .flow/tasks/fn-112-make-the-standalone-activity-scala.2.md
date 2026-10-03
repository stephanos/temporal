---
satisfies: [R6, R7, R8, R16]
---
# fn-112-make-the-standalone-activity-scala.2 Add captured declaration names, evidence defaults and refinement reads

Touches: [model/umpire/**, model/lifter/Context.scala, model/lifter/Declarations.scala, model/lifter/Claims.scala, model/lifter/Realizations.scala, model/lifter/test/**, model/lifter/testdata/**, model/README.md]

## Description
Add the general declaration/default machinery that later feature migrations consume, without moving feature files yet.

**Size:** M
**Files:** model/umpire declaration, identity-scope and claim APIs; model/lifter Context/Declarations/Claims; focused lifter fixtures and README.

### Approach
- Teach the lifter to take names from declaring vals for the R7 declaration matrix while retaining explicit-name overloads only where a name intentionally differs. Keep computed Query names supported.
- Name capture is a default, not a rule that every Property or Query has a `val` (spec R7, Decision Context "Capabilities and laws"): a Property or Query built inside a `def` over a machine argument (`admissionQueries(m)`, the R4 shared-claim defs of task 4) keeps the one explicit-name form, and the lifter must not refuse or rename a declaration because no `val` declares it. fn-122 later generates claims with no `val`, named `<machine>.<law>`; do not add a lifter check that would block that.
- Add `DefinitionScope`, pinned once per former source owner, and route symbol-based IDs through its owner plus the captured val name. Cover actions, monitors, assumptions, channels and realizations; reject duplicate/nested/conflicting scopes. Do not add per-declaration ID strings.
- Add machine type inference, declared-start Scenario defaults, fact-to-same-name evidence defaults and refined-machine Property reads. For R8, `in` drops its `using Reads[S, P]` clause (`model/umpire/Claims.scala:147`): `Machine[S, O, F]` does not carry the refined state type, so no given can be derived from the declared refinement; the lifter's existing refusal of a Property and a Scenario of unrelated machines (`model/lifter/Claims.scala:123`) stays the check, and `Reads`/`Reads.through` leave the author surface. Record the choice in the done summary (the alternative, a type parameter on `Machine`, changes every declaration's shape).
- Detect duplicate captured names and invalid/missing refinement projections with located diagnostics.
- Exercise generated symbol names and local helper calls in positive and refusal fixtures; compare against task 1 after every fixture migration.
## Acceptance
- [ ] Fixtures cover captured names for machine, derived machine, composition, Property, Scenario, Query, Limits, timer, action, monitor, assumption, hole, channel and realization declarations, plus scoped legacy owners for every symbol-based ID kind.
- [ ] Family-as-given, inferred machine types, omitted Scenario starts, evidence exceptions and implicit refinement reads lift to the original IR meaning; `in` takes no `Reads` given and the lifter's unrelated-machine refusal has a fixture.
- [ ] A Property or Query declared inside a `def` over a machine argument with the explicit-name form lifts as today; one fixture proves the lifter requires no `val` for a Property or Query.
- [ ] Duplicate/ambiguous names and unavailable refinement reads are refused at their source.
- [ ] DefinitionScope fixtures reproduce the task-1 owner/name map exactly and refuse nested, duplicate and conflicting pins without a per-declaration escape hatch.
- [ ] Task-1 equivalence, focused lifter tests and lint-model pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
