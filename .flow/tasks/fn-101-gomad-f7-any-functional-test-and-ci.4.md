---
satisfies: [R3, R4]
---
# fn-101-gomad-f7-any-functional-test-and-ci.4 Make a functional-test smoke check the required CI gate

## Description
Make the Temporal qualification a required CI smoke check (user decision 2026-09-28: CI runs only a smoke test on selected functional tests). Define a small named smoke selection (e.g. the frontend probe, user-timers, and a few F6 suites covering activities, signals, updates, chosen for coverage per minute) as its own manifest or a subset of the representative set; add a linux/amd64 workflow job (the macOS job moved to fn-105-gomad-follow-ups-deferred-scope.7 on 2026-09-29) that runs on pull requests touching tools/gomad3, tests, tests/testcore, go.mod, and the closure's server packages, requires unsupported == 0, failed == 0, infrastructure_errors == 0, and fits well inside 90 minutes. The full ./tests set is not run in CI; keep make gomad3-tests-qualification as the local gate and document it. The manifest-generator staleness check runs in CI's validate step on tests/** changes.
## Acceptance
- workflow defined and validated; linux counts asserted

## Done summary
The linux functional smoke gate exists and passes. `tools/gomad3integration/qualification/smoke.json` names four `./tests` suites chosen for coverage per minute from the linux metrics: user timers and the task poller, activities, updates, and child workflows (signals left out because the chasm suite keeps a recorded 1-in-28 replay residual that would make a required gate flaky). Each suite is copied verbatim from `temporal.json`; `TestSmokeSuitesMatchRepresentativeSuites` fails on drift. `make gomad3-smoke-qualification` runs it with pruning.

`.github/workflows/gomad3-smoke.yml` runs it on linux/amd64 for pull requests and main pushes touching Gomad, the functional tests, and the server packages their closure reaches (`chasm`, `client`, `common`, `components`, `schema`, `service`, `temporal`, `go.mod`), first checks that the generated `./tests` manifest is current, then asserts `supported == 4`, `unsupported == 0`, `failed == 0`, `infrastructure_errors == 0`, expectations met, and exact replay for every suite and seed. `gomad3.yml` calls it as a reusable workflow so a dispatch of Gomad v3 runs it too; the concurrency group includes the caller so direct and called runs do not cancel each other. The macOS job moved to fn-105 D7. Making the check required is a branch-protection setting, documented in the integration README.

Evidence: local darwin run 4/4 qualified in 9.5 min; fork run 36668156879 job functional-smoke-linux succeeded in 9.5 min with 4/4 qualified, both seeds replayed exactly.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 141492f4c, f2d5e2742
- Tests: make gomad3-smoke-qualification (darwin, 4/4 qualified), go test -tags test_dep,gomad3_integration ./tools/gomad3integration, fork run 36668156879 functional-smoke-linux: success
- PRs: