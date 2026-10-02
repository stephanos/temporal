---
satisfies: [R6]
---
# fn-108-gomad-reduce-code-size-without-removing.5 Shared completed-execution assessment owner in Runner (fulfils fn-105.1 D1)

## Description
Stage 2 (R6): give the common interpretation of a completed execution one private owner in the `runner` package and migrate the seed, choice-exploration and simulation-exploration completion paths to it.

**Obligation link:** completing this task fulfils `fn-105-gomad-follow-ups-deferred-scope.1` (D1, origin fn-102 R2). Do not start or close fn-105.1 from here and do not create a second D1 task: on completion, report "fulfils fn-105.1 (D1)" with the evidence paths so the conductor can close fn-105.1 against this task's evidence. The original brief is `.flow/tasks/fn-102-gomad-architecture-consolidate.2.md`.

**Size:** M
**Files:** `tools/gomad3/runner/runner.go`, `tools/gomad3/runner/choice_exploration_campaign.go`, `tools/gomad3/runner/simulation_exploration_campaign.go`, new `tools/gomad3/runner/completion.go` and `completion_test.go`, `tools/gomad3/runner/runner_test.go`
**Touches:** [tools/gomad3/runner/runner.go, tools/gomad3/runner/choice_exploration_campaign.go, tools/gomad3/runner/simulation_exploration_campaign.go, tools/gomad3/runner/completion*.go, tools/gomad3/runner/coverage.go, tools/gomad3/runner/minimize_operation.go, tools/gomad3/runner/runner_test.go]

### Approach

The three copies, as of `d4d800fb47`:

| Step | Seed (`runner.go`, inside `runLocal`) | Choice (`choice_exploration_campaign.go`, `processExplorationCompletion` at `:291`) | Simulation (`simulation_exploration_campaign.go`, `processSimulationExplorationCompletion` at `:307`) |
| --- | --- | --- | --- |
| World decode → `execution.ComposeRecording` → `execution.Validate` → snapshot schema + seed check | `:845-872` | `:316-333` | `:333-350` |
| Semantic coverage (`SummarizeSemanticProbes(nil)`, then `DecodeSemanticCoverage` when `coverageHasSemantic`) | `:873-893` | `:334-343` | `:351-360` |
| Choice-feature projection (`coverageHasChoice && choiceTraceObserved` → `projectChoiceFeatures`) | `:894-911` | `:344-355` | `:361-372` |
| `execution.Classify(result, false, terminal)` | `:914` | `:356` | `:373` |

1. **Characterize first.** Before moving code, add tests through the existing fake-executor harness in `runner_test.go` that fix, per strategy, the `HostError.Reason` and the published/journaled evidence for: malformed World record, World seed mismatch, malformed semantic coverage, malformed choice trace, missing terminal evidence, watchdog timeout, cancellation, and two simultaneous faults (for example malformed World plus malformed coverage). Reuse existing cases where they already pin a reason (`TestRunRejectsInvalidConnectedWorldBeforePublication`, `TestRunClassifiesInvalidChoiceTraceTerminalEvidence`, `TestRunClassifiesWatchdogTimeoutBeforeUnterminatedChoiceTrace`, `TestRunCancellationIsAHostFailure`); add only missing strategy/fault combinations. They must pass unchanged before and after the extraction; this is the equivalence pin.
2. **Extract** concrete private types/functions into `runner/completion.go`. Inputs are narrow: the captured `execution.Result`, the job seed, `WorldTransitionLimit`, the coverage mode, and what `projectChoiceFeatures` needs from the prepared target — not `CampaignSpec`. Output is detached validated data: the World bundle, `deterministicio.SemanticCoverage`, the choice features with their optional `choice.FeatureProjection`, and the `execution.Classification`.
3. **Staged, not monolithic.** The callers interleave different effects between the steps, so expose the steps so each caller keeps its own reaction: on a World error the seed path calls `publishRunnerFailure(completion, "world_record")` and continues the loop with `hostFailure`/`controller.Stop()`/`activeCancel()`; on coverage and choice errors it calls `preservePartial(completion.journal)`; the exploration paths return `&HostError{Reason: ...}` directly. The reasons `world_record`, `semantic_coverage`, `choice_coverage` and their order stay exactly as today. No strategy flag parameter.
4. **Stays with the callers:** `prepared.Verify()` and the `ctx.Err()` check at the top of both exploration functions, journal transitions, counters/statistics, cancellation, the "controller result is not expandable" check, `choice.ProjectReplayPlan` and tape fields, `explorationOutcomeSHA256`, `simulationrecord.*` (runtime decisions, `ResultForRecord`, `ProjectArtifact`), forced prefixes, and `mountArtifactForRun` — the seed path builds the mount artifact lazily with reason `execution_evidence` or `manifest`, the exploration paths build it up front with `manifest`.
5. Reuse `world.DecodeRecording`, `execution.ComposeRecording`, `execution.Validate`, `deterministicio.DecodeSemanticCoverage`, `projectChoiceFeatures` (`coverage.go:21`); do not reimplement a codec. `recordedWorldForMinimization` (`minimize_operation.go:398-418`) performs the same World steps with different diagnostics; route it through the shared World step only if its error text and behaviour stay identical, otherwise leave it and say so.
6. Move the comment "A target the watchdog or a cancellation killed wrote no choice trace to project; the termination is its outcome." with the logic it describes, once.
7. Test the private interface directly in `completion_test.go` and keep the strategy-level tests.

