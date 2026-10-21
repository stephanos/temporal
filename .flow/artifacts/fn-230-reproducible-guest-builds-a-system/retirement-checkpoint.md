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
tag or bypass the guard, and did not attempt the remaining targets. All four
targets and all four clones remain. No reclaimed-space claim is made.

Free space was 61 GiB, above the 30 GiB admission threshold. The root retains the
warm Cargo/Go cache lease under `/home/agent/.cache/codex-build/fn2301-unwind/` for
fn-231. Active evidence with historical fn2174 names, recovery archives and foreign
fn-233 plans remain protected. No remote operation occurred.
