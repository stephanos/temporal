# fn-108.5 evidence: shared completed-execution assessment owner

Recorded 2026-10-01 on darwin/arm64 (go1.27.1, toolchain key `8d28bd44...`). Nothing is staged
or committed. The change is the working-tree delta against the pre-edit copies, stored as
[task5.diff](task5.diff). This task fulfils `fn-105-gomad-follow-ups-deferred-scope.1` (D1); that
task's state is untouched.

## The owner

`tools/gomad3/runner/completion.go` holds two pure functions and one private type.

| Step | Function | Result |
| --- | --- | --- |
| World decode, compose, validate, schema and seed check | `assessWorld(result, seed, transitionLimit)` | `execution.Bundle`, plain `error` |
| Semantic coverage, choice-feature projection, classification | `assessCompletion(result, terminal, mode, prepared)` | `completedExecution`, `*HostError` with reason `semantic_coverage` or `choice_coverage` |

The interface is staged because the seed path reacts differently per step: a World error
publishes a Runner failure artifact, a coverage or choice error preserves the partial. The
exploration paths return a `HostError` for either. Neither function reads `CampaignSpec`, takes
a strategy flag, or touches a process, file, journal, counter or context.

Left with the callers: `prepared.Verify()`, the `ctx.Err()` checks, journal transitions,
counters, cancellation, the "controller result is not expandable" check, `ProjectReplayPlan`
and tape fields, `explorationOutcomeSHA256`, `simulationrecord.*`, and `mountArtifactForRun`.

