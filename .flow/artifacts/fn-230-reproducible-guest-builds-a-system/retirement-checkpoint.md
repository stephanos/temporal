# fn-230 retirement checkpoint

Closure `5c60587318` is committed on `umpire`; both tasks are done and parent
completion status is ship. All fn-230 workers and review commands have ended.
The metadata and unwind worktrees were already retired normally. Remaining
registered worktrees belong to other or uncertain work and remain protected.

The read-only audit at `.flow/tmp/fn230-retirement-audit.md` identified four
independent Cargo proof targets totaling about 1.85 GiB and four clean proof
clones totaling about 1.24 GiB. The clones contain differing ignored descriptors;
no clone removal was attempted. Proof binaries, logs and source pins remain.

The conductor compared both stock/Temporal proof pairs, checked the latest pair
against the committed asset, and generated `.flow/tmp/fn230-proof-retained.sha256`.
The first `cargo clean --target-dir` refused because the target lacks a valid root
`CACHEDIR.TAG`. Cargo aborted before deletion. The conductor did not fabricate a
tag or bypass the guard. A profile-scoped dry run also refused the untagged root.

Each target's nested `wasm32-wasip1` directory already had a valid cache tag.
An exact nested-directory dry run succeeded, followed by four successful pinned
Cargo cleanup commands. The selected directories were:

- `.flow/tmp/fn2302-reproduction-proof.JrgaIS/independent-target-a/wasm32-wasip1`
- `.flow/tmp/fn2302-reproduction-proof.JrgaIS/independent-target-b/wasm32-wasip1`
- `.flow/tmp/fn2301-integrated-proof.IJl2tf/independent-target-a/wasm32-wasip1`
- `.flow/tmp/fn2301-integrated-proof.IJl2tf/independent-target-b/wasm32-wasip1`

Each command removed 457 files and reported 284.9 MiB, totaling 1139.6 MiB.
All four selected nested directories are absent. Their parent host-side outputs
and all four clones remain. All eight top-level proof binaries still pass the
retained SHA-256 manifest. The retained scripts and source pins can rebuild the
removed outputs. Details remain in `.flow/tmp/fn230-retirement-result.md`.

Free space increased from 61 to 62 GiB at cleanup, above the 30 GiB admission
threshold. Subsequent active builds use space normally. The fn-231 worker holds
the warm Cargo/Go cache lease under `/home/agent/.cache/codex-build/fn2301-unwind/`;
both caches remain intact. Active evidence with historical fn2174 names, recovery
archives and foreign fn-233 plans remain protected. No remote operation occurred.
