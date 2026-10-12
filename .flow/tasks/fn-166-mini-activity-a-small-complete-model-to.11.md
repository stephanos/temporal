# fn-166-mini-activity-a-small-complete-model-to.11 Lifter suites scoped to what they lift: root-reachable admission, production fixtures retargeted to mini

## Description
Owner request, 2026-10-12: the lifter test suites should be fast and meaningful under the mini-only scope.

Found (conductor, from the fn-167.2 full `model/irgen` run: 264 tests, 149 failing, over 11 min wall):
- **Incidental real-feature coupling.** Fixture tests lift a small fixture of their own, but with `model-scala.jar` (every feature) on the classpath; the lifter's source admission checks every authored source it can see, not only what the lifted roots reach. One refusal in `features/activity/standalone` (e.g. `protocol/Dispatching.scala:99`, Query placement) fails dozens of unrelated construct tests (`Fixtures.declarations`, many reported at 0.0s), and every fixture lift re-admits the whole Model.
- **Deliberate production fixtures.** A few suites use the real standalone activity as their fixture: `ActivityHeartbeatFixtures`, the witness exact-composed selector catalog test, `HintSnapshotsSuite`, the standalone production compositions/realizations tests in `Fixtures`, `CapabilityPropertiesTest`.

Do:
1. Make lifter admission (and any whole-source scan the lift performs) cover only the sources reachable from the requested roots (or, if reachability is impractical, only the packages of the requested roots plus framework/shared code), so a fixture or the mini lift neither fails on nor pays for unrelated features. The production `FEATURES=activity_mini` lift must stay byte-identical; a full-feature lift must still admit everything it lifts.
2. Retarget each deliberate production-fixture test to `features/activity_mini` where the mini has the construct; where it does not, move the test to a sweep-held place (or mark it skipped with a reason naming the sweep), never deleting its assertion silently. Record the inventory.
3. Fix the remaining inherited `ConventionsFixtures` fixture drift from fn-161 (FacetSource/PhaseImports) if it is lifter-test-only work.
4. Measure: wall time and pass/fail counts of the full `model/irgen` suite before and after, and per-suite top costs. Record them in the done summary.

## Acceptance
- A fixture lift does not report refusals from sources its roots do not reach; a test proves it with a deliberately broken unrelated source.
- Mini selected lift byte-identical; mini steps-only equivalence passes.
- No test fails because of the real features; each deliberate production-fixture test is retargeted to mini or sweep-held with a recorded reason.
- Before/after full `model/irgen` wall time and counts recorded.


## Done summary
Lifter suites scoped to what they lift (owner request 2026-10-12). Commits ba4a7a34b5 (admission + ModelBuild/SweepHeld/SuiteCoverage), cc063e66f1 (suite retargets and fixture drift).

Requirement evidence
- Root-reachable admission (model/irgen/Admission.scala, Lift.scala, Order.scala): the order/structure/conventions lints read only sources declaring the roots, the sources they reference (terms and types), each with its folder and feature (a kind's form is its own unit), to a fixed point. `--ir` with no names, with every declared file named, or naming a file no feature declares, admits every source as before (full-feature lift unchanged). AdmissionSuite (4/4): a fixture lift does not report a deliberately broken unrelated source (testdata/admission/unrelated); a root that reads it is refused at its line; `--ir` of every file reports it.
- Mini selected lift byte-identical: HEAD lifter vs this lifter, same jar/classpath, activity-mini.json and .waivers.json identical (mini-cmp.log). FEATURES=activity_mini --check-equivalence mini-selected-baseline-00d22597d2 fn160/task3/steps.map --steps-only: equivalent, 16 tables, exit 0 (equivalence.log).
- No test fails because of the real features. Inventory, retargeted to the mini: lifts/Hints, HintRejects, Literals (expected hints/hintsRefused regenerated; fixture realizations identical modulo positions, only the target machine differs), conventions (Checks, Imports, Kind/Form/Framework imports, Qualified*), typedRealizations, hintsInvalid, features, deadlinePresets (Catalog over the mini's two presets: 4 classes, was 6), rejection-code table (MiniActivityRun), composition members (mini Queue), CapabilityNames mini instance test (live mini lift replaces a stale stored digest), Conventions ActivityRecord -> mini ActivityProtocol, MiniRealizationFixtures/BoundsTest on one shared mini lift. Sweep-held (SweepHeld, reported ignored with the feature named; assertions kept): DeadlinePresets "48 start classes" and "timeout retry realization"; Fixtures "script helpers/Temporal kit", "standalone typed members/syncs/replaces", "every real feature's rejection-code table", "WithTaskQueue capability root"; held sources lifts/Scripts.scala and ScriptRejects.scala (SweepHeld.sources; their 20 reject roots and expected rejects lines held with them). CapabilityPropertiesTest is not a model/irgen suite (untouched).
- Inherited fixture drift fixed (lifter-test-only): ConventionsFixtures (fn-161 imports retargeted, stale testdata .scala-build avoided by building copies, dotted-in lines 38/39), FacetsTest (fn-149 grouping; unknown-vocabulary file named for a design family so the vocabulary refusal is reached), BoundsTest (mini query positions, 90 functions), initOrder/sectionOrder/eligibleWitness/witness placement grouping (refusal lines kept; borrowed 128->129), taskqueue identity contract re-anchored (diff proved identity+order only), Retryable UpTo[2] identity, fn-159.10 defaults (heartbeat, presets) and rejects.txt digest, unsupported fixture package (fn-161 facets-only), stale .flow/tmp fn159 scoped jars replaced by the packaged Models.
- Measured (full `scala-cli test model/irgen --server=false`): before 972 s wall, 156 tests reported: 121 pass, 35 fail (Fixtures ~113 tests and DeadlinePresetsSuite collapsed into beforeAll compile failures), summed test time 886 s; top: MiniRealizationFixtures 133 s, FacetsTest 122 s, QualifiedNames 84 s, MiniLayoutFixtures 59 s, PropertyGroups 58 s. After: 755 s wall, 280 tests: 274 pass, 0 fail, 6 sweep-held ignored, summed 704 s; top: Fixtures 160 s, FacetsTest 136 s, BoundedIntegers 68 s, QualifiedNames 66 s, Witness 32 s. Remaining cost is per-test fixture packaging (scala-cli --server=false); diff-scoped selection is fn-166.13 (SuiteCoverage.mini lists the suites that read the mini).

Verification: irgen-before.log, irgen-after.log (exit 0), mini-cmp.log, equivalence.log, scalafmt --check on changed files (Fixtures/FixturePositions test files were unformatted at HEAD, left so), git diff --check.

Open obligations / deferred: the sweep restores the six held tests and two held sources (drop SweepHeld). Deferred gates: full model gate, independent review (REVIEW_MODE=none), default every-feature run. Speed: 22% less wall time while running 124 more tests; not the order-of-magnitude a diff-scoped check (fn-166.13) gives.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: ba4a7a34b5, cc063e66f1
- Tests: timeout 5400 mise exec -- scala-cli test model/irgen --server=false (before: exit 1, 121/156 pass, 972s; after: exit 0, 274 pass, 6 sweep-held, 0 fail, 755s), umpire.irgen.AdmissionSuite 4/4, HEAD vs fn-166.11 lifter --ir activity-mini: byte-identical, FEATURES=activity_mini scala-cli run model/check -- --check-equivalence mini-selected-baseline-00d22597d2 steps.map --steps-only: equivalent, scala-cli fmt --check changed files, git diff --check
- PRs: