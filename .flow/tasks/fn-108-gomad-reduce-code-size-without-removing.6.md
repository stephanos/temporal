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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
