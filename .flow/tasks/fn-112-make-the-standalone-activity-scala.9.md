---
satisfies: [R12, R13, R16]
---
# fn-112-make-the-standalone-activity-scala.9 Extract the shared Temporal realization kit and script helpers

Touches: [model/umpire/realize/**, model/lifter/Realizations.scala, model/lifter/test/**, model/lifter/testdata/**, model/temporal/realize/**, model/temporal/standaloneactivity/Realization.scala, model/temporal/nexuscaller/**, model/README.md, model/SEMANTICS.md, .plans/UMPIRE_MODULES.md, model/ir/**, model/cases/**]

## Description
Create the shared Temporal-specific kit and rewrite activity scripts through value references and typed Item-mode helpers.

**Size:** M
**Files:** model/temporal/realize/**, standaloneactivity Realization.scala, nexuscaller integration, umpire.realize helpers and lifter Realizations.scala/tests.

### Approach
- Move only the identical roles, environment bindings and correlation window shared by activity and Nexus into temporal.realize; parameterize feature-specific values.
- Add general script/perform/onPath/always helpers that lower to existing Item fields and preserve order, optional modes and declaration/reference IDs. Consult fn-118's inventory/interface decision so the shared kit exposes the typed method/read-condition seam future hints need, without adding hint fields, derived waits or Program changes here.
- Rewrite standalone activity realization with typed fn117 API selections, fact values and declaration references; migrate Nexus only enough to consume the shared kit.
- Add positive/refusal fixtures for unknown references and invalid helper combinations, keeping fn118 behavior hints out.

## Acceptance
- [ ] Shared roles, bindings and window are written once, documented and consumed by activity and Nexus with unchanged lifted values.
- [ ] Standalone Realization.scala contains no direct Item constructor, fact-name string or same-realization ID/reference duplication.
- [ ] Script order/modes, realization IDs, evidence catalogs and all Cases equal task 1.
- [ ] The helper interface accommodates fn-118's inventoried typed API and read-condition use, while this task introduces no hint-driven wait or Case Program delta.
- [ ] Focused lifter/model/Nexus tests and lint-model pass; no behavior hint or Go SDK work is introduced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
