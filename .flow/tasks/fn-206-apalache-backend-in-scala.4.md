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
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: