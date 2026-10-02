# fn-108.6 evidence: shared retention policy and artifact-input composition

Recorded 2026-10-01 on darwin/arm64 (go1.27.1, toolchain key `8d28bd44...`). Nothing is staged
or committed. The change is the working-tree delta against the pre-edit copies, stored as
[task6.diff](task6.diff). This task fulfils `fn-105-gomad-follow-ups-deferred-scope.2` (D2); that
task's state is untouched. The same extraction is the evidence fn-109 R3 reuses.

## The owner

`tools/gomad3/runner/retention.go` holds one private type and four functions. None keeps state,
takes a strategy flag, or touches a file, journal, counter or context.

| Policy | Owner | Result |
| --- | --- | --- |
| Novelty, retain decision, transcript prerequisite, count/byte pre-check, remaining bytes | `decideSuccessRetention(config, assessed, transcriptComplete, seenProbes, seenChoices, retained, retainedBytes)` | `successRetention{retain, novelProbes, novelChoices, maximumBytes}`, `*HostError` with reason `success_artifact_publication` or `success_retention_capacity` |
| Journal fields of a kept success | `successRetention.annotate(run, relative, storedBytes)` | sets `SuccessArtifact`, `SuccessArtifactBytes` and the novel values on the in-memory record |
| Store failure classification | `successPublicationFailure(err)` | `*HostError`; `artifact.CapacityError` becomes `success_retention_capacity` |
| Common artifact input | `executionArtifactInput(manifest, prepared, result, mountArtifact, worldBundle)` | `artifact.ArtifactInput` with the eight common fields |

`decideSuccessRetention` takes the `completedExecution` of fn-108.5 as its evidence and reads
the committed novelty sets and counters the caller passes. It computes novelty only under the
novel policy; the old code computed it under every policy and used it only under novel.

The callers keep everything durable: the `artifact.PublishArtifact` call and its store root
(`journal.SuccessesPath()` or the staged round), path relativization, `summary.SuccessArtifacts`,
`RetainedSuccesses++`, `RetainedSuccessBytes +=`, `addStrings` on the two novelty sets,
`journal.AppendExecution`, round staging and commit, duplicate-failure removal, and guidance.
Each of those statements is on the line it was on, relative to its neighbours. The seed path
used to compute novelty before the execution-evidence and journal-transition steps and now
computes it inside the decision, directly before the retain branch; nothing writes the two sets
in between.

`executionArtifactInput` is used at nine sites: three in `runner.go`, two in each exploration
file (the simulation sites then set `Simulation`), `guidance.go`, and `minimize_operation.go`
(which then sets `Simulation`). Corpus admission shares only this constructor; its eligibility
check and `corpus.Admit` are unchanged. `publishBoundedFailureArtifact` is unchanged and remains
the only failure-capacity owner. `novelStrings` and `addStrings` stay in `coverage.go`; the
duplicate pair was already deleted by fn-108.5.

No state grows with the number of selected seeds: the owner has none, and a decision holds two
slices bounded by one execution's observed probes and features. `execution.Result` is passed by
value, which copies slice headers and digests and no payload bytes.

## Characterization, written and run before the extraction

`runner/retention_characterization_test.go` drives `Explore` through the fake executors. Every
campaign has three executions: seeds 1 to 3 for the seed strategy, the root and its two other
alternatives for the exploration strategies, which put the two alternatives into one round.
Two test helpers gained a width field for that (`explorationExecutor.alternatives`,
`simulationExplorationExecutor.scenarios`; zero keeps the old width).

It passed on the unmodified production files first. The final version of the file was run
against the pre-edit production files again through `go test -overlay`
([task6-pre-edit-overlay.json](task6-pre-edit-overlay.json) maps the five edited files to their
pre-edit copies and removes `retention.go` and `retention_test.go`): exit 0, twice. It passed
unchanged after the seed migration, after the choice migration, after the simulation migration,
and at the end.

