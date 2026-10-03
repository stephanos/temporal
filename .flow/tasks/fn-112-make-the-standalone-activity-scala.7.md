---
satisfies: [R2, R3, R4, R5]
---
# fn-112-make-the-standalone-activity-scala.7 Migrate queue compositions and shared properties without string keys

Touches: [model/temporal/standaloneactivity/System.scala, model/temporal/standaloneactivity/Claims.scala, model/lifter/test/**, model/lifter/testdata/**, model/ir/**, model/cases/**]

## Description
Apply typed composition/member/sync references and shared claims to dispatch queue and both composition families.

**Size:** M
**Files:** standaloneactivity System/Claims roots and focused feature/lifter tests.

### Approach
- Derive queue providers and composed machines with rebind/extend/withMember so each steps/sync set is declared once.
- Replace actionKeys, whenAction and fact/action key literals with typed own/synced/records selectors.
- Declare notAdmittedWhilePaused, atMostOneActive and terminalStays once and attach the one definition to machines and both composition families.
- Exercise ambiguous member/action spellings and sync qualification while retaining exact original composition keys and order.

## Acceptance
- [ ] No two feature machines share a steps list, each composed state type has one sync declaration set and both composition families derive by member replacement.
- [ ] actionKeys, whenAction and fact/action string keys are absent from the feature Model.
- [ ] Each of the three shared properties has one source definition and the original Property rows/answers.
- [ ] Task-1 full equivalence and focused feature/lifter tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
