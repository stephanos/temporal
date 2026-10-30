Coordinator format repair preserves the contracts draw P2; all three draws returned SHIP. Optional FYIs remain in raw draw files.

## Issue 1
- **Severity**: P2
- **Title**: Source-mutant fixture direct frontend imports need an explicit module contract
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: model/umpire/dev/fixtures/Cargo.toml:21
- **R-IDs**: [R2]
- **Problem**: The module map row for `umpire-fixtures` (`.plans/UMPIRE_MODULES.md:48`) says fixtures "use author and test support". This change adds `umpire-front` and `syn` as direct dev-dependencies so the test can call `umpire_front::read` on the mutant source. No existing route exposes the front tree from test support (the `model` field on `umpire_tla::Lifted` is private), so the deviation is understandable, but it is a stated-contract inconsistency on the module boundary that the other mutant fixtures avoided (`optional_ticks` mutates the embedded tree; `phased_rejection` goes through `bound().lift_source`).
- **Suggestion**: Either record the exception in the module map row, or expose a small `Bound::read_source(file, source) -> umpire_front::Lifted` in `umpire-testing` and drop the two dev-dependencies from the fixture crate.

Unaddressed R-IDs: []

Classification counts: 1 introduced, 0 pre_existing.

<verdict>SHIP</verdict>
