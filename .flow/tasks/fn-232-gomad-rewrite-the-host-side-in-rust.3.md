---
satisfies: [R3, R14]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.3 Rust artifact store: new lineage, staged publication, budget, target pool

Touches: tools/gomad/rust/gomad/src/evidence/store/{mod.rs,publish.rs,open.rs,lineage.rs,pool.rs}

## Description
Implement a new storage lineage, private staging directories in the store root managed with std and existing pinned helpers, retained-bytes accounting, maximum-bytes refusal, shared target pool with intentional module hard links or copies as today, and operating-system file locks through existing pinned rustix/filesystem helpers where available, preserving existing contention, ownership and release semantics rather than a homemade lock-file protocol. Publication must use a real atomic no-replace directory rename: Linux renameat2/RENAME_NOREPLACE and Darwin renamex_np/RENAME_EXCL, matching the existing platform semantics. Staging lifetime management is not a directory-publication primitive; no file-persistence, hard-link or check-then-rename fallback. Fail closed if the required exclusive rename is unsupported; no new dependency without approval. Sync payloads, envelope/manifest and staging directories before rename, then store/created parents before returning a reference. Authenticate collisions through the shared opener and reuse only equivalent identities; otherwise apply the existing deterministic collision policy without overwriting. Collision reuse also requires the durable store sync.

Implement the shared authenticated opener used by all artifact consumers: pin a real 0700 root handle and verify its identity across open, reject symlink/substituted roots during open, and resolve contained relative paths through that handle, never reopen its original pathname. Refuse old lineage before manifest decoding. Bound regular 0600 manifest reads to 16 MiB; freeze the validated manifest in the opened object. Reject traversal, symlinks, nonregular files, missing/unlisted entries, nonprivate modes and pre-open/opened identity changes; subdirectories are 0700. Require single links except for the target pool's intentional module links. Payload opens enforce per-call bounds, exact manifest size, EOF and SHA-256 on the same opened handle before returning it for use; mutation or mismatch refuses.

## Acceptance
Use the foundation's fixture-only driver for the execution/store behavior below; no later public verb is a prerequisite. Actual public-verb goldens land with their owning tasks.
Foundation CLI fixtures cover publish/open, stage/rename/sync crash boundaries, equivalent and non-equivalent collisions, budget refusal and old lineage before manifest decoding. Raw integrity is checked before the R14 diagnostic-only golden view. Root-symlink/substitution and post-open pathname replacement fixtures prove reads stay pinned; traversal, escaping symlinks, wrong file/directory modes, forbidden links, missing/unlisted paths, manifest/payload bounds, size changes and digest corruption refuse. Intentional target-pool module hard links still open. No crash produces a referenced half artifact and no reference precedes durability. Fault/fixture controls are excluded from release. Lock ordering gets one unit test that names its invariant.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
