# Extract the activity Phase terminal predicate

The owner approved adding only Phase::is_terminal(), with action-specific guards and transitions retained in Model.

## Acceptance Criteria

- **R1:** Phase::is_terminal() returns false for Scheduled/Started and true for Completed/Failed/Canceled/TimedOut. Keep its impl adjacent to Phase and use it in Model::terminal_remains. No transition, action ordering, claim outcome or golden changes.

## Verification

Add an exhaustive six-phase predicate test first; confirm it fails because the helper is absent. Implement the helper and replace only the existing terminal classification. Run the Temporal fixture suite and make model-check, plus git diff --check. Preserve unrelated fn-216, fn-221 and MILESTONES edits.


## Closure context (2026-10-21 backlog audit)

Task .1 shipped the helper in cb3a5f5586 and its verification receipt in b5f13e2ee1. The later completed fn-221 generalized the same classification to umpire_dsl::Phase::closed and shared closure laws. The current activity Model implements that trait and retains terminal_classification_covers_every_phase. The original helper spelling is historical; do not reintroduce it alongside the shared trait. Parent closure records the completed checkpoint and its successor, not a new test run.
