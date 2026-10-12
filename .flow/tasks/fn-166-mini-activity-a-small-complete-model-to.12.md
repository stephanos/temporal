# fn-166-mini-activity-a-small-complete-model-to.12 Model Make targets run without held API generation (cached API jar)

## Description
Owner decision, 2026-10-12 (part 2 of the change-scoped model loop; runs before fn-141; authorizes this Makefile change outside `model/`).

Problem: the scoped Make targets (`make model FEATURES=...`, `make model-mini`) first regenerate the API jar (`build/api-scalapb.jar`), which needs owner-held linked protobuf generation (`make protoc`). So every worker falls back to hand-written scala-cli commands, loses `FEATURES` scoping and re-derives its proof steps.

Do: make the model-side Make targets run without the held generation step whenever a usable API jar already exists (cached under `model/build/`, or produced once by an explicit opt-in target), and fail with one clear message naming the opt-in target when none exists. Never run the held protobuf/linked-API generation implicitly, never change Go or proto sources, and keep the canonical full route (with regeneration) available under its existing name for the sweep. Record the exact commands workers now use.

## Acceptance
- `make model FEATURES=activity_mini` (and `make model-mini`) completes its Scala stages without invoking held protobuf generation, using the cached API jar; Go-check stages that remain held are skipped with a named message, not reported as passing.
- With no API jar present, the target fails fast naming the opt-in target.
- Stage wall times recorded (compile, lift, equivalence) with warm caches.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
