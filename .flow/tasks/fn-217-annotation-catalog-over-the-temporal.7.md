# fn-217-annotation-catalog-over-the-temporal.7 Rename model/temporal/features/ to model/temporal/feature/

## Description
Owner (2026-10-20): rename `model/temporal/features/` to `model/temporal/feature/` (singular, matching the planned `model/temporal/foundation/`). Do it before fn-217.6 so the foundation split and the `api.rs` rewrite land in the final layout.

Touch points found on 2026-10-20 (outside the directory itself):
- `model/temporal/api/build.rs:47` catalog input root.
- `model/temporal/realization/src/lib.rs` `#[path]` attributes for `api.rs` and `temporal.rs`.
- `model/temporal/realization/tests/realization.rs` `include_str!` of the expected snapshots; `realization/tests/ui/supplemental_*.rs` `#[path]` attributes and their `.stderr` files (regenerate with trybuild).
- `model/temporal/realization/README.md`, `model/README.md` (fixture roots sentence), `model/DESIGN.md` (activity model reference).
- `Makefile`: `MODEL_FEATURE_ROOT` and the `temporal.features.` package prefix in `MODEL_UNLISTED_PACKAGES`; check whether that prefix is a path or a Scala package name from the archive before changing it.
- `model/temporal/features/activity/expected/lift.snap.txt` carries the relative path in diagnostics; regenerate.

Verify with `make model-check` and the Makefile lint/lift targets that read `MODEL_FEATURE_ROOT`.

**Files:** the existing feature directory, API build root, Realization module paths and fixtures, Model documentation, Makefile.
**Touches:** `model/temporal/features/**`, `model/temporal/feature/**`, `model/temporal/api/build.rs`, `model/temporal/realization/src/lib.rs`, `model/temporal/realization/tests/**`, `model/temporal/realization/README.md`, `model/README.md`, `model/DESIGN.md`, `Makefile`, `model/temporal/testing/build.rs`, `model/temporal/testing/fixtures.rs`, `model/temporal/api/tests/build_inputs.rs`, `model/design/second-opinion.md`
**Quick:** scoped old-root reference check, workspace format check, Realization trybuild and fixture snapshots, applicable Makefile fixture-root targets.

Keep historical Flow specs, task descriptions, proof evidence and archived Scala package names intact. The old-path check covers active source, build configuration and documentation; exclude `.flow/`, `.worktrees/`, generated build/cache trees, and `model/archived/`. Regenerate path-bearing goldens with their owning tools. Do not change panic behavior or any annotation, Model, protocol or judgment semantics.

## Acceptance
- [ ] `model/temporal/feature/activity/` exists and no `model/temporal/features/` remains; every operational reference listed in the description is updated and the scoped old-root search over active source, build configuration and documentation is empty.
- [ ] `make model-check` passes; snapshot and trybuild `.stderr` files are regenerated, not hand-edited.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
