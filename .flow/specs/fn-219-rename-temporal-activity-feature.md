# Rename Temporal activity feature directory

## Goal & Context

The owner requested renaming the existing activity_mini feature to activity without changing its behavior. This is a completed standalone rename, not authorization to resume the paused Scala backlog.

## Acceptance Criteria

- **R1:** The active feature is named activity, its realization references follow the rename, and the old activity_mini directory is absent. Model and golden bytes are unchanged by the rename. Errors: a stale active reference, missing moved file or changed Model/golden fails verification; archived Scala names remain historical.
- **R2:** The Model gate and real SDK slice pass after the move. Errors: a failed check leaves the rename incomplete.

## Boundaries

No Model behavior, archived Scala, protocol or milestone-scope changes.

## Decision Context

This replaces an unfilled scaffold with the scope and acceptance already recorded in task .1. Its completed receipt records byte comparisons of all eight moved files, make model-check and make umpire-check-model-slice. The receipt was committed in c7bc39d4ad. Later layout work renamed the parent features directory to feature (cb739765e8); the current location is model/temporal/feature/activity. Do not recreate the former parent spelling.

## Requirement coverage

| Req | Task(s) |
| --- | --- |
| R1 | fn-219-rename-temporal-activity-feature.1 |
| R2 | fn-219-rename-temporal-activity-feature.1 |
