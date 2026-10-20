---
satisfies: [R1, R2, R3]
---
# fn-225-gomad-bounded-harness-result-retention.1 Bound harness outputs and replace completed results safely

## Description
Implement the approved bounded retention design at the existing harness output lifecycle. Stage complete runs, safely publish latest results, release disposable superseded outputs, and retain bounded failures plus pinned inputs. Keep authenticated receipt paths valid, preserve concurrent runs and unrelated files, and document defaults. Implement and test in this isolated checkout; integrate after fn-189 qualification.

## Acceptance
Repeated real harness runs have bounded storage; failed publication preserves prior results; active and pinned inputs survive; concurrency and path/symlink safety are covered by meaningful filesystem tests. Affected tests and lint pass, and independent review confirms receipt/replay compatibility.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
