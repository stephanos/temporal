---
satisfies: [R1, R6]
---
# fn-101-gomad-f7-any-functional-test-and-ci.1 Generate the ./tests qualification manifest from go test -list with a staleness check

## Description
Generator tool plus check; new tests default to `qualified`; exclusions by name with owner and date.

## Acceptance
- generator and check exist and pass

## Done summary
Added `qualification/set/manifestgen` and `gomadtool qualification-manifest-generate`. Together they generate `tools/gomad3integration/qualification/tests.json`: one tier-3 workload per top-level test in `./tests` (147 today), defaulting to `qualified` on both platforms. The input is `tests.generator.json`, which holds defaults, per-test overrides (the F6 slice's required_probes), and exclusions that must carry an owner, a date and a reason. `-check` runs in `make validate` and CI's validate step and fails on a stale manifest. `make gomad3-tests-qualification` regenerates the manifest and runs the set with pruning. The set's workload bound went from 64 to 512, and its test-name pattern now accepts every name the go command runs.

Tests: `manifestgen_test.go` covers a new test appearing as default qualified, a removed test disappearing, overrides and exclusions being applied, exclusions missing an owner, date or reason being rejected, the stale check failing, parity with real `go test -list` on a fixture, platform-specific and feature-tag mismatches, ID collisions, and the checked-in manifest being current. The parser's 147 names match real `go test -list` on `./tests`.

Follow-ups for .2/.4: `gomad3.yml` does not trigger on `tests/**` yet (task .4 owns path triggers). Workloads inherit `run_timeout` 2m; per-test timeout overrides may be needed during .2's triage.

baseline: green (make validate; focused qualification/set, cmd/gomadtool, cli tests)

stage: impl-review - ran (codex fan-out: NEEDS_WORK on feature ToolTags -> fixed; re-dispatch NEEDS_WORK on test-name validation -> fixed; re-review SHIP)
## Evidence
- Commits: cfeef43f03cb8b0ed1d0efcf077878234497c3ad, b9fb2e51d84e4ed0a3d1659d77d0bc02670e1178, d6bb940da05ec3f6bcae18efd2848cde74ce537a, febb32c5e29bd3498dcf53a646b574c0ad85a3fe
- Tests: make -C tools/gomad3 validate, tools/gomad3/.toolchain/bin/go test -count=1 -tags test_dep . ./qualification/... ./cmd/... (in tools/gomad3), tools/gomad3/.bin/gomad qualify-set --check --manifest=tools/gomad3integration/qualification/tests.json --working-dir=., go test -tags disable_grpc_modules,gomad,test_dep -list '.*' ./tests (147 names, identical to generated manifest)
- PRs: