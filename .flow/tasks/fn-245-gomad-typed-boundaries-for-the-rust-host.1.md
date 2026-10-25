---
satisfies: [R1, R3]
---
# fn-245-gomad-typed-boundaries-for-the-rust-host.1 One authenticated-read abstraction in gomad-evidence

## Description
Add gomad_evidence::io with BoundedRead (bounded read from an already pinned handle, existing boundary message), Digest (the one SHA-256/hex helper plus the canonical-bytes variant) and Authenticated<T> (bounded read, digest, compare to expected identity, typed decode). Route every host-side caller through it and delete the local copies: campaign (8 Sha256 sites), qualify (4), measurement (3), sweep (2), replay (2), install (2), minimize, stress, cli, and the fixture_* modules (one each). The wasmhost identity sites stay. Record before/after counts of Sha256 constructions, fn read/digest/authenticate definitions with exact revisions. Ordering: after fn-240.2 (gomad-evidence exists) and before fn-240.9 moves the callers; cross-spec order recorded here per house convention.

Crate amendment (owner, 2026-10-24): build `BoundedRead`/`Authenticated<T>` on `cap-std` `Dir` handles (pinned `=`), replacing the hand-rolled `openat`/`NOFOLLOW`/`statat` sequences (~184 sites in ~25 files at 2026-10-24). The five local `fn hex` copies fold into `Digest`.

Complexity amendment (owner, 2026-10-24): `gomad_evidence::io` also owns durable publication (temp file, write, fsync, rename, fsync directory), on `cap-std` plus `tempfile`'s `persist`. It replaces the 17 local publish/atomic-write helpers and the hand-written write-fsync-rename sequences in about 10 production files (2026-10-24).

Throughput amendment (owner, 2026-10-24): runs after fn-240 completes (spec dependency). flowctl cannot enforce the earlier "after fn-240.2, before fn-240.9" interleave across specs, and moving local helpers mechanically before replacing them costs little.
## Acceptance
grep -rl 'Sha256::' over the host crates lists at most gomad-evidence::io plus the two engine identity sites; no fn read/digest/authenticate definition remains outside gomad-evidence::io; identity goldens, the fn-240 flip-byte test and cargo test --workspace unchanged and green; counts in .flow/artifacts/fn-245/counts.md.
- Pinned-directory access goes through `cap-std`; no `rustix::fs::openat`/`OFlags::NOFOLLOW` call remains in the migrated callers; no local `fn hex` remains.
- No local publish/atomic-write helper or hand-written write-fsync-rename sequence remains outside `gomad_evidence::io`.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
