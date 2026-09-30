---
satisfies: [R3, R4]
---
# fn-101-gomad-f7-any-functional-test-and-ci.4 Make a functional-test smoke check the required CI gate

## Description
Make the Temporal qualification a required CI smoke check (user decision 2026-09-28: CI runs only a smoke test on selected functional tests). Define a small named smoke selection (e.g. the frontend probe, user-timers, and a few F6 suites covering activities, signals, updates, chosen for coverage per minute) as its own manifest or a subset of the representative set; add a linux/amd64 workflow job (the macOS job moved to fn-105-gomad-follow-ups-deferred-scope.7 on 2026-09-29) that runs on pull requests touching tools/gomad3, tests, tests/testcore, go.mod, and the closure's server packages, requires unsupported == 0, failed == 0, infrastructure_errors == 0, and fits well inside 90 minutes. The full ./tests set is not run in CI; keep make gomad3-tests-qualification as the local gate and document it. The manifest-generator staleness check runs in CI's validate step on tests/** changes.
## Acceptance
- workflow defined and validated; linux counts asserted

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
