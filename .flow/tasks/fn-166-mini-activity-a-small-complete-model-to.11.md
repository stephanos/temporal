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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
