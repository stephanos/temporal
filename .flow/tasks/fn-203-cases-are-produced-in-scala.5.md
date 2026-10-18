---
satisfies: [R5]
---
# fn-203-cases-are-produced-in-scala.5 Temporal adapter: descriptors, settlements, waits, rejection codes, exhaustive evidence

## Description
Port the Go behavior, settlement, wait and rejection-code admission into model/temporal: evidence messages and field paths from ScalaPB descriptors, activity attempt and reset settlement, external settlements, read waits from behavior hints, rejection codes and exhaustive evidence, each refusal naming the realization and the rule. Compiles only when Temporal sources are re-enabled.

**Files:** model/temporal/realize/*.scala (re-enabled), model/temporal/testpilot/*.scala, tests.

**Quick commands:** `mise exec -- scala-cli test --server=false model/temporal/testpilot --require-tests`

## Acceptance
- Every Go refusal in validate_behavior/activity_* has a Scala counterpart test.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
