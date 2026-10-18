---
satisfies: [R5, R6]
---
# fn-207-tlc-backend-in-scala.5 Liveness verdicts agree with native progress; checkedBy(Checker.tlc) selection

## Description
Compare TLC liveness and deadlock verdicts with the native progress verdicts on the samples; undecided within the ceiling is resource-limit; route claims with `checkedBy(Checker.tlc)` and their `confirm` list through the opt-in backend stage; `Checker.umpire` stays the default.

**Files:** model/framework/backends/tlc/*.scala, model/framework/check/FrameworkLoop.scala, tests.

**Quick commands:** `UMPIRE_BACKENDS=require make check-framework`

## Acceptance
- Progress receipts agree on the samples, including deadlock controls.
- Unrunnable backend is unsupported with the reason.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