### Investigation targets

**Required:**
- the ranges in the table above
- `tools/gomad3/runner/runner.go:640-720` — `publishRunnerFailure` closure and seed-loop failure idiom
- `tools/gomad3/runner/coverage.go`
- `.flow/tasks/fn-102-gomad-architecture-consolidate.2.md` — original brief
- `.flow/artifacts/fn-102-gomad-architecture-consolidate/architecture-assessment.md` — "Ranked improvements" item 2

**Optional:**
- `tools/gomad3/runner/minimize_operation.go:300-420`
- `tools/gomad3/architecture_test.go:468-495` — forbidden intra-runner imports (a new same-package file needs no rule change)

### Quick commands

```bash
cd tools/gomad3
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -tags test_dep ./runner -run 'Test(Run|ValidateConfig|ExecutionEvidence|Completion|Classify)'
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -tags test_dep ./runner/... ./artifact/... ./record/...
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -tags test_dep . -run TestPackageArchitecture
```

### Key context

- Net authored production lines must go down after the new file is counted (spec "Shared retention" closing paragraph and R1). Run `sh .flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/size-count.sh` before and after and record both. If a faithful extraction cannot reduce the count, stop and report rather than merging behaviours to force it.
- Success/failure retention, novelty and `artifact.ArtifactInput` composition belong to the next task; leave them in place here.
- Fixed-identity comparison: tests use supplied identities; a rebuilt Runner's build identity legitimately differs and is reported separately.

### Standing constraints (every fn-108 task)

- The user owns commits: no `git commit`, `git add`, `git stash`, and no worktrees. Leave changes in the working tree and report the paths.
- No new dependencies (Go modules or external tools). `tools/gomad3` is a nested module pinned to go1.27.1; `tools/gomad3sim` and `tools/gomad3integration` belong to the root module.
- Preserve existing comments: keep them with the logic they describe when code moves, and delete a comment only together with the dead code it documents. Do not compress formatting.
- Public Go names/signatures/fields/defaults, CLI commands/flags/exit statuses, schemas, canonical bytes, `HostError.Reason` values and failure precedence stay unchanged (spec "API Contracts", R8).
- Host is darwin/arm64. linux/amd64 gates cannot run here: list them as "not run (no host)" in the evidence, never as passed.
- Focused tests run from `tools/gomad3` as `env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -tags test_dep <packages>`. `.toolchain/bin/go` is the patched toolchain; `make -C tools/gomad3 toolchain` rebuilds it (needs go.dev access). New test assertions use `require` with whole-value equality.
- Evidence (commands, platform, results, remaining failures) goes under `.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/`. A defect found on the way is recorded for its existing owner, not fixed here.

## Acceptance
- [ ] Seed, choice and simulation completion paths call one private assessment owner for World decoding/seed validation, semantic coverage, choice-feature projection and outcome classification; no second copy of those steps remains in the three callers.
- [ ] The owner takes narrow inputs (no `CampaignSpec`, no strategy flag), performs no process, filesystem, journal, counter or cancellation effect, and returns detached values.
- [ ] Characterization tests written before the extraction pass unchanged after it, covering per strategy: malformed World, seed mismatch, malformed coverage, malformed choices, missing terminal evidence, watchdog, cancellation and simultaneous faults, with exact `HostError.Reason` and precedence.
- [ ] The private interface has direct tests; strategy-level tests are retained; existing comments are preserved.
- [ ] Focused runner tests and `TestPackageArchitecture` pass on darwin/arm64; production line count for `tools/gomad3/runner` is lower than before this task, with both counts recorded.
- [ ] Completion report states "fulfils fn-105.1 (D1)" with evidence paths; fn-105.1 itself is left for the conductor; nothing staged or committed.


## Done summary
The seed, choice-exploration and simulation-exploration completion paths now call one private owner, `tools/gomad3/runner/completion.go`, for World decoding and seed validation, semantic coverage, choice-feature projection and outcome classification. Fulfils fn-105.1 (D1); that task's state is untouched. Nothing is staged or committed; the delta is `task5.diff` and the full record is `task5-evidence.md` under `.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/`.

- Owner: `assessWorld(result, seed, transitionLimit)` returns the validated `execution.Bundle`; `assessCompletion(result, terminal, mode, prepared)` returns `completedExecution` (coverage, choice features and projection, classification) or a `*HostError` with reason `semantic_coverage` or `choice_coverage`. Both are pure and take no `CampaignSpec` or strategy flag. The interface is staged because the seed path publishes a Runner failure artifact on a World error and preserves the partial on a coverage or choice error.
- Characterization came first: `completion_characterization_test.go` passed on the unmodified production files and unchanged after each of the three migrations. It pins reason, cause text, counters, published artifacts, journaled executions and partial states for 16 faults (malformed World, seed mismatch, malformed coverage, malformed choices, missing terminal frame, watchdog, cancellation, eight simultaneous combinations) on all three strategies, plus context cancellation for both exploration strategies. Canonical execution evidence and journal records for fixed identities are byte-identical before and after.
- Direct tests: `completion_test.go` (`TestAssessWorldValidatesTheRecordAgainstItsSeed`, `TestAssessCompletionProjectsCoverageInOrderAndClassifies`). Seven of eight overlay mutants of the owner fail them; the survivor (ignoring `execution.Validate`'s error) has no reachable input.
- Size (rule v2): production Go -63 code lines, -1843 code bytes; `runner` package 7255 -> 7192 code lines; test Go +563 code lines. `size-compare.sh` exits 0 against the task start and the fn-108 baseline. About 13 of the 63 lines are the deleted `novelSemanticProbes`/`addSemanticProbes`, exact copies of `novelStrings`/`addStrings`.
- One unreachable difference: the seed path used to return at once if `SummarizeSemanticProbes(nil)` failed and now treats it like a decode failure. That call cannot fail with no probes; the reason string is unchanged.
- Not done: `recordedWorldForMinimization` keeps its own World steps because its error text differs. `coverage.go`, `minimize_operation.go` and `runner_test.go` are unchanged.
- Gates on darwin/arm64, all exit 0: gofmt, vet, `go test ./runner/... ./artifact/... ./record/...`, architecture tests, `make -C tools/gomad3 validate test-harness world-test test-host` (45 packages ok), `go test ./tools/gomad3sim/...`, `make gomad3-integration-test`, `make gomad3`, `make gomad3-smoke-qualification` (4/4 supported). `world-test` and `gomad3sim` results came from the test cache; no input of theirs changed. Public `go doc` and CLI capture diff against `api-baseline/`: empty. 378 baseline tests in the 11 affected packages keep their disposition. linux/amd64: not run (no host).

baseline: green (focused Quick commands and `TestPackageArchitecture`, before any edit)

stage: impl-review - ran (raw codex bridge on working-tree diff; commits forbidden) (model: gpt-5.6-sol) [round 1 SHIP, no findings; record in task5-review.md]
stage: plan-sync - skipped(config: planSync.enabled != true)

GATE_SKIPPED lines: none.
## Evidence
- Commits:
- Tests: gofmt -l runner (empty), env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go vet -tags test_dep ./runner/..., env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./runner/... ./artifact/... ./record/..., env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ., make -C tools/gomad3 validate, make -C tools/gomad3 test-harness, make -C tools/gomad3 world-test (ok, test cache), make -C tools/gomad3 test-host (45 packages ok), go test -tags test_dep ./tools/gomad3sim/... (ok, test cache), make gomad3-integration-test, make gomad3, make gomad3-smoke-qualification (supported=4 completed=4/4), sh api-capture.sh + diff -r api-baseline (empty), sh size-compare.sh task5-size-files-before.txt task5-size-files.txt (exit 0, production -63 code lines), go test -overlay <mutant> ./runner -run 'TestAssess|TestCompletion' (7 of 8 mutants red), linux/amd64 gates: not run (no host), baseline: green (focused Quick commands + TestPackageArchitecture pre-edit)
- PRs: