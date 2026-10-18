---
satisfies: [R4]
---
# fn-206-apalache-backend-in-scala.4 Bounded verdicts, counterexample replay and receipts

## Description
Emit one check per monitor at the native product depth plus one; decode an ITF trace through the lifted types to a `ReplayScript`; replay through `Checker.replay`; receipts agreed, witness-rejected or disagreement as contracted.

**Files:** model/framework/backends/apalache/*.scala, tests.

**Quick commands:** as above

## Acceptance
- The known two-step counterexamples are found and confirmed on the monitored sample controls.
- Undecodable trace is an error of the call.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
