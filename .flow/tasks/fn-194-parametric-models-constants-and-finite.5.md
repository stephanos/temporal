---
satisfies: [R8]
---
# fn-194-parametric-models-constants-and-finite.5 Simplify TwoPhaseCommit predicates to read Model vars directly

## Description
**Owner approval (2026-10-18):** After the proposal "Those predicates can live on Model and read its vars directly; tests can evaluate them with Model.at(state)(...). That would let us remove Transaction and current", the owner said "do that".

Move TwoPhaseCommit's state-reading predicates onto its Model, where they read rmStates, tmState, tmPrepared and messages directly. Remove the redundant Transaction record and current adapter. Adapt transitions, claims and tests to the direct predicates; tests evaluate model predicates on supplied states through Model.at. Keep unrelated collection helpers unless removal is required for this approved change.

**Size:** S

**Touches:** `model/framework/samples/TwoPhaseCommit.scala`, `model/framework/samples/TwoPhaseCommit.test.scala`

Any additional live caller needs exact scope authorization before editing. Do not rewrite historical source snapshots or upstream comparison files.

Use test-first refactoring: preserve the existing behavior baseline, add a failing control for direct instance predicates, then make the minimal source change. Retain every existing test purpose and expected refusal. Preserve selected finite manager domains, starts, guards, outcomes/facts, end condition, weak fairness, deadline3/longest3/depth10 in the default world, singleton-world behavior and the EagerCommit negative counterexample. Remove test-side record reconstruction too rather than move the adapter under another name. No DSL/API redesign, class-only migration or broader sample polish belongs here.

The measurement worker fn-196.3 owns a historical already-running JVM. Do not compile or change sample production bytes until the conductor confirms that command terminal and transfers the lane; no timeout-driven restart or overlap. Preserve its untracked design-prefix files and all other user work. After this task is verified and committed, the conductor returns the new baseline to fn-196.3 and it rebuilds its measurements.

Run own focused pre-edit TwoPC baseline, regression RED/GREEN, full affected native/lint/source classification and fresh sample/realization source and separate-library clients under unchanged strict flags. Use fn1962 final recipes and actual gate classify/check/receipt outcomes; no held Make/Temporal/Go route. Format before final exact-byte tests. All222 disabled Temporal sources, broad GuideSourcesTest, actual-Go sample bodies, IR/export/lifting/schema/generated work remain held. Scope-only staging, current umpire branch, review phase-batch deferred, plan-sync false, tracker inactive, parent open. No publishing/history rewrite. Handovers under `.flow/tmp/fn1945/{summary.md,evidence.json,final-commands.md}` and all commands terminal before return.
## Acceptance
- [ ] Transaction and current are removed from the active TwoPC sample and tests; state predicates read Model vars directly.
- [ ] Tests use Model.at on concrete states and preserve all previous assertion purposes, refusals and negative-control evidence.
- [ ] Selected domains, graph behavior, fairness/progress bounds and default/alternate-world mathematical answers are unchanged.
- [ ] Focused red/green plus full affected native/lint and source/separate-library sample realization verification pass with all holds preserved.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
