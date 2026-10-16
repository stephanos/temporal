---
satisfies: [R6]
---
# fn-188-framework-only-source-iteration.3 Finish native DSL public-client verification and full batch review

## Description
Complete the native DSL batch gate required by parent R6. This is the next admitted priority-0 task, independent of the blocked broad exporter loop in .2. Native implementation checkpoints are already committed; do not reopen them or add new DSL features merely to fill this task.

**Size:** M
**Files:** model/framework/README.md; model/framework/dsl/**; model/framework/realize/**; model/framework/samples/**; model/framework/check/lint/**; MILESTONES.md; existing native public-client recipes/evidence under .flow/tmp.
**Touches:** [model/framework/dsl/**, model/framework/realize/**, model/framework/samples/**, model/framework/check/lint/**, model/framework/README.md, MILESTONES.md]

### Approach

- Reuse the native-only commands in model/framework/README.md and existing public-client recipes; no parallel verifier, task-specific Scalafix or new proof machinery.
- Retain the committed-source strict compile, 452 native tests, 29 lint tests, three boundary tests, actual boundary scan and 125-file format checks. Rerun only evidence invalidated by a source correction or otherwise demonstrated stale/incomplete.
- Finish and audit fresh separate-library public clients for record/scalar/channel inputs, domains/guards, Query configuration, durability, illustrations and Inbox ownership. Preserve every failed attempt. Verify intended refusal locations/reasons, not merely nonzero exit status. Historical partial continuations are not a complete passing recipe.
- Run one formal review of the entire native batch from baseline 437016b7d148b87cccee78c926af92119dec7fb4, not only the last fixture fix. Resolve demonstrated native findings with scoped checks and rerun affected final gates. Parallel read-only review/scouting is allowed; source integration and compiler/cache writes retain one coordinated lane.
- Record exact evidence and remaining proof debt in the task receipt and update MILESTONES.md in place. Native admission and literal callback examples do not prove reachability, full Query answers, checked-int64 interpretation, classification or graph equivalence.

### Boundaries

No model/temporal reads, restoration, compilation or migration, including disabled source contents. No broad make check-framework, model/project.scala, Temporal Model/API jars, capture/exporter repair, schema, Go, backend or regeneration. Existing generic IR dependencies do not authorize new IR work. Finishing this task does not release .2 or other held integration tasks without an explicit later scheduling decision.

## Acceptance
- [ ] Final-source native compile/test/lint/boundary/format evidence is complete and audited, with invalidated checks rerun after any correction.
- [ ] Independent public-client positives and intended runtime/compiler refusals pass using fresh native framework artifacts; no partial or stale continuation earns passing credit.
- [ ] Formal full-native-batch implementation review has a passing verdict and surviving native findings are resolved with verification.
- [ ] Receipt and milestone overview preserve deferred IR/Go/backend/Temporal obligations; native completion neither closes their requirements nor releases their holds automatically.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
