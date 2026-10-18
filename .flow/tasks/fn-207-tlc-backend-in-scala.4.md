---
satisfies: [R4]
---
# fn-207-tlc-backend-in-scala.4 Agreement: reachable set, invariant verdicts, counterexample replay

## Description
Decode the dumped states through the lifted types with the shared value encoder (add the TLA+ spelling), compare as a set with the native graph's states, compare invariant verdicts per Property and monitor with the native verdicts, replay counterexamples through `Checker.replay`; receipts transition, property, monitor, coverage, witness-rejected, not-run.

**Files:** model/framework/backends/tlc/Agreement.scala, the shared encoder (fn-203.1 rows), tests.

**Quick commands:** as above

## Acceptance
- Reachable sets equal on all six samples; a planted extra or missing state is a difference.
- Refused replay is witness-rejected before any difference.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
