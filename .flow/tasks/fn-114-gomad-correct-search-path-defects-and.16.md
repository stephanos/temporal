# fn-114-gomad-correct-search-path-defects-and.16 Finish shared-target byte accounting inside a campaign and in merged-record validation
## Description
Follow-up to task 10 (R7/E2). Task 10 states the rule on `artifact.RetainedBytes`: per-artifact stored bytes keep the target, and a sum over artifacts kept together counts a shared target once. Two checks do not follow it yet, and the task 10 review left four notes open.

**Size:** S
**Files:** `tools/gomad3/runner/runner.go`, `tools/gomad3/runner/retention.go`, their tests, `tools/gomad3/runner/internal/campaign/merge.go`, `open_campaign.go`, `tools/gomad3/artifact/store.go`, `tools/gomad3/runner/inspect.go`
**Touches:** [tools/gomad3/runner/runner.go, tools/gomad3/runner/retention.go, tools/gomad3/runner/*_test.go, tools/gomad3/runner/internal/campaign/**, tools/gomad3/artifact/**, tools/gomad3/runner/inspect.go, tools/gomad3/cmd/gomad/internal/cli/**]

### Approach
- The `--success-bytes` limit inside one campaign charges every retained success its full stored bytes, while merge and the corpus count a shared target once. Decide from the code whether the in-campaign sum can follow the rule without changing any recorded identity, journal bytes, or the per-artifact stored-byte value. If it can, apply the rule and keep the limit failing visibly; if a recorded value would have to change, leave the check as is and state the difference on `RetainedBytes` and in the done summary for task 14's docs.
- `validateMergedCampaign` widens the limits it hands `validateCampaign` to get past the per-execution success-byte bound. Replace the widening with an explicit option on `validateCampaign` that skips only that bound, so every other limit is checked with the real values.
- State on the `RetainedBytes` contract that merge accounts a single store holding all merged evidence, whether or not the shard stores share the file on disk.
- Correct the `TargetReport.Sharing` comment to what `inspect` checks (the link count), or check pool membership if that is cheap and exact.
- Add a test that opens a merged record written before `target_sha256`/`target_bytes` through `OpenMergedCampaign`, using retained bytes from the commit before task 10.

## Acceptance
- [ ] The in-campaign success-byte limit either counts a shared target once with unchanged recorded identities (pinned by a retained value), or its difference from merge and corpus accounting is stated on the contract with the reason
- [ ] `validateMergedCampaign` passes real limits; only the per-execution success-byte bound is skipped, by an explicit option, and a test shows another limit still rejects a merged record
- [ ] A merged record written before task 10 opens through `OpenMergedCampaign`
- [ ] `GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host` and `make -C tools/gomad3 validate` pass

## Done summary
Merged-record validation now checks a merged campaign against its real limits and skips only the execution success-byte bound, by an explicit option. The success-byte limit inside one campaign is left counting every artifact in full, and the `RetainedBytes` contract says so with the reason (R7/E2 follow-up to task 10).

What changed:

- `validateCampaign` takes `executionSuccessBytesBounded` or `executionSuccessBytesUnbounded`. `OpenCampaign` passes the first, `validateMergedCampaign` the second together with the record's own `artifacts` limits. The widening is gone.
- One behavior change follows: a merged record whose `total_bytes` is not the sum of its two byte limits is now rejected even when its executions report more success bytes than the limit. The widened limits recomputed the total and hid that. A record `gomad merge` writes always has a consistent total.
- `artifact.RetainedBytes` states who sums with it and what each takes as shared: a corpus by the pool link on disk; a merged campaign by target SHA-256, as one store holding all merged evidence whether or not the shard stores share the file; a campaign's own limits not at all.
- `TargetReport.Sharing` says what `inspect` reads, the link count. The pool is not looked up, so a pooled store's fallback copy reads `private`.
- A merged record published at b5b498004 (before task 10) is retained under `runner/internal/campaign/testdata/pre-target-evidence-merged` and opens through `OpenMergedCampaign`.

In-campaign `--success-bytes`: left as is. For task 14's docs, a campaign's byte limits bound what its artifacts take as standalone copies, which is more than they take in a store that shares their target; merge and the corpus count a shared target once. The reasons, from the code:

- `campaign.json` records `retained_success_bytes`. `validateCampaign` requires it to equal the sum of the executions' `success_artifact_bytes` and to stay within `artifacts.success_bytes` of the same record, and `validateResumeExecutions` holds the same sum to the plan's limit.
- Counting a shared target once makes that recorded sum either a different number for the same campaign (which changes `campaign.json` and the `campaign_sha256` a merged record binds) or a number above its recorded limit, which every reader of `gomad3.campaign/v1` rejects.
- Neither the campaign record nor an execution record names the target size or says whether an artifact linked to the pool or fell back to a private copy, so a record-only validator cannot recompute the bound. The retention decision and `Store.MaximumBytes` are also fixed before publication, and the link is known only after it.
- Applying the rule would also need `runner/choice_exploration_campaign.go`, `runner/simulation_exploration_campaign.go`, and `runner/resume.go`, which are outside this task's Touches.

Tests, by acceptance item:

- In-campaign difference pinned: `TestRunCountsASharedTargetInFullAgainstTheSuccessByteLimit` (two successes linked to one pool entry do not fit a limit half a target short of their stored bytes).
- Real limits, one bound skipped: `TestOpenMergedCampaignChecksItsOtherLimitsWhenExecutionsReportMoreSuccessBytes` (the record opens; then an inconsistent total, too few success artifacts, and zero transcript bytes each reject it). The first case was red before the change.
- Older record: `TestOpenMergedCampaignOpensARecordWrittenBeforeEvidenceNamedItsTarget`.

No existing assertion changed; `copyRetainedExplorationJournal` now calls an extracted `copyRetainedRecords` helper. Four single-change mutations each turned a named test red.

Gates on darwin/arm64: `test-host` exit 0 at 92cb85de8 (45 packages, 344 s) and at 6f01faffa (196 s); `validate` exit 0 at the final code; architecture test and vet (host and linux/amd64) exit 0.
GATE_SKIPPED:unittest:green-receipt 92cb85de - Verify at the evidence commits reused the post-commit pass; only .flow files changed after it

Three `test-host` runs were red before the green ones:

- At fe8e14c21, `TestWatchdogDiagnosticReplayUsesCapturedInputs`. Not this task: it fails 5 of 20 focused runs at the base commit and 7 of 20 at fe8e14c21, at the same assertions.
- At fe8e14c21, this task's new retention test in its first form. It kept two successes of one outcome signature, which the store holds as one artifact, and set its limit one byte under stored bytes that vary between runs. Rewritten in 6f01faffa; 30 of 30 focused runs pass.
- At 92cb85de8, `TestRunBoundsActiveExecutionsAndRetentionAtTenAndOneHundredJobs/100_jobs/discard` hit `overall_timeout` at load average 14 to 16 while another session ran `go clean -cache`. It passes focused at the same revision, and the full gate passed on the next run.

Not met or not run:

- The in-campaign limit does not count a shared target once (the acceptance item's second branch is the one met).
- linux/amd64 (no native host); root `make lint-code-fast`; the spec's literal `go -C tools/gomad3 test` Quick command with an unpatched go; any qualification set.

Review: SHIP on the first round from claude-fable-5-1 at high through the `claude` backend (same family as the writer; the reviewer had no shell and read the committed evidence). Two P3 findings stay open:

- `open_campaign.go:152`: move the execution success-byte term out of the long `||` chain into its own `if`.
- `copyRetainedRecords` is a shared helper in `choice_exploration_start_test.go`; move it to a helpers test file.

Found outside this task, not fixed:

- A seed campaign that keeps two successes of one outcome signature stores them as one artifact directory, counts both in `retained_success_bytes`, and publishes a record `OpenCampaign` rejects ("retained success artifact does not match its campaign execution"). Seen with the fake executor, which returns the same result for every seed; whether two real seeds can share a signature was not checked.
- `TestWatchdogDiagnosticReplayUsesCapturedInputs` is intermittent on this host at the base commit.

Evidence: `.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/task-16/`.

stage: impl-review - ran (SHIP, claude:claude-fable-5-1:high, round 1)
## Evidence
- Commits: fe8e14c2156cea78230f8dc38e62807d24c4bb05, 6f01faffa2d1268d99c55861a38c14c65acd31d9, 92cb85de86d179022768b14a758bfce733be0d9d, 7867d51ccda2b306364bfe7a856d966df421052e, bbb2a74168d0b8671b23a51ef40d11a94ca98a74
- Tests: baseline: green via handoff (full test-host verified at b12b15b1 by fn-114-gomad-correct-search-path-defects-and.10; only .flow commits since), GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host (at 92cb85de8: exit 0, 45 packages, 344 s, pinned stock go1.27.1 first on PATH, load average 15 to 17; also exit 0 at 6f01faffa in 196 s), RED, not this task: test-host at fe8e14c21 exit 2 on TestWatchdogDiagnosticReplayUsesCapturedInputs; the same test fails 5 of 20 focused runs at the base commit 9216937b8 and 7 of 20 at fe8e14c21, RED, this task, fixed: test-host at fe8e14c21 exit 2 on the first form of TestRunCountsASharedTargetInFullAgainstTheSuccessByteLimit; rewritten in 6f01faffa (30 of 30 focused runs pass), RED, host load: test-host at 92cb85de8 exit 2 on TestRunBoundsActiveExecutionsAndRetentionAtTenAndOneHundredJobs/100_jobs/discard (overall_timeout, load average 14 to 16 while another session ran go clean -cache); focused rerun at the same revision exit 0 in 13 s; the full gate rerun at the same revision exit 0, GATE_SKIPPED:unittest:green-receipt 92cb85de - Verify at the evidence commits reused the post-commit pass; only .flow files changed after it, make -C tools/gomad3 validate (exit 0 at 7867d51cc, code equal to 92cb85de8), .toolchain/bin/go test -tags test_dep -count=1 -run TestPackageArchitecture . (exit 0 at 6f01faffa), .toolchain/bin/go vet -tags test_dep ./artifact/... ./runner/... ./cmd/... (exit 0; GOOS=linux GOARCH=amd64 exit 0), 4 single-change mutations, each red against its named tests (.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/task-16/mutation-checks.json), red-first: a merged record with total_bytes off its byte limits opened without error at the base code (.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/task-16/red-first.txt), NOT RUN: linux/amd64 (no native host); root make lint-code-fast; the spec's literal go -C tools/gomad3 test Quick command with an unpatched go; any qualification set
- PRs: