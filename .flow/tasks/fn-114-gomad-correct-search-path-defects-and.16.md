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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