| Test | What it pins |
| --- | --- |
| `TestRetentionKeepsTheSameRunsAndArtifactsForEveryStrategy` | discard, all, novel (a probe after a known one, a probe ahead of its repeat, novel choices) and failures, per strategy. Each case runs with natural completion order and with completion forced into reverse rank order; the two projections must be equal. |
| `TestRetentionCapacityExhaustionFailsVisiblyForEveryStrategy` | count used up; count used up inside a round (seed keeps the committed second success, exploration returns only the committed round); a byte bound of one artifact (rejected before publication when used up exactly, otherwise the store is offered the bound minus the committed bytes). |
| `TestRetentionFailureLeavesNoveltyAndCountersAtTheCommittedState` | incomplete transcript behind a kept success, incomplete transcript ahead of its repeat, unusable success store, cancellation. Pins the interrupted counters and journal per strategy, then resumes and pins the result. |
| `TestGuidedAdmissionReplaysBeforeTheCorpusAdvances` | exact, diverged, unverified and failed replay, and an incomplete transcript. The case is published and the corpus index absent when the replay is requested; only an exact replay leaves an index and a case. |
| `TestRunBoundsActiveExecutionsAndRetentionAtTenAndOneHundredJobs` | 10 and 100 seeds with `Parallel=2`: exactly 2 active executions; discard, a count bound of 3 and a byte bound of 3.5 artifacts stop at the same place with `success_retention_capacity` at both sizes. |

Every kept success is opened and checked against its journal record (stored bytes, ordinal,
exact-replay success kind), the summary's success list against journal order, and its byte total
against the sum.

Each case also logs a canonical projection of the journal records and published manifests with
campaign id, timestamps, record hash and store paths held fixed. The 31 lines are byte-identical
before and after (sha256 `c9448c30...89756f6`):
[before](task6-projections-before.txt), [after](task6-projections-after.txt).

## Direct tests and mutants

`runner/retention_test.go` has four table tests with whole-value comparison, one per owner
function. Thirteen mutants of `retention.go`, compiled in with `go test -overlay`
([task6-mutation-red.txt](task6-mutation-red.txt)):

| Mutant | Direct tests | Characterization |
| --- | --- | --- |
| novelty ignores choices | 1 fails | 2 fail |
| novelty needs both a probe and a choice | 4 fail | 17 fail |
| no transcript check | 3 fail | 6 fail |
| count bound off by one | 1 fails | 8 fail |
| byte bound off by one | 1 fails | 1 fails |
| store offered the full byte bound | 2 fail | 4 fail |
| novelty recorded under the all policy | 9 fail | 20 fail |
| novel probes not recorded | 1 fails | 18 fail |
| capacity classified as publication | 1 fails | 3 fail |
| World payloads dropped from the input | 1 fails | pass; 28 of 31 projection lines change |
| probes judged against the choice set | 1 fails | 15 fail |
| stdout and stderr swapped | 1 fails | 1 fails |
| novel slice shared with the record | 1 fails | pass |

The byte-off-by-one mutant fails the characterization only when the second artifact has exactly
the measured size, which depends on the length of a timestamp; the direct test is the reliable
guard for that boundary.

## Size (counting rule v2)

| Scope | Before | After | Change |
| --- | --- | --- | --- |
| `runner` package production code lines | 7192 | 7172 | -20 |
| total production code lines | 58358 | 58338 | -20 |
| total production code bytes | 1961628 | 1959633 | -1995 |
| total test code lines | 43872 | 44778 | +906 |

"Before" is the tree at task start, which holds fn-108.2 to fn-108.5. Per file, in code lines:
`runner.go` -28, `choice_exploration_campaign.go` -22, `simulation_exploration_campaign.go` -18,
`guidance.go` -4, `minimize_operation.go` -4, new `retention.go` +56. No comment was removed
from an existing file. `size-compare.sh` exits 0 against the task start and against the fn-108
baseline (residual -286 code lines, -10691 bytes): [task6-size-compare.txt](task6-size-compare.txt).

## Gates (darwin/arm64, `-count=1`, no log contains `(cached)`)

| Gate | Result |
| --- | --- |
| `gofmt -l runner` | empty |
| `go vet -tags test_dep ./runner/...` | exit 0 |
| `go test -count=1 -tags test_dep ./runner/... ./artifact/... ./record/...` | exit 0 |
| `go test -count=1 -tags test_dep .` (architecture tests) | exit 0 |
| `make -C tools/gomad3 validate` | exit 0 |
| `make -C tools/gomad3 test-harness` | exit 0 |
| `make -C tools/gomad3 world-test` | exit 0 |
| `make -C tools/gomad3 test-host` | exit 0, 45 packages ok |
| `go test -count=1 -tags test_dep ./tools/gomad3sim/...` | exit 0 |
| `make gomad3-integration-test` | exit 0 |
| `make gomad3` | exit 0 |
| `make gomad3-smoke-qualification` | exit 0, `supported=4 unsupported=0 failed=0 infrastructure-errors=0 completed=4/4`, 4 replayed, 0 diverged |
| `api-capture.sh` diff against `api-baseline/` | empty |
| per-test dispositions | all 1140 baseline tests present with the same result; 23 new tests, 9 from this task: [listing](task6-test-dispositions-darwin-arm64.tsv) |
| guided campaign through the built CLI, `./basic/filesystem` of the core fixture module, seeds 1-4, `--keep-successes=novel` | exit 0; 1 success kept, 1 corpus entry admitted; `gomad replay` of the kept success reproduces it; a second campaign on the corpus adds nothing |
| choice-exploration through the built CLI, `./basic/concurrency`, 6 executions, `--coverage=semantic+choice --keep-successes=novel` | exit 0; 3 successes kept; `gomad replay` reproduces each with `choice-replay=exact` |

Start and end times are in [task6-gates.txt](task6-gates.txt). Two entries there need reading:

- The first `test-harness` run exited 2. The gate script had put `-json` into `GOFLAGS`, so
  `go env GOROOT` inside `TestBootstrapGofmtResolvesInstalledTool` printed JSON. That run is
  inconclusive. The tier was rerun with `GOFLAGS='-count=1 -v'` and exited 0.
- Three test helper functions were renamed after the tiers ran. `gofmt`, `vet`, the focused tier
  and the architecture tests were rerun on the final tree (`final-*` entries, all exit 0); the
  other tiers ran on the tree before the rename.

Not run: every linux/amd64 gate (no host). The task file's `go build ./...` was not used; it
fails on the runtime overlay before and after, as fn-108.2 recorded. `flowctl gate receipt`
declined to write a receipt because the worktree is dirty.

Another process cleared the shared Go build cache once during the task, which made two builds
fail on missing cache files. They were rerun; no result above comes from those attempts.

## Review

One round, `gpt-5.6-sol` at high effort, on the working-tree diff: SHIP, no findings. The
record is in [task6-review.md](task6-review.md).

## Recorded for the owner, not changed here

In the seed strategy a retention failure at ordinal N stops the campaign, but a later ordinal
that completed beside it is still judged, published and journaled. Its success counts as
retained, the journal then holds ordinal N+1 ahead of the resumed N, and the resumed N is judged
against the novelty N+1 committed. The pre-edit code behaves the same;
`TestRetentionFailureLeavesNoveltyAndCountersAtTheCommittedState/incomplete_transcript_ahead_of_its_repeat/seed`
pins it. Whether a seed campaign should journal past a failed ordinal belongs to the Runner
campaign owner.

`tools/gomad3/ARCHITECTURE.md` does not name `retention.go`; documentation is outside this
task's files.

baseline: green (`go test ./runner/... ./artifact/...` and `TestPackageArchitecture`, before any edit).
