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
- Declare notAdmittedWhilePaused, atMostOneActive and terminalStays once, as the parameterized top-level defs task 4 shaped (spec R4): each takes the model and the state-dependent parts as function parameters (`paused`, `running`, `terminal`), never a closure over `AdmissionState`, and is written with the claim patterns (`never(…).from(…)`, `never(…)`, `once(…).keeps(…)`). Attach the one definition to the admission machines, passing the record's named status sets, and to both composition families, passing named predicates over the member projection (`_.activity`); a lambda literal is not accepted as a def argument (task 4's refusal), so the compositions' predicates are named defs beside their vocabulary. Each instance keeps its frozen name through the explicit-name form; the `*.any.*` Queries keep their computed spelling. These three defs are the first law bodies `fn-122-capabilities-and-their-laws` generalizes; they move under `temporal/laws` there, not here.
- Exercise ambiguous member/action spellings and sync qualification while retaining exact original composition keys and order.
## Acceptance
- [ ] No two feature machines share a steps list, each composed state type has one sync declaration set and both composition families derive by member replacement.
- [ ] actionKeys, whenAction and fact/action string keys are absent from the feature Model.
- [ ] Each of the three shared properties has one source definition, a parameterized top-level def written with the claim patterns, and produces the original Property rows and answers on the admission machines and both composition families; no closure over `AdmissionState` and no lambda-literal def argument remains.
- [ ] Task-1 full equivalence and focused feature/lifter tests pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
