---
satisfies: [R3, R4]
---
# fn-166-mini-activity-a-small-complete-model-to.8 Scala mini realization and typed live witness authoring

## Description
Implements the remaining Scala source portion of R3 and prepares R4. Case lowering and live execution remain in held task 4.

**Size:** M
**Files:** mini realization `MiniActivityRun.scala`, mini-local `Kit.scala`, query declarations/header in `MiniActivity.scala` and the existing mini levels; focused mini source/lift fixtures.
**Touches:** [model/temporal/features/activity_mini/**, model/irgen/test/Fixtures.test.scala, model/irgen/testdata/layout/mini/**]

### Approach
- Subtract the existing standalone realization from `model/temporal/features/activity/standalone/realization/StandaloneActivityRun.scala:47`. Bind the committed mini Protocol and its facts; do not import the sibling standalone kit or feature.
- Reuse `Realizes(machine, runs = ...)` and ordered sections from `model/temporal/realize/Objects.scala:27`, shared typed evidence modules and `deadlines` from `model/temporal/realize/Modules.scala:124`. Concrete start presets retain the non-timeout maxAttempts input.
- Declare evidence for every mini fact, a deadlines use, one satisfied live witness and one reasoned inconclusive live witness. Keep the feature header's hard-instance inventory current.
- Prove discovery, typed bindings, evidence coverage, selected Queries and expectation metadata through isolated Scala compile/lift fixtures. Reuse `model/irgen/Claims.scala:651` and `model/irgen/Lifting.scala:201`; no new consumer or schema is required.
- Keep scratch outputs ignored. No checked-in artifact publication, Case lowering, Go changes, pinning or live run is authorized. Source evidence cannot establish executable evidence coverage or a live verdict.

### Verification
Run mini-focused compile/lift and focused tests for changed source/lifter fixtures with --server=false. Assert the inventory independently from declarations and include wrong-owner/missing-evidence/expectation controls. Broad validation and review remain at the owner-approved batch-end gate.

## Acceptance
- [ ] The mini realization compiles and lifts with the expected machine, bindings, Query membership and evidence for every fact.
- [ ] Typed deadlines and satisfied/reasoned-inconclusive live expectations execute in focused positive/refusal fixtures.
- [ ] The header lists the source hard instances; no sibling feature or protected artifact changes. R3 transport and R4 live proof remain explicitly open in held tasks.


## Done summary
The mini activity now has a Scala realization of its Protocol with typed controller calls, a worker script, seven evidence kinds and deadline presets retaining the one/two-attempt policy. Its two selected Queries author a satisfied expectation and an inconclusive expectation with an explicit evidence limitation; the feature header names these source hard instances.

baseline: green. The pre-edit selected mini/shared compile exited 0 in `baseline.log`. The persisted base is `1443467f5750dd7082757690de28f1403658b603`.

Focused verification
- `MiniRealizationFixtures` executed four tests and exited 0 in `realization-final.log`. Its independently specified inventory checks the realization ID and machine, exactly seven fact/evidence IDs, typed describe reads and run-event sources, all eight start classes, the concrete retry-policy and timeout fields, performed actions and derived delivery/timer steps, dispatch-hold ordering, exact selected Query membership and all expectedRun fields. Mutations refuse missing evidence, a wrong machine and a wrong expectation reason. Actual compiled/lifted controls refuse a foreign Query owner and a blank inconclusive explanation without writing output.
- `MiniProtocolTest` executed two tests against the final selected mini/shared jar and exited 0 in `native-isolated.log`.
- `MiniLayoutFixtures` executed two tests and exited 0 in `layout-final.log`. All three existing layouts and the production-substitute refusal remain intact; no fixture assertion or production inventory was widened.
- `git diff --check` and the staged diff check passed. Logs, `scoped-model.jar`, its classpath and the final scratch lift `lift/activity-mini.json` are retained under `.flow/tmp/fn166/task8/`.

Red and inconclusive observations
- `red-intended.log` failed on the absent realization before production edits. The first test attempt (`red.log`) failed compilation on an unused assertion result; the first implementation attempt (`green-first.log`) failed on an unused import.
- Refusal-fixture setup failures remain in `controls-first.log`, `controls-second.log` and `fixtures-final.log`. These identify the missing ScalaPB dependency, compiler-object root spelling and the requirement to derive through an explicit operation. The final actual controls pass.
- `hold-order-red.log` failed for the intended source-order defect, finding zero start bindings before the dispatch hold where eight were required. Moving the hold after the start bindings restored all four focused tests.
- `native-final.log` is INCONCLUSIVE for the intended mini native surface. Directory-root test compilation pulled in the shared capability test's excluded standalone/Nexus imports and exited 1 before any test ran. The corrected exact test-file invocation against the final selected jar executes both mini tests; no real feature or shared test was changed to obtain that result.

The lifted expectation reason `REASON_EXPLANATIONS_DISAGREE` is authored metadata, not an observed assessment. The source explains that first-delivery and final-status evidence does not establish the second delivery or retry backoff, so the attempt-count claim can remain ambiguous. Only `protocolCompleted` and `retryCompleted` belong to this realization; every other Query retains its existing model-only standing. These source declarations establish neither executable evidence coverage nor a live verdict.

R3 consumer/artifact obligations and R4 Case lowering/live execution remain open in held task 4; the canonical loop and full-set closure remain held in task 5. The known linked-protobuf Make prerequisite was not retried or regenerated. Checked-in mini IR/Cases, real-feature source/artifacts, lint companions, acceptance policy, schemas, Go/runtime code, table pins and the batch1d baseline were untouched. Lint was not run under the owner override. The conductor captures the new mini-only baseline later.

stage: impl-review - skipped(policy: owner-approved Scala batch-end)
Broad validation and independent review remain deferred to the owner-approved Scala batch-end gate. `gate classify` returned FULL on unrelated `IDEAS.md`; no docs-only skip, full-suite green receipt or SHIP is claimed.

Tier: session (jev-unavailable(no_key)); project implementer routing retained

Checkpoint `f2b5e91730` includes the root-acknowledged fn166 spec JSON, task4/5/8/9/10 MD+JSON and fn167.7 MD+JSON passengers. All externally modified Spec MD files, IDEAS, MILESTONES and fn168 files were preserved and excluded from catchall staging. The conductor retains milestone and metadata-only artifact ownership.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: f2b5e91730360cf2157916b9f7a63ccd53619916
- Tests: mise exec -- scala-cli compile --server=false --suppress-outdated-dependency-warning model/project.scala model/framework model/temporal/Bounds.scala model/temporal/foundations model/temporal/actors model/temporal/capabilities model/temporal/realize model/temporal/features/activity/Activity.scala model/temporal/features/activity_mini, timeout 600 mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/irgen --test-only umpire.irgen.MiniRealizationFixtures, timeout 600 mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/project.scala model/temporal/features/activity_mini/protocol/Protocol.test.scala --jar .flow/tmp/fn166/task8/scoped-model.jar --test-only temporal.features.activity.mini.protocol.MiniProtocolTest, timeout 600 mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/irgen --test-only umpire.irgen.MiniLayoutFixtures, git diff --check, RED: absent realization before implementation (red-intended.log); dispatch-hold ordering before repair (hold-order-red.log), FAILED SETUP: red.log, green-first.log, controls-first.log, controls-second.log, fixtures-final.log; corrected before final focused gates, INCONCLUSIVE: directory-root native test compilation selected shared excluded-feature imports before tests ran (native-final.log); corrected isolated two-test run in native-isolated.log
- PRs: