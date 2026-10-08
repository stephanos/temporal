---
satisfies: [R5]
---
# fn-129-activity-coverage.5 Close: new Cases, live run, MILESTONES

## Description
**Batch:** deferred Model batch (see MILESTONES.md, Deferred, fn-128). Runs after that batch's single regeneration and full gates; its live run is shared by fn-128.6 and fn-129.5.
R5. List every new Query and Case from tasks 1-4 with the requirement it covers. Run each new live Case once more together (only the known ShutdownWorker-race INCONCLUSIVEs allowed). Remove fn-129 from `MILESTONES.md`. Close the spec.

## Acceptance
- [ ] Every requirement has at least one live Query, listed with its Case.
- [ ] The live run is recorded with its log path.
- [ ] `MILESTONES.md` updated and the spec closed.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
