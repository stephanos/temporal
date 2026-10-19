---
satisfies: [R1]
---
# fn-222-extract-activity-phase-terminal.1 Add Phase::is_terminal and reuse in preservation claim

## Description
# Extract the activity Phase terminal predicate

The owner approved adding only Phase::is_terminal(), with action-specific guards and transitions retained in Model.

### Acceptance Criteria

- **R1:** Phase::is_terminal() returns false for Scheduled/Started and true for Completed/Failed/Canceled/TimedOut. Keep its impl adjacent to Phase and use it in Model::terminal_remains. No transition, action ordering, claim outcome or golden changes.

### Verification

Add an exhaustive six-phase predicate test first; confirm it fails because the helper is absent. Implement the helper and replace only the existing terminal classification. Run the Temporal fixture suite and make model-check, plus git diff --check. Preserve unrelated fn-216, fn-221 and MILESTONES edits.

## Acceptance
Exhaustive phase classification passes; terminal_remains uses helper; model-check passes with unchanged goldens and transitions.

## Done summary
Added public const Phase::is_terminal(self) adjacent to the enum and reused it in Model::terminal_remains. Action guards and transitions are unchanged. An exhaustive six-phase classification test failed first with E0599 (missing is_terminal), then all five Temporal fixture tests passed. make model-check passed (fmt, clippy, docs, native fixtures, Wasm builds and both Go hosts). git diff --check passed; no golden edits. Source checkpoint cb3a5f5586. Unrelated fn-216/fn-221/MILESTONES edits preserved. No milestone continuation, push or PR.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: cb3a5f5586
- Tests: expected-red: cargo test -p umpire-temporal-tests --test fixtures terminal_classification_covers_every_phase (E0599 missing is_terminal), CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target mise exec -- cargo test -p umpire-temporal-tests --test fixtures (5 passed), CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target make model-check (passed), git diff --check (passed)
- PRs: