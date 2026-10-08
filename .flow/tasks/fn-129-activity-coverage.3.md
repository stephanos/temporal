---
satisfies: [R3, R5]
---
# fn-129-activity-coverage.3 Reset: deferred reset with keepPaused

## Description
**Batch:** deferred Model batch (see MILESTONES.md, Deferred, fn-128). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at that batch's single regeneration against its baseline (the tree at the DSL batch's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.
R3, R5 (comparison P2-5). A `Control.reset(keepPaused)` class; a `resetRequested` phase for a reset deferred while an attempt is held, and its application when the attempt settles, as the Go model's `applyDeferredReset` does; `keepPaused` kept in the state. Properties: keep-paused (a reset with `keepPaused` of a paused activity leaves it paused) and the Cancel > Reset > Pause precedence, extending fn-128's `cancelIsNotUndone`. Recompute the state count and Limits.

Realize through the server's reset API for standalone activities (the task confirms its name and fields), with at least one live Query (e.g. a reset while an attempt is held that applies on the attempt's failure).

## Acceptance
- [ ] Reset rows match `model.go` (deferred apply, keepPaused, precedence), citing it.
- [ ] The keep-paused and precedence Properties are checked and hold.
- [ ] At least one live Query is realized and its Case ran once.
- [ ] New Cases listed; the spec's Verification gates pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
