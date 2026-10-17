---
satisfies: [R1, R2, R3, R4]
---
# fn-189-gomad-sync-with-upstream-temporal-main.1 Merge upstream temporalio/temporal main into gomad and requalify

## Description
Fetch github.com/temporalio/temporal main, merge it into gomad (merge commit, not history rewrite: retained evidence binds existing SHAs), resolve conflicts preserving Gomad seams, reconcile packs/pins, rerun default host gates and the WASM qualification ladder, record identity rotations and the handover.

The owner requested wrapping the current in-flight batch and then stopping on 2026-10-17. The pending upstream merge at `48a1e5ff025e55b3d4d0f44247423d4c279ff636` is resolved without a commit, following the owner requirement to leave commits to them. The [source checkpoint](../artifacts/fn189-upstream-sync-2026-10-17.md) records resolutions, exact adapter upgrades, focused repairs, source review and current check results. fn-189 acceptance remains incomplete until the outstanding WASM gates and pending merge are resolved. No later fixture or profiling task was started.
## Acceptance
R1-R4 of the spec pass on the merged candidate with one handover.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
