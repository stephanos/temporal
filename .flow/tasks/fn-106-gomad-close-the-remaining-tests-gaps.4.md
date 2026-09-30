---
satisfies: [R4]
---
# fn-106-gomad-close-the-remaining-tests-gaps.4 Size the testcore dedicated-cluster pool independently of GOMAXPROCS under gomad

## Description
Under the gomad tag, give the dedicated pool enough slots for tests that need two dedicated clusters, so TestNexusOTELSuite/TestOperation no longer waits on itself.

## Acceptance
- TestNexusOTELSuite/TestOperation runs under Gomad; skip removed if it qualifies


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
