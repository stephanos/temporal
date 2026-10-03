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
- Add `DefinitionScope`, pinned once per former source owner, and route symbol-based IDs through its owner plus the captured val name. Cover actions, monitors, assumptions, channels and realizations; reject duplicate/nested/conflicting scopes. Do not add per-declaration ID strings.
- Add machine type inference, declared-start Scenario defaults, fact-to-same-name evidence defaults and refined-machine Property reads.
- Detect duplicate captured names and invalid/missing refinement projections with located diagnostics.
- Exercise generated symbol names and local helper calls in positive and refusal fixtures; compare against task 1 after every fixture migration.

## Acceptance
- [ ] Fixtures cover captured names for machine, derived machine, composition, Property, Scenario, Query, Limits, timer, action, monitor, assumption, hole, channel and realization declarations, plus scoped legacy owners for every symbol-based ID kind.
- [ ] Family-as-given, inferred machine types, omitted Scenario starts, evidence exceptions and implicit refinement reads lift to the original IR meaning.
- [ ] Duplicate/ambiguous names and unavailable refinement reads are refused at their source.
- [ ] DefinitionScope fixtures reproduce the task-1 owner/name map exactly and refuse nested, duplicate and conflicting pins without a per-declaration escape hatch.
- [ ] Task-1 equivalence, focused lifter tests and lint-model pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