`recordedWorldForMinimization` is unchanged. Its diagnostics differ ("decode minimization World
record: ...", "minimization World record seed or schema changed"), so routing it through
`assessWorld` would change error text.

`novelSemanticProbes` and `addSemanticProbes` were byte-for-byte copies of `novelStrings` and
`addStrings` in `coverage.go`. They are deleted and their seven call sites use the `coverage.go`
functions. The novelty decisions themselves stay where they were for fn-108.6.

## One difference that no input reaches

The seed path returned from `runLocal` immediately when `SummarizeSemanticProbes(nil)` failed.
The shared step reports that failure as `semantic_coverage` like a decode failure, so the seed
path would now preserve the partial and continue draining. `SummarizeSemanticProbes(nil)` cannot
fail: with no probes its only error branch, an unknown probe name, is never entered. The reason
string is the same in both versions.

## Characterization, written and run before the extraction

`runner/completion_characterization_test.go` drives `Explore` through the existing fake
executors. `char-before` ran against the unmodified production files (53 subtests, exit 0, and
exit 0 again with `-count=3`). The same file passed unchanged after the seed migration, after
the choice migration and after the simulation migration.

`TestCompletionFaultsKeepReasonPrecedenceAndEvidence` runs 16 faults against the three
strategies. Each case pins `HostError.Reason`, the exact cause text, the five counters, every
published failure artifact (kind, reason, replay mode, World terminal, choice profile), every
journaled execution and every surviving partial state.

| Fault | Seed | Choice and simulation exploration |
| --- | --- | --- |
| malformed World | `world_record`, Runner failure artifact journaled | `world_record` |
| World seed mismatch | `world_record`, artifact journaled | `world_record` |
| malformed semantic coverage | `semantic_coverage`, partial preserved | `semantic_coverage` |
| malformed choice trace (projection) | `choice_coverage`, partial preserved | `choice_coverage` |
| choice trace rejected by supervision | `choice_trace_malformed` | `choice_trace_malformed` |
| missing terminal choice frame | `choice_trace_unterminated` | `choice_trace_unterminated` |
| watchdog | watchdog artifact, no choice profile | `choice_trace_malformed` (no tape to expand) |
| cancelled execution | `runner_cancelled` artifact | `runner_cancelled` |
| malformed World + coverage + choices | `world_record` | `world_record` |
| malformed coverage + choices | `semantic_coverage` | `semantic_coverage` |
| watchdog + malformed World | `world_record` | `world_record` |
| watchdog + malformed coverage | `semantic_coverage` | `semantic_coverage` |
| watchdog + malformed choice trace | `artifact_publication` (projection skipped) | `choice_trace_malformed` |
| cancelled + malformed World | `world_record` | `runner_cancelled` |
| cancelled + malformed coverage | `semantic_coverage` | `runner_cancelled` |
| supervision failure + seed mismatch | `choice_trace_malformed` | `choice_trace_malformed` |

`TestExplorationCancellationIsAHostFailure` covers a cancelled context for both exploration
strategies (`cancelled`); the seed strategy already had `TestRunCancellationIsAHostFailure`.

`TestCompletionProjectsWorldCoverageAndChoicesForEveryStrategy` checks the success path per
strategy against independently computed values: outcome, World manifest, semantic coverage,
choice-feature projection and the journaled probes and features. It also logs the canonical
JSON of the execution evidence and the journal records with elapsed time zeroed. The three log
lines are byte-identical before and after the extraction (sha256 `661d0fe3...acb40e5e`):
[before](task5-canonical-record-inputs-before.txt),
[after](task5-canonical-record-inputs-after.txt).

## Direct tests of the owner

`runner/completion_test.go`: `TestAssessWorldValidatesTheRecordAgainstItsSeed` (6 cases) and
`TestAssessCompletionProjectsCoverageInOrderAndClassifies` (11 cases), whole-value comparison.

Mutants of `completion.go`, compiled in with `go test -overlay`
([task5-mutation-red.txt](task5-mutation-red.txt)):

| Mutant | Direct tests | Characterization |
| --- | --- | --- |
| project choices of a killed target | 2 fail | 7 fail |
| drop the seed check | 1 fails | 3 fail |
| decode semantic coverage in every mode | 4 fail | pass |
| classify as cancelled | 6 fail | 8 fail |
| ignore the World terminal | 1 fails | 3 fail |
| report `choice_coverage` as `semantic_coverage` | 1 fails | 3 fail |
| project choices before semantic coverage | 1 fails | 3 fail |
| ignore the `execution.Validate` error | pass | pass |

The last mutant survives. `Validate` recomposes the bundle `ComposeRecording` just built, and
no recording was found that composes and then fails validation.

## Size (counting rule v2)

| Scope | Before | After | Change |
| --- | --- | --- | --- |
| `runner` package production code lines | 7255 | 7192 | -63 |
| `runner` tree production code lines | 20003 | 19940 | -63 |
| total production code lines | 58421 | 58358 | -63 |
| total production code bytes | 1963471 | 1961628 | -1843 |
| total test code lines | 43309 | 43872 | +563 |

"Before" is the tree at task start, which already holds fn-108.2/.3/.4. The stored "before"
listing was taken after the characterization test file existed (422 test code lines) and before
any production edit, so the test row subtracts that file. Per file, in physical
lines against the pre-edit copies: `runner.go` +23/-82, `choice_exploration_campaign.go` +8/-41,
`simulation_exploration_campaign.go` +8/-42, new `completion.go` 70 (55 code lines). About 13
of the 63 lines are the two deleted duplicate helpers. `size-compare.sh` exits 0 against the
task start and against the fn-108 baseline (residual -266 code lines, -8696 bytes):
[task5-size-compare.txt](task5-size-compare.txt).

`coverage.go`, `minimize_operation.go` and `runner_test.go` are unchanged.

## Gates (darwin/arm64)

| Gate | Result |
| --- | --- |
| `gofmt -l runner` | empty |
| `go vet -tags test_dep ./runner/...` | exit 0 |
| `go test -count=1 -tags test_dep ./runner/... ./artifact/... ./record/...` | exit 0 |
| `go test -count=1 -tags test_dep .` (architecture tests) | exit 0 |
| `make -C tools/gomad3 validate` | exit 0 |
| `make -C tools/gomad3 test-harness` | exit 0 |
| `make -C tools/gomad3 world-test` | exit 0, all three packages from the test cache (no World input changed) |
| `make -C tools/gomad3 test-host` | exit 0, 45 packages ok |
| `go test -tags test_dep ./tools/gomad3sim/...` | exit 0, from the test cache (no input changed) |
| `make gomad3-integration-test` | exit 0 |
| `make gomad3` | exit 0 |
| `make gomad3-smoke-qualification` | exit 0, `supported=4 unsupported=0 failed=0 infrastructure-errors=0 completed=4/4` |
| `api-capture.sh` diff against `api-baseline/` | empty |
| per-test dispositions, 11 affected packages | 378 baseline tests present with the same disposition; 7 new tests (5 from this task, 2 from fn-108.2) |

Start and end times are in [task5-gates.txt](task5-gates.txt).

Not run: every linux/amd64 gate (no host). The task file's `go build ./...` was not used; it
fails on the runtime overlay before and after, as fn-108.2 recorded.

baseline: green (focused Quick commands and `TestPackageArchitecture`, before any edit).
