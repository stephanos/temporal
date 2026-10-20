---
satisfies: [R3]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.3 Rust artifact store: new lineage, staged publication, budget, target pool

Touches: tools/gomad/rust/gomad/src/evidence/store/{mod.rs,publish.rs,open.rs,lineage.rs,pool.rs}

## Description
Implement the store with a new storage lineage marker, staging via tempfile in the store root, fsync, no-replace rename (persist_noclobber), retained-bytes accounting, maximum-bytes refusal, shared target pool with hard links or copies as today, and open-time lineage check that refuses any Go-lineage artifact with the unsupported-format error before decoding the manifest. Lock files via fd-lock.

## Acceptance
Publish and open round trip through the CLI diffed against goldens; a determinism-fixture fault between stage and rename leaves no referenced half artifact (store state golden); budget exceeded returns the existing status; a Go-lineage artifact directory is refused without reading its manifest; two concurrent publishers of the same signature produce the golden store state. Lock ordering gets one unit test that names its invariant.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
