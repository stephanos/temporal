---
satisfies: [R1]
---
# fn-190-gomad-wasm-checkpoint-restore-experiment.2 Crate snapshot/restore boundaries and the image dump

## Description
wasm-sim and wasm-model each expose snapshot() -> Image and restore(Image) for their own state (clock run state and pending folds, network buffers and connections, volume bytes and dirty state, simulation controller; namespace contents, tape cursor, seeded draw state, descriptor offsets and rights, evidence accumulators), each with a boundary test: snapshot, mutate, restore, compare commitment. Add --dump=image to the Rust replay inspection path writing ARTIFACT.inspection/image/ with linear memory, globals, tables, remaining fuel, the crate snapshots and a manifest (sizes, boundary, capture wall time). Refuse cluster artifacts, artifacts without inspection metadata, non-inspectable stops, and a capture that would need inspection allocations first. No restore entry point in replay or recording paths. Depends on .1.

## Acceptance
Boundary tests in both crates; image dump goldens for the fixture at the three boundaries; every refusal case tested; existing replay and recording evidence unchanged.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
