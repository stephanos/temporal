# fn-219-rename-temporal-activity-feature.1 Rename activity_mini to activity without changing behavior

## Description
Move model/temporal/features/activity_mini to activity and migrate active realization references. Preserve archived Scala names and independent Model and golden bytes. This is a standalone owner request; the milestone queue remains stopped.

## Acceptance
Old feature directory absent; all current realization references use features/activity; Model and expected files unchanged byte-for-byte; make model-check and make umpire-check-model-slice pass.

## Done summary
Renamed model/temporal/features/activity_mini to activity, including all eight source and snapshot files. Updated the realization include paths and README. Compared every moved file against HEAD using cmp: all bytes unchanged. No archived Scala or concurrent fn-216/MILESTONES changes modified. Milestone execution remains stopped.

Verification: make model-check passed; make umpire-check-model-slice passed all four live SDK outcomes; git diff --check passed. The initial gate saw an empty old directory left by the patch move; removed only those empty directories and reran successfully. Changes remain uncommitted.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target make model-check, CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target make umpire-check-model-slice, git diff --check, cmp of all eight moved files against HEAD
- PRs: