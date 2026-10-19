---
satisfies: [R1]
---
# fn-223-one-lifted-source-form-the-lifter-reads.1 Trait-form shared fixtures: attribute rewrites or checker-only moves, with a disposition table

## Description
Survey the eight trait-form shared fixtures under model/umpire/tests/fixtures (sample_kettle_trait, choices, edge_checks, vacuity, shared_starts, progress_graph, deep_chain, interleaved_claims). For each, decide attribute rewrite, checker-only move to umpire/checker/tests/fixtures, or removal (only if another fixture pins the same behaviour on both backends). Rewrite the expressible ones as #[machine] mod Models keeping behaviour goldens byte-identical up to explicitly listed spelling substitutions; move the rest with checker goldens unchanged; sample_kettle_trait becomes checker-only as the independent engine reference. Record the disposition table (fixture, disposition, reason or surface gap) in model/DESIGN.md's TLC section. Verify make model-check and the required TLC run (UMPIRE_BACKENDS=require) and paste both results into the done summary. Does not extend the attribute surface; gaps are recorded only. Starts after fn-215 is closed in umpire's history.

## Acceptance
- [ ] Every trait-form shared fixture has a disposition row in model/DESIGN.md
- [ ] Attribute rewrites keep report/rows/states/paths/tlc goldens byte-identical except for listed spellings
- [ ] Checker-only moves keep checker goldens byte-identical
- [ ] make model-check and UMPIRE_BACKENDS=require runs pass

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
