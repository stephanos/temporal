---
satisfies: [R2, R6]
---
# fn-100-gomad-f6-a-package-level-functional.4 Run make gomad3-qualification with zero failures and record F6

## Description
Full set on darwin; infrastructure_errors 0, failed 0; update F6 status.

## Acceptance
- report clean, F6 status updated

## Done summary
`make gomad3-qualification` now completes on darwin/arm64 in one uninterrupted run: 28 of 28 workloads supported, 0 failed, 0 infrastructure errors, expectations met, and the darwin CI jq assertion true. F6 is recorded as done on darwin/arm64 in `.plans/GOMAD_MILESTONES.md` (Status and Work tracking).

**Why the run now fits on disk.** The set report carries no artifact paths. Its per-seed evidence (choice coverage, artifact and trace bytes, replay result) is projected from the retained Campaigns right after each seed's `gomad qualify`, and that command has already replayed every success. After that point the Campaigns are not needed. `qualify-set --prune-qualified-artifacts` (Makefile: `GOMAD3_QUALIFICATION_PRUNE=0|1`, default 0, any other value rejected) deletes a seed's Campaigns only when all of these hold:
- the seed is `qualified`
- the workload has `replay_successes` set
- every execution replayed with `match`

Pruning keeps the seed's qualification report. The checkpoint records the seed as `artifacts_pruned` before anything is deleted. Deletion goes through `os.Root`, so it cannot escape the artifact root, and it refuses symlinked or non-`v1/campaign-*` paths. The report records `qualified_artifacts_pruned`, and `OpenReport` rejects a pruned seed that is not qualified, replayed, and matching. A pruning failure stops the run with status 3. The default behavior is unchanged, so CI still retains and uploads everything.

**Tests** (tools/gomad3/qualification/set/prune_test.go, plus the qualify-set CLI flag in cli_test.go):
- `TestPruneQualifiedCampaignsRemovesOnlyReplayedCampaigns`
- `TestPruneQualifiedCampaignsRefusesUnverifiedOrEscapingEvidence` (not qualified, no replay, diverged replay, outside root, root layout, artifact outside campaign, symlink)
- `TestRunRecordsPruningAndKeepsUnqualifiedSeedArtifacts`
- `TestRunKeepsCampaignsOfQualifiedSeedsWithoutReplay`
- `TestRecordSeedCheckpointsPruningBeforeDeletingCampaigns`
- `TestValidateSetReportRejectsPruningWithoutReplayedQualification`

Each test was confirmed red with its guard disabled.

**Run** (darwin/arm64, toolchain `bb4304eb…`, manifest `sha256:e10842d4`, code at 47269f7a12):
- `make gomad3-qualification GOMAD3_QUALIFICATION_PRUNE=1` ran for 31 min with rc=0.
- Retained artifacts peaked at 340 MiB, against the ~11 GiB the report's `artifact_bytes` totals. Free disk never went below 10.7 GiB.
- 56 of 56 seeds replayed with none diverged, and all 56 were pruned.
- Every seed report has outcome `exit/success` and matching evidence across both repetitions.
- The review fixes in c66a2762e8 change only the pruning order and add the no-replay skip. Every temporal.json workload sets `replay_successes`, so the run at 47269f7a12 exercised the same pruning decisions.

**F6 status text:**
- Per-suite table: decisions and choice records for both seeds, virtual time, peak goroutines, wall time, watchdogs 0, denials 0. Every slice suite has 17 transcript records. The decision counts match tasks .1 and .2.
- No evidence divergence on darwin.
- The linux explanation is written as an inference, with the platform-neutral fixes listed and linux not re-measured.
- 10 of 10 suites qualify.
- `required_probes` and the `finding` field are described.

**Artifacts and reports.** Retained artifacts are deleted. The report and all 56 qualification reports are copied to `scratchpad/f6t4/` (`temporal-qualification-set.json`, `qualifications/`, `table.tsv`, `make.log`, `disk.log`).

**Not done:** memory capture failed because memory is not initialized (`flowctl memory init`). Follow-up: F7 and CI macOS runners could use `GOMAD3_QUALIFICATION_PRUNE=1` when disk is short. The CI workflow is unchanged.

stage: impl-review - ran [2026-09-28T16:33..2026-09-28T16:47] (codex fan-out: round 1 three NEEDS_WORK draws with 2 findings, no-replay abort and prune-before-checkpoint, both fixed in c66a2762e8; that round was refunded as head_moved because the fixes were committed before finalize; the re-dispatched fan-out on c66a2762e8 returned SHIP x3, finalized SHIP)
## Evidence
- Commits: 47269f7a12d684f247624e37108badd65be6a728, 12a22658f82bf3a4fac3d09b1cc4b3f40fffe32d, c66a2762e8cc206d4f99ca5540a9933c7c6d783c
- Tests: baseline: none (spec defines no Quick commands); focused pre-edit baseline green: go test ./qualification/set/ ./cmd/gomad/internal/cli/ (tools/gomad3), go test -count=1 . ./qualification/... ./cmd/gomad/... (tools/gomad3, at c66a2762e8), make gomad3-qualification GOMAD3_QUALIFICATION_PRUNE=1 (darwin/arm64, at 47269f7a12, manifest sha256:e10842d4): rc=0, 28/28 supported, failed 0, infrastructure_errors 0, expectations_met, darwin CI jq assertion true
- PRs: