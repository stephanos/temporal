---
satisfies: [R3]
---
# fn-103-gomad-configurable-virtual-clock-tick.2 Measure the tick on the tie-excluded suites and retire their exclusions

## Description
Run the suites whose failures are demonstrated timestamp ties (the named tie exclusions in tests.generator.json, e.g. TestListWorkflow_OrQuery's suite, NexusOTEL TestWorkerOperation, the describe same-instant skip) with the tick on; switch those that qualify to the tick in the manifest and remove their exclusions; record in the milestone doc. Do not run the full ./tests set.
## Acceptance
- report lists exclusions removed; manifest updated; validate green

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
