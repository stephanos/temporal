---
satisfies: [R4]
---
# fn-106-gomad-close-the-remaining-tests-gaps.4 Size the testcore dedicated-cluster pool independently of GOMAXPROCS under gomad

## Description
Under the gomad tag, give the dedicated pool enough slots for tests that need two dedicated clusters, so TestNexusOTELSuite/TestOperation no longer waits on itself.

## Acceptance
- TestNexusOTELSuite/TestOperation runs under Gomad; skip removed if it qualifies


## Done summary
testcore already honors TEMPORAL_TEST_DEDICATED_CLUSTERS, and the ./tests generator gained a per-test environment override. With two dedicated clusters TestNexusOTELSuite/TestOperation runs instead of waiting on itself, but the suite is then not deterministic (seed 17 nondeterministic; a seed 11 replay differed in stderr and wrote no terminal frame). Classified: TestOperation stays skipped with that finding, which keeps the rest of the suite qualified. Along the way the Runner no longer reports a watchdog-killed or cancelled target as a choice-coverage runner failure.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: gomad qualify-set TestNexusOTELSuite with TEMPORAL_TEST_DEDICATED_CLUSTERS=2: seed 17 nondeterministic, seed 11 runner failure, go test ./runner -run ChoiceTraceObserved
- PRs: