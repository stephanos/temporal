---
satisfies: [R16]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.3 Give the seed controller one atomic completion transition

## Description


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

Stage 2 (S4). R2/R3 need no task here: fn-108 R6/R7 own shared assessment and retention, and the final evidence task links them. This task covers the remaining stage-2 work: `SeedController` requires callers to pair `FinishAttempt` with a later `RecordSuccess`/`RecordCancelled`/`RecordFailure`, so attempted, active and classified counters can disagree between the two calls.

**External ordering:** start only after the fn-108 R6/R7 tasks are done and verified: `fn-108-gomad-reduce-code-size-without-removing.5` (shared assessment, R6) and `.6` (retention and artifact-input composition, R7) in `tools/gomad3/runner`. Re-anchor the line references below against the post-fn-108 source first. flowctl cannot record a cross-spec task edge, so check `flowctl tasks --spec fn-108-gomad-reduce-code-size-without-removing` before `flowctl start`. The call sites below sit inside the seed completion loop that fn-108 restructures.

**Size:** S/M
**Files:** `tools/gomad3/runner/internal/campaign/controller.go`, `controller_test.go`, `tools/gomad3/runner/runner.go` (call sites).
**Touches:** [`.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-3/source-acceptance-20261008/**`, `tools/gomad3/runner/runner_test.go` (only `TestRunRejectsSuccessfulRetentionWithoutReplayTranscript`), `tools/gomad3/runner/completion_characterization_test.go` (only `TestCompletionFaultsKeepReasonPrecedenceAndEvidence`)] (current acceptance evidence plus the narrowly admitted historical whole-statistics assertion; no product implementation rewrite)

### Approach
- Current protocol: `FinishAttempt` (`controller.go:98`, panics when `active == 0`), `RecordSuccess` (`:106`), `RecordCancelled` (`:110`), `RecordFailure(domain, reason, distinct)` (`:114-138`). Callers: `runner.go:778` (finish), `:818` (cancelled), `:1035` (success), `:1100` (failure, returns cancel-active).
- Replace them with one pure completion transition taking a completion value (success, cancelled, or failure with domain, reason and the explicit distinct-failure count) and returning whether active work must be cancelled. Attempted, active, the classified counter and the failure-policy stop change in that one call.
- Completion without active work is an invariant violation, as today. Decide error-return versus panic by what `runner.go` can handle without changing its error precedence, and state the choice in the summary.
- Between `runner.go:778` and the classification sites the loop performs evidence work that can return early. Characterize first what `Statistics()` reports on each early-return path, then keep those observable counters identical; the atomic transition must not turn an early return into a lost attempt.
- Keep `Next`, `Stop`, `Stopped`, `Active`, `Done`, `Finalize`, `Statistics` and ordinal scheduling untouched. The controller stays pure: no host calls, no publication.

### Investigation targets
**Required:**
- `tools/gomad3/runner/internal/campaign/controller.go` (whole file, 170 lines)
- `tools/gomad3/runner/internal/campaign/controller_test.go`
- `tools/gomad3/runner/runner.go:760-830,1025-1110` (re-anchor after fn-108)
**Optional:**
- `tools/gomad3/runner/runner_test.go:671-712` (first-failure and budget policies), `:1324-1440` (cancellation, resume counters)

### Quick commands
```bash
cd tools/gomad3
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./runner/internal/campaign -run 'SeedController|Controller'
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./runner -run 'TestRun'
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./runner/...
```

### Constraints
- No `git add`, commit, stash or worktree: the user owns commits. Record `"commits": []` in the `flowctl done` evidence and say so in the summary.
- No new third-party dependency. `tools/gomad3/go.mod` requires only `golang.org/x/mod`, so testify is unavailable inside `tools/gomad3`: follow the existing `t.Fatalf` style with whole-value comparisons there. In the root module (`tools/gomad3sim`, `tools/gomad3integration`) use `require` with `Equal`/`EqualValues`.
- Preserve existing comments with their owning code, CLI grammar/defaults, canonical bytes for fixed supplied identities, and error precedence/classification.
- This host is `darwin/arm64`. `linux/amd64` gates cannot run here: list them as incomplete in the done summary, never claim them.
- fn-105 D12/D14 replay-divergence dispositions stay unchanged. Attribute a failure to those owners with retained evidence instead of relaxing an expectation.
- Run tests with `-tags test_dep`. Baseline the Quick commands before editing so a pre-existing failure is not attributed to this task.
- Evidence and decision records go under `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/`.
### Current retained-source acceptance admission (2026-10-08)

Retain the integrated atomic-completion implementation; no product rewrite is admitted. Formal dependency fn-109.2 reached verified Done after current source acceptance and three-axis SHIP; external fn-108.5/.6 are Done and verified. Reconstruct all original task3 postimages and bind current controller/callers, original whole-statistics and early-return characterization to actual later owners, including rebased completion constructors and the already accepted fn-109.28 exact two-site invariant-panic exception. Preserve every unaffected counter, failure-policy stop/cancel distinction, admission/ordinal scheduling, early unclassified attempt, comment, fixed-identity byte, error precedence and D12/D14 disposition. The first fn109 baseline remains6782b55 plus dirty fn108; do not substitute later trees. Obtain current portable coverage, generator validation, both-supported-source-set static input bindings, task-owned lint and source-preservation evidence; global residuals remain with correction/aggregate owners without waiver. Only fn-114.11/.12 approved migrations and fn-109.28's two exact sites receive the existing bounded exceptions. Write only this source-acceptance evidence directory, except the following root-admitted regression assertion (2026-10-08): retain the exact existing `TestRunRejectsSuccessfulRetentionWithoutReplayTranscript` trigger and error assertions, capture its returned summary, and restore the original independently specified whole `CampaignStatistics{Attempted: 1}` assertion lost during later consolidation. This is additive coverage, not a changed expectation or production behavior. The same bounded restoration also applies to the existing `World seed mismatch` and `malformed semantic coverage` rows in `TestCompletionFaultsKeepReasonPrecedenceAndEvidence`: add an optional expected whole-statistics field, set only those two original literals (`{Attempted: 1, DistinctFailures: 1}` and `{Attempted: 1}`), and compare the existing `observeSeedCompletion` projection only for StrategySeed after the untouched existing completion assertion. Preserve every current input, fixture, error/precedence assertion, exploration expectation and comment. Do not count these native-guarded assertions as executed on this host. Retain the canonical restored-assertion outcome. If the unmodified native-profile guard prevents reaching that assertion on this unsupported host, record it as unexecuted; its native-dependent execution and Runner-specific mutation checks remain with the already named native owners. Independently exercise the pure Controller whole-statistics tests with artifact-local stock-Go overlays that remove the unclassified Attempted increment or wrongly classify it as success, retaining actual failing receipts and canonical GREEN. Those controls prove Controller behavior sensitivity only, not execution of the Runner publication assertion. No production mutation, guard bypass, platform spoof or portable pass inferred from a native refusal is permitted. Bind the pre-edit test bytes, exact minimal test delta and fresh post-edit coverage/lint/generation/static identities. Any further gap requires root admission before expansion. No fixture recapture, source assertion relaxation, platform shim, native-runtime bypass, native pass, PR, push or CI work. Root owns lifecycle/MILESTONES/Git/review/completion; worker owns the single serialized Go/generator lane and returns terminal handover/evidence without verdict or completion.

### Current source review context

Review the original controller transition, full current completion loop and consumers, the narrowly restored whole-statistics assertions, frozen worker evidence and independently rerun current gates using `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-3/conductor-source-acceptance-20261008/source-review-context.md`. Native-coupled assertions remain explicitly unexecuted, transferred native gates remain deferred, and global lint residuals remain unwaived.
## Acceptance


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- [x] `SeedController` exposes one completion transition; the `FinishAttempt` plus `Record*` pairing no longer exists and `runner.go` calls the transition once per completed job.
- [x] Controller tests cover success, cancellation, watchdog, replay divergence, distinct and duplicate failures, resume-seeded counters and each failure policy (`all`, `first`, `budget`), comparing whole `CampaignStatistics` values.
- [x] Completion without active work is rejected as an invariant violation; no path leaves attempted and classified counters partially updated.
- [x] Ordinal scheduling, stop reasons and every counter reported by existing runner tests are unchanged.
## Done summary
SeedController now exposes one pure `Complete` transition. Runner completes each received job once, including an explicit unclassified outcome on early host/evidence errors. Active, attempted, classified and failure-policy counters update together. Existing ordinal scheduling, comments, error precedence and stop semantics are preserved: first cancels active work; budget stops admission only. Inactive completion retains the existing invariant panic.

Whole-statistics controller tests and the pre-refactor early-return characterization pass. Final focused Runner tests, architecture, vet, CLI build, formatting and diff checks pass on darwin/arm64. Parent independently verifies the original byte snapshots, final hashes, comments and patch, and runs final controller tests (0.296s).

The initial full host gate exited 2 on an unchanged execution watchdog test; Runner and all other host packages passed. Its exact selector, test family and entire execution package subsequently passed on the same frozen source/environment. Dependency inspection excludes Runner/controller from that test closure. The initial failure and unknown trigger remain retained; this is combined package evidence, not a full-host pass. After a controller-only lint cleanup, the affected focused and boundary checks pass. Scoped lint remains red with 430 existing findings; root lint retains its documented nested-module loading failure.

Independent codex:gpt-6-sol:high review returned SHIP at 2026-10-03T11:19:37.270978Z with zero introduced findings and R16 met. R18/R19 full-spec and native linux/amd64 qualification remain open in task 21. Review is same-family with fresh context. See `handover.json`, `task-only.patch`, `source-pre.json`, `source-post.json`, `parent-source-verification.json` and `working-tree-review.json` in this task directory.

Nothing was staged, committed or pushed; the user owns commits.

stage: plan-sync - skipped(config: planSync.enabled != true)
Tracker sync: n/a (bridge inactive)
## Evidence
- Commits:
- Tests: Pinned patched Go, stock helper compiler, classic GC, test_dep: final controller focused gate exit0 0.305s; parent independent final TestSeedController gate exit0 0.296s, Final focused Runner gate exit0 8.341s (focused-final.log); TestRun selector exit0 45.838s before valid-kind lint simplification, GOFLAGS="-tags=test_dep -count=1" make -C tools/gomad3 test-host: exit2 retained, only unchanged execution watchdog test failed; Runner and other host packages passed, Same-source/environment go test -tags test_dep ./runner/internal/execution: exit0 68.311s; exact watchdog selector and complete family also pass, TestPackageArchitecture: exit0 1.375s (architecture-final.log); Runner vet and CLI build exit0, gofmt and git diff --check exit0; task-only patch reverse-check and final source hashes verified, Scoped Runner lint exit1: 430 existing findings retained, no introduced diagnostic identified; root lint limitation retained
- PRs:
