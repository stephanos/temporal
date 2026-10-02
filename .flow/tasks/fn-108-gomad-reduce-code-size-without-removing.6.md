---
satisfies: [R7]
---
# fn-108-gomad-reduce-code-size-without-removing.6 Shared retention policy and artifact-input composition (fulfils fn-105.2 D2)

## Description
Stage 3 (R7): give the common novelty predicate, success-retention capacity decision and artifact-input composition one private owner, fed by the assessment result of the previous task. The durable transactions stay separate: seed ordinal commits, atomic exploration round commits with hash-linked journals, and replay-verified corpus admission.

**Obligation link:** completing this task fulfils `fn-105-gomad-follow-ups-deferred-scope.2` (D2, origin fn-102 R3). Do not start or close fn-105.2 from here and do not create a second D2 task: on completion, report "fulfils fn-105.2 (D2)" with the evidence paths so the conductor can close fn-105.2 against this task's evidence. The original brief is `.flow/tasks/fn-102-gomad-architecture-consolidate.3.md`.

**Size:** M
**Files:** `tools/gomad3/runner/runner.go`, `choice_exploration_campaign.go`, `simulation_exploration_campaign.go`, `guidance.go`, `coverage.go`, new private `tools/gomad3/runner/retention.go` and `retention_test.go` (or the files added by the previous task), `runner_test.go`
**Touches:** [tools/gomad3/runner/*.go] — wide on purpose: retention helpers and the strategy/resume tests share the `runner` package.

### Approach

Repeated policy, as located at `d4d800fb47` (line numbers shift after the previous task; re-anchor by symbol):

| Policy | Seed (`runner.go`) | Choice | Simulation | Other |
| --- | --- | --- | --- | --- |
| Novelty: `novelSemanticProbes` + `novelStrings` | `:912-913` | `:398-399` | `:430-431` | `novelSemanticProbes`/`addSemanticProbes` (`runner.go:1860`, `:1870`) are line-for-line copies of `novelStrings`/`addStrings` (`coverage.go:56`, `:66`) |
| Retain decision (`KeepSuccessesAll`, or `KeepSuccessesNovel` with a novel probe or choice feature) | `:957` | `:400` | `:432` | |
| Complete-transcript prerequisite → `success_artifact_publication` | `:959-964` | `:402-404` | `:434-436` | |
| Count/byte pre-check → `success_retention_capacity` | `:965-970` | `:405-407` | `:437-439` | |
| Success store bound `SuccessBytesLimit - RetainedSuccessBytes`, `artifact.CapacityError` → `success_retention_capacity` | `:977`, `:1001-1005` | `:412`, `:416-422` | `:445`, `:449-455` | |
| `artifact.ArtifactInput{Manifest, TargetPath, Stdout, Stderr, IOTranscript, ChoiceTrace, ReadOnlyMounts, World}` | `:683`, `:977`, `:1062` | `:412`, `:445` | `:445`, `:479` (adds simulation payloads) | `guidance.go:91`, `minimize_operation.go:348` |
| Failure count/byte capacity and deduplication | `publishBoundedFailureArtifact` (`runner.go:1883-1928`) is already the single owner — reuse it, do not duplicate or split it | | | |

1. **Characterize first** (equivalence pin). Record a fixed-input projection of the journaled `campaign.ExecutionRecord`s and published manifests for each strategy before editing, using supplied identities, and assert the same canonical bytes after. Reuse the existing tests as the base: `TestRunRetainsOnlyProbeNovelSuccessesWithinExplicitBounds`, `TestRunRetainsOnlyChoiceNovelSuccessesAndRecordsTheirFeatures`, `TestRunFailsClosedWhenSuccessRetentionCountIsExhausted`, `TestRunRejectsSuccessfulRetentionWithoutReplayTranscript`, `TestFailureArtifactCapacityRejectsBeforePublication`, `TestRunSimulationExplorationRetainsExactDeduplicatedSimulationFailure`, `TestRunMergesParallelCompletionsInSelectionOrdinalOrder`, `TestRunResumeRestoresSeenChoiceFeaturesBeforeNovelRetention`, `TestRunChoiceExplorationResumeRerunsTheWholeIncompleteRound`, `TestRunSimulationExplorationResumePreservesCommittedCandidates`, `TestRunGuidesFromReplayVerifiedChoiceCoverage`.
2. **Extract** one private owner for: the novelty computation (one string-set helper replacing the duplicate pair), the retain decision with its transcript prerequisite and count/byte pre-check, the remaining-bytes calculation with capacity-error classification, and one constructor for the common `artifact.ArtifactInput` fields (simulation payloads and guidance coverage are added by their callers). The owner computes and returns decisions; it holds no campaign state.
3. **Callers keep:** the publication call site and store root (`journal.SuccessesPath()` versus `staged.Path()/successes`), path relativization (`filepath.Rel(batchPath, …)` versus `explorationPublishedPath`), duplicate-failure removal and directory sync in exploration rounds, `journal.AppendExecution`, round staging/commit, and every counter update. Novelty sets and `RetainedSuccesses`/`RetainedSuccessBytes` advance only at the existing commit points (seed: after `AppendExecution` succeeds with no host failure, `runner.go` near `:1040`; exploration: on round commit). A failed publication, cancellation or interrupted round must leave them at the committed state.
4. **Corpus admission stays independent:** `guidanceCampaign.MergeRun` (`guidance.go:67-108`) keeps its own eligibility (`IOTranscript.Complete` and `ReplayMode != ReplayNone`) and mandatory replay through `corpus.Admit` before the index advances; it may share only the artifact-input constructor.
5. **Bound check from the original brief:** extend the existing fake-executor parallelism test (`TestRunPreparesOnceBoundsParallelismAndGroupsMatchingFailures`, `runner_test.go:40`) to 10 and 100 jobs with `Parallel=2`, `KeepSuccesses=none` and identical bounded payloads, asserting at most 2 active executions; exercise fixed success count and byte limits separately at both sizes and require the same capacity classification. Review the new owner for state proportional to total selected seeds or extra full-payload copies. This is a control-bound check, not a timing benchmark.

### Investigation targets

**Required:**
- the ranges in the table above, plus `tools/gomad3/runner/runner.go:1015-1045` (seed commit point) and `:1883-1928`
- `tools/gomad3/runner/guidance.go`
- `.flow/tasks/fn-102-gomad-architecture-consolidate.3.md` — original brief
- the files added by the previous task (assessment result type)

**Optional:**
- `tools/gomad3/runner/internal/campaign/retained_evidence_test.go`, `choice_exploration_journal_test.go`, `simulation_exploration_journal_test.go`
- `tools/gomad3/runner/internal/corpus/corpus.go`

### Quick commands

```bash
cd tools/gomad3
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -tags test_dep ./runner -run 'TestRun|TestFailureArtifactCapacity|Retention'
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -tags test_dep ./runner/... ./artifact/...
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -tags test_dep . -run TestPackageArchitecture
```

### Key context

- No generic strategy framework, no new package-forwarding layer, no persistent entity, no second replay controller (spec "Shared retention and artifact composition"). `Artifact` remains the publication owner.
- The extraction must reduce net authored production lines with its new types counted; record `size-count.sh` before and after. If it cannot, stop and report.
- Deterministic publication order under out-of-order host completion is owned by `orderShardRunCompletions` (`campaign.go`) and the exploration round loops; do not move it.

### Standing constraints (every fn-108 task)

- The user owns commits: no `git commit`, `git add`, `git stash`, and no worktrees. Leave changes in the working tree and report the paths.
- No new dependencies (Go modules or external tools). `tools/gomad3` is a nested module pinned to go1.27.1; `tools/gomad3sim` and `tools/gomad3integration` belong to the root module.
- Preserve existing comments: keep them with the logic they describe when code moves, and delete a comment only together with the dead code it documents. Do not compress formatting.
- Public Go names/signatures/fields/defaults, CLI commands/flags/exit statuses, schemas, canonical bytes, `HostError.Reason` values and failure precedence stay unchanged (spec "API Contracts", R8).
- Host is darwin/arm64. linux/amd64 gates cannot run here: list them as "not run (no host)" in the evidence, never as passed.
- Focused tests run from `tools/gomad3` as `env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -tags test_dep <packages>`. `.toolchain/bin/go` is the patched toolchain; `make -C tools/gomad3 toolchain` rebuilds it (needs go.dev access). New test assertions use `require` with whole-value equality.
- Evidence (commands, platform, results, remaining failures) goes under `.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/`. A defect found on the way is recorded for its existing owner, not fixed here.

## Acceptance
- [ ] Novelty, the retain decision with its transcript prerequisite, success capacity calculation/classification and common artifact-input composition each have one private owner used by seed, choice and simulation paths; the duplicate `novelSemanticProbes`/`addSemanticProbes` pair is gone; `publishBoundedFailureArtifact` remains the single failure-capacity owner.
- [ ] Seed ordinal commits, exploration round staging/commit and journals, and replay-verified corpus admission remain separate; no counter, novelty set or journal write moved into the policy owner.
- [ ] All/novel/discard success modes, failure deduplication, guided corpus admission and first-novel ordering produce the same fixed-identity canonical manifests and journal records as the pre-change projection.
- [ ] Count and byte exhaustion fail with `success_retention_capacity` (or the existing failure-capacity error); publication failure, cancellation and an interrupted round leave novelty and counters at the committed state; resume/recover tests pass unchanged.
- [ ] 10- and 100-job runs with `Parallel=2` keep at most 2 active executions and the same capacity classification; review notes confirm no state proportional to total selected seeds and no added full-payload copies.
- [ ] Focused runner/artifact tests and `TestPackageArchitecture` pass on darwin/arm64; `tools/gomad3/runner` production line count is lower than before this task, both counts recorded.
- [ ] Completion report states "fulfils fn-105.2 (D2)" with evidence paths; fn-105.2 itself is left for the conductor; nothing staged or committed.


## Done summary
The seed, choice-exploration and simulation-exploration paths now take the success-retention decision and the common artifact input from one private owner, `tools/gomad3/runner/retention.go`. Fulfils fn-105.2 (D2); that task's state is untouched. Nothing is staged or committed; the delta is `task6.diff` and the full record is `task6-evidence.md` under `.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/`.

- Owner: `decideSuccessRetention(config, assessed, transcriptComplete, seenProbes, seenChoices, retained, retainedBytes)` returns `successRetention{retain, novelProbes, novelChoices, maximumBytes}` or a `*HostError` (`success_artifact_publication`, `success_retention_capacity`). `annotate` sets the journal fields of a kept success, `successPublicationFailure` classifies a store error, and `executionArtifactInput` builds the eight common `artifact.ArtifactInput` fields at nine sites, including guidance and minimization. The owner has no state and no strategy flag.
- Callers keep the publication call, store root, path relativization, every counter update, the novelty-set advances, journal appends, round staging and commit, and corpus admission, each where it was. `publishBoundedFailureArtifact` is unchanged.
- Characterization came first: `retention_characterization_test.go` passed on the pre-edit production files and unchanged after each of the three migrations. It covers discard/all/novel and failures per strategy under natural and reversed completion order, count and byte exhaustion, failed publication and cancellation followed by resume, guided admission with replay before the index advances, and 10 and 100 jobs at `Parallel=2`. The 31 fixed-identity projections of journal records and manifests are byte-identical before and after.
- Direct tests: `retention_test.go`. All 13 overlay mutants of the owner fail them; 11 also fail the characterization.
- Size (rule v2): production Go -20 code lines, -1995 bytes; `runner` package 7192 -> 7172; test Go +906 code lines. `size-compare.sh` exits 0 against the task start and the fn-108 baseline.
- Recorded, not changed: in the seed strategy an ordinal that completed beside a failed retention is still published and journaled, so the journal can hold ordinal N+1 ahead of the resumed N. The pre-edit code does the same and a characterization case pins it.
- Not done: `tools/gomad3/ARCHITECTURE.md` does not name the new owner (outside this task's files).
- Gates on darwin/arm64 with `-count=1`, all exit 0: gofmt, vet, `go test ./runner/... ./artifact/... ./record/...`, architecture tests, `make -C tools/gomad3 validate test-harness world-test test-host` (45 packages ok), `go test ./tools/gomad3sim/...`, `make gomad3-integration-test`, `make gomad3`, `make gomad3-smoke-qualification` (4/4 supported, 4 replayed, 0 diverged). A first `test-harness` run was inconclusive because the gate script put `-json` into `GOFLAGS`; the rerun exited 0. Guided and choice-exploration campaigns through the built CLI on the core fixture module kept novel successes that replay exactly. `go doc` and CLI capture diff against `api-baseline/`: empty. All 1140 baseline tests keep their result. linux/amd64: not run (no host).

baseline: green (focused Quick commands and `TestPackageArchitecture`, before any edit)

stage: impl-review - ran (raw codex bridge on working-tree diff; commits forbidden) (model: gpt-5.6-sol) [round 1 SHIP, no findings; record in task6-review.md]
stage: plan-sync - skipped(config: planSync.enabled != true)

GATE_SKIPPED lines: none.
## Evidence
- Commits:
- Tests: baseline (pre-edit): env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./runner/... ./artifact/... -> exit 0; . -run TestPackageArchitecture -> exit 0, characterization on pre-edit production (go test -overlay task6-pre-edit-overlay.json, -count=2) -run 'TestRetention|TestGuidedAdmission|TestRunBoundsActive' -> exit 0, gofmt -l runner -> empty, env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go vet -tags test_dep ./runner/... -> exit 0, env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./runner/... ./artifact/... ./record/... -> exit 0, env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep . -> exit 0, GOFLAGS=-count=1 make -C tools/gomad3 validate -> exit 0, GOFLAGS='-count=1 -v' make -C tools/gomad3 test-harness -> exit 0 (a first run with -json in GOFLAGS was inconclusive: exit 2 caused by the flag), GOFLAGS='-count=1 -v' make -C tools/gomad3 world-test -> exit 0, GOFLAGS='-count=1 -v' make -C tools/gomad3 test-host -> exit 0 (45 packages ok), go test -count=1 -json -tags test_dep ./tools/gomad3sim/... -> exit 0, make gomad3-integration-test -> exit 0, make gomad3 -> exit 0, make gomad3-smoke-qualification -> exit 0 (supported=4 completed=4/4, replayed 4, diverged 0), api-capture.sh diff against api-baseline/ -> empty, per-test dispositions: 1140 baseline tests unchanged, 23 new, gomad explore --guide --corpus ... --keep-successes=novel --seeds 1-4 go-test ./basic/filesystem (core fixture) -> exit 0, 1 kept, 1 admitted; gomad replay -> reproduced=true, gomad explore --strategy=choice-exploration --seeds 7 --max-executions=6 --coverage=semantic+choice --keep-successes=novel go-test ./basic/concurrency -> exit 0, 3 kept; gomad replay x3 -> reproduced=true choice-replay=exact, 13 overlay mutants of retention.go: 13 fail the direct tests, 11 fail the characterization tests, linux/amd64 gates: not run (no host), flowctl gate classify: FULL; flowctl gate receipt: NO_RECEIPT (worktree dirty)
- PRs: