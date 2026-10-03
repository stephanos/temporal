---
satisfies: [R5, R7]
---
# fn-121-shard-generated-cases-per-case-in-ci.3 Build the generated-Case test under test_dep alone so the sharded functional job runs it, and record the CI evidence

## Description
Move the generated-Case test and the helpers it uses out of the `integration` tag so `make functional-test-coverage` (the sharded job in `.github/workflows/run-tests.yml`) compiles and runs it, while `make umpire-check-live-tests` keeps running it under both tags. Then prove the sharded run locally and write the done summary with the numbers R7 asks for.

**Cross-spec entry gate:** after fn-112.10 (structural Case freeze); no Case, manifest, IR or fixture byte changes. fn-119.4/.6 also edit `tests/testpilot_generated_test.go`; whichever lands second rebases.

**Size:** M
**Files:** `tests/testpilot_generated_test.go` (tag line); new `tests/testpilot_helpers_test.go` (`//go:build test_dep`) receiving, bodies unchanged: `newTestpilotTestEnvironment` (`testpilot_testenv_test.go`), `CaseBinding`/`bindCase`/`defaultBinding`/`bindsNexusEndpoint` (`testpilot_run_case_test.go:18-88`), `testpilotLiveCase` and what `bindCase` needs from `testpilot_live_case_test.go:42+`, `controlledCase`/`bindControlledCase`/`requireInconclusiveWithoutDurableEvidence` (`testpilot_activity_control_test.go:29-104`), `umpireRepeatRunDirVariable`/`capturePath` (`testpilot_signature_test.go:23,110`), `writeExplorationArtifact` (`testpilot_exploration_test.go:24`); the files they leave keep `test_dep && integration`.
**Touches:** [tests/testpilot_*_test.go]

### Approach
- Move helpers transitively until `go vet -tags test_dep ./tests` and `go test -c -tags test_dep ./tests` build; then `go vet -tags 'test_dep integration' ./tests` must still build (nothing duplicated, nothing orphaned). Keep moved bodies byte-identical; the move is the only change.
- Keep `make umpire-check-live-tests` as is (Makefile :543-570 selects `^TestTestpilot` under `test_dep integration`, which still matches); no Makefile or workflow edit is needed, record that in the summary.
- Evidence: run `make functional-test-coverage`'s command shape locally for one shard (`TEST_TOTAL_SHARDS=5 TEST_SHARD_INDEX=<i>`, `-tags test_dep`, `-run '^TestTestpilotGeneratedCases'`) for each index and once unsharded; record per-shard and unsharded wall-clock, cluster count (24) and Run count (96) against fn-121.1's before-numbers (2 clusters, 128 Runs); confirm the sharded job's build (`make pre-build-functional-test-coverage`) compiles the test. If a shard exceeds a reasonable share of the job's 35m budget, say so in the summary rather than re-shaping.
- Run `make umpire-check-cases`, `make lint-code-fast` and `make umpire-check-live-tests` once; cite them in the done summary.

### Investigation targets
**Required:**
- `tests/testpilot_generated_test.go:1-60`
- `tests/testpilot_live_case_test.go:30-120`
- `Makefile:60-64,543-570,979-986`
- `.github/workflows/run-tests.yml:140-150,415-427`
**Optional:**
- `.github/workflows/umpire.yml:26-29`
- `tools/umpire0/campaign/integration_test.go:1` - the other user of the `integration` tag, outside `./tests`

### Quick commands
```bash
go vet -tags test_dep ./tests && go vet -tags 'test_dep integration' ./tests
make pre-build-functional-test-coverage
for i in 0 1 2 3 4; do TEST_TOTAL_SHARDS=5 TEST_SHARD_INDEX=$i go test -count=1 -v -tags test_dep ./tests -run '^TestTestpilotGeneratedCases'; done
make umpire-check-cases && make lint-code-fast && make umpire-check-live-tests
```

### Execution constraints
- No change to the salt file, the optimizer, the shard count, the database matrix or any workflow.
- The hand-written Testpilot suites keep `test_dep && integration`.

## Acceptance
- [ ] `./tests` builds under `-tags test_dep` and under `-tags 'test_dep integration'`; the generated-Case test and its helpers carry `test_dep` alone, every other Testpilot file keeps `test_dep && integration`, moved bodies unchanged.
- [ ] A `test_dep`-only run with five local shards runs every Case once across the shards; `make umpire-check-live-tests` still runs `TestTestpilotGeneratedCases` and passes.
- [ ] Done summary records unsharded and per-shard wall-clock before and after, cluster and Run counts, and that no Makefile or workflow edit was needed.
- [ ] `make umpire-check-cases` and `make lint-code-fast` pass; no Case, manifest, IR or fixture byte changed.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
