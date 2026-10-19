---
satisfies: [R8, R23, R29]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.22 Name sample fixtures explicitly and rename progress dead-end control

## Description
Owner-approved naming-only checkpoint: rename model/umpire/tests/fixtures/kettle to sample_kettle, kettle_trait to sample_kettle_trait, and kettle_no_heat to progress_dead_end. The sample_ prefix identifies the canonical attribute and manual-trait authoring examples; focused controls are named for the behavior they test. progress_dead_end demonstrates an unmet progress target on a finite maximal path in the assumption-filtered graph, with no enabled internal action to rescue it. Rename its NoHeat wrapper to a behavior-focused name. Update generated-registry module references in dependent fixtures (currently fixture_kettle), wasm source include path, wasm dispatch and Go host expected payload paths, and live documentation. Use the post-.19/.20 source as authoritative, preserve the .21 payload-only snapshot layout, and do not reintroduce symlinks. Preserve fixture semantics and payload bytes; no golden refresh except strictly necessary path attribution in snapshot review metadata. Do not rename other behavioral fixtures or archived Scala files. Reuse build-time discovery rather than adding name aliases or special-case registration. Touches: model/umpire/tests/fixtures/**, model/umpire/wasm/src/lib.rs, model/umpire/wasm/tests/dispatch.rs, model/umpire/hosts/go/host_test.go, model/README.md, model/DESIGN.md, MIGRATE.md. Quick: cargo test -p umpire-tests -p umpire-wasm; make model-check. This task stays todo until predecessors have committed local closures.

## Acceptance
All three old fixture directories are gone and replacements are discovered exactly once; all source consumers and live documentation resolve the new names. Canonical sample source behavior, graph statistics, verdicts, ordering and snapshot payload bytes remain unchanged. progress_dead_end retains its finite maximal path progress-refutation proof and uses behavior-focused wrapper naming. Snapshot metadata remains valid; no fixture symlinks or compatibility aliases are added. Full make model-check passes, including native shared fixtures, stock wasm and Go host parity; record before/after payload identity.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
