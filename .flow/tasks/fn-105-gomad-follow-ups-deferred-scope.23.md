---
satisfies: [R23]
---
# fn-105-gomad-follow-ups-deferred-scope.23 D23: correct migration idempotency tests without changing the contract

## Description
Decision on 2026-09-30: preserve the current migration contract and require correction of TestScheduleMigrationTestSuite/TestScheduleMigrationV2ToV1Idempotent. A repeated request succeeds while migration is pending; after migration closes the CHASM schedule, the current contract returns a closed error. The test currently relies on its second call beating the side-effect task. Make state preconditions explicit and verify pending and closed behavior without altering production migration semantics.

## Acceptance
- Explicitly establish the pending-migration state before verifying that a repeated migration request succeeds without duplicate migration work; use existing state/side-effect controls or an isolated contract test instead of timing assumptions.
- Explicitly establish completed migration and the old schedule's closed state, then verify the existing closed-state response separately.
- Preserve current production/API behavior and migration/no-duplicate-work coverage; no broader completed-migration retry contract, Gomad-only source rewrite, or sleep-based ordering workaround.
- Verify the corrected tests under native Go and Gomad on seeds 11 and 17 with recorded commands, platform identity, and outcomes.
- Remove the migration-idempotency skip from the source qualification generator and regenerate its manifest after verification passes. A still-skipped or merely classified test cannot close this required correction.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
