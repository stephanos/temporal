---
satisfies: [R22]
---
# fn-105-gomad-follow-ups-deferred-scope.22 D22: fix parallel Nexus outcome endpoint collisions

## Description
Required fix approved on 2026-09-30. The start/cancel outcome tests in TestNexusApiTestSuiteWithLegacyErrorPaths and TestNexusApiTestSuiteWithTemporalFailures reuse testcase endpoint names between parallel ByNamespaceAndTaskQueue and ByEndpoint subtests on a shared cluster. Assign independent endpoint identities per subtest while preserving both dispatch paths, outcome assertions, and parallel execution. The fix applies to ordinary tests and does not change production endpoint uniqueness behavior.

## Acceptance
- Independent outcome subtests register distinct endpoint identities, and handler/request endpoint assertions remain consistent with each subtest's registered identity.
- Preserve start/cancel operations, both dispatch paths, both error-handling variants, existing outcome assertions, and parallel execution; no Gomad-only source rewrite or serialization workaround.
- Demonstrate the collision with a retained regression reproducer or focused test execution, then verify the corrected outcome tests under native Go and Gomad on seeds 11 and 17 with recorded commands, platform identity, and outcomes.
- Remove the four corresponding start/cancel outcome skips from the source qualification generator and regenerate its manifest only after verification passes.
- Production endpoint uniqueness behavior remains unchanged; diagnosis or a still-skipped test cannot close this required fix.

## Done summary
Each start/cancel outcome leaf subtest in `tests/nexus_api_test.go` now registers its own endpoint name, and every by-endpoint leaf asserts the poll request carries the name that leaf registered. The four outcome skips are removed from `tests.generator.json`, `tests.json` is regenerated and passes its staleness check, and the D22 sentence in `MILESTONES.md` states the verified result.

- Collision reproduced natively before the fix with `TEMPORAL_TEST_SHARED_CLUSTERS=1` (10 leaves fail with "already registered"); after the fix the same run, the default-pool run, and both whole suites pass.
- Gomad on darwin/arm64 (toolchain build `c0661e38b4e0`, runner `sha256:3062e6f84a85`): both Nexus API suites qualify on seeds 11 and 17 with repeat 2 and 4 exact replays, using their manifest settings without skips; a verbose run shows 32 outcome leaves passing per seed.
- linux/amd64 verification was not run; the host is unavailable here.
- No commits: the user retains commit ownership, so all changes are uncommitted in the working tree (`"commits": []`). The Gomad build compiled the shared tree, which also held other workers' uncommitted edits.
- `make lint-code` on `./tests` reports 0 issues; `make lint-code-fast` exits 2 for an inherited reason (the nested `tools/gomad3` module in the branch diff).
- Evidence: `.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d22-*` and `task22-review.md`.

stage: impl-review - ran (raw codex bridge on working-tree diff; commits forbidden)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: baseline: none (the spec defines no Quick commands), red before fix (exit 1): TEMPORAL_TEST_SHARED_CLUSTERS=1 go test -tags test_dep,disable_grpc_modules -count=1 -v -run '^TestNexusApiTestSuiteWith(LegacyErrorPaths|TemporalFailures)$/^TestNexus(Start|Cancel)Operation_Outcomes$' ./tests, TEMPORAL_TEST_SHARED_CLUSTERS=1 go test -tags test_dep,disable_grpc_modules -count=1 -v -run '^TestNexusApiTestSuiteWith(LegacyErrorPaths|TemporalFailures)$/^TestNexus(Start|Cancel)Operation_Outcomes$' ./tests, go test -tags test_dep,disable_grpc_modules -count=1 -v -run '^TestNexusApiTestSuiteWith(LegacyErrorPaths|TemporalFailures)$/^TestNexus(Start|Cancel)Operation_Outcomes$' ./tests, TEMPORAL_TEST_SHARED_CLUSTERS=1 go test -tags test_dep,disable_grpc_modules -count=1 -v -run '^TestNexusApiTestSuiteWith(LegacyErrorPaths|TemporalFailures)$' ./tests, tools/gomad3/.bin/gomad qualify-set --manifest=<two Nexus API suites of tests.json, skips removed> --working-dir=<repo> --prune-qualified-artifacts (darwin/arm64, seeds 11 and 17, repeat 2: qualified, 4 exact replays), tools/gomad3/.bin/gomad explore --seeds 11,17 ... go-test ./tests -- -test.run=<four outcome tests> -test.parallel=8 -test.v (32 leaves pass per seed), make -C tools/gomad3 tests-qualification-generate, make -C tools/gomad3 validate-qualification, make lint-code LINT_CODE_TARGETS=./tests GOLANGCI_LINT_BASE_REV=HEAD, make lint-code-fast: exit 2, inherited (nested tools/gomad3 module in the branch diff cannot be typechecked from the root module; 0 issues), linux/amd64: not run (unavailable on this host)
- PRs: