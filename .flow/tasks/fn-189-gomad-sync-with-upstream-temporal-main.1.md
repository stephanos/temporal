---
satisfies: [R1, R2, R3, R4]
---
# fn-189-gomad-sync-with-upstream-temporal-main.1 Merge upstream temporalio/temporal main into gomad and requalify

## Description
Fetch github.com/temporalio/temporal main, merge it into gomad (merge commit, not history rewrite: retained evidence binds existing SHAs), resolve conflicts preserving Gomad seams, reconcile packs/pins, rerun default host gates and the WASM qualification ladder, record identity rotations and the handover.

## Acceptance
R1-R4 of the spec pass on the merged candidate with one handover.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
