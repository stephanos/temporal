---
satisfies: [R1]
---
# fn-240-gomad-context-sized-rust-crates-with.1 Dependency survey and crate cut table

## Description
For every top-level module of rust/gomad and wasmhost, list crate:: edges and wasmtime imports; derive the cut table; decide whether wasmhost joins the rust/ workspace or stays a path dep (keep Go build/helper artifact path unchanged); record cargo build --timings baseline clean+incremental. Artifact: .flow/artifacts/fn-240/survey.md. Ordering: independent of fn-232.

## Acceptance
Survey artifact lists every module edge; cut table derived from it; build-time baseline recorded; no code changes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
