---
satisfies: [R6]
---
# fn-253-gomad-fast-validation-gates.5 Content-addressed evidence store and retention verb

## Description
R6: deduplicate qualification evidence by content hash and add a retention rule.

Audit, 2026-10-25, on the `/` overlay (196 GB total, 32 GB free):
- **`/home/agent/wt/evidence`: ≈79 GiB allocated, 624k files.**
  - ≈13 GiB of copied executables: 106 `gomad`, 123 Rust test/bin, plus `gomad-wasmhost`, fixture and libtest copies.
  - ≈8.5 GiB of hash-named blobs, 64k files of per-run artifact stores and preparation caches.
  - 9 GiB of `.wasm`, ≥3.6 GiB of it duplicates.
  - 3.7 GiB of `*.stdout`, 4.8 GiB of JSON, ≈3 GiB of `.rlib` and `target/` copies.
  - The 1 GiB "large-store" fixtures are sparse files and use almost no space.
- **`/home/agent/wt/*` worktrees: ≈34 GiB in 93 worktrees,** 43 untouched for more than 2 days.
- **`~/.cache/codex-build`: 36 GiB.** 19 GiB is umpire (`umpire-milestones-20261024`); ≈8 GiB is separate cargo targets for each fn-242 lane.

Do:
1. **Content-addressed store.** Qualification and lane runs put executables, modules and preparation caches into one content-addressed store under the evidence root, keyed by sha256. Runs reference or hardlink from it instead of copying. Receipts already carry digests, so verifying a run means checking those digests against the store.
2. **Retention.** A `gomad` verb (no new Python) prunes evidence. Per lane, it keeps full payloads for the newest green and the newest failed run. Older or superseded runs keep `receipt.json`, digests, `outcome.json`, `schedule.json` and the logs of failing stages. Anything a `.flow/artifacts` file or an open task's handoff cites is pinned. The verb runs in dry-run mode by default and prints what it would delete.
3. **Worktrees and caches.** Prune worktrees whose branch is merged into `gomad` and that no open Flow task names, together with their per-lane cargo targets. Shared caches (`cargo-target`, `go-build`, `pkg/mod`) are only cleaned by package (house rule).

Never delete without the owner's go-ahead on the first dry run. Handoffs live inside evidence directories (B08, B14), and codex's lanes are paused, not finished. Umpire's cache is outside this spec; flag it to the owner.

The spec's implementation clarification distinguishes retained immutable blobs from leased private relocation/mutation scratch. Keep the original private-tree isolation and file authentication; do not hardlink executables to bypass `nlink == 1`. Reader resolution or deterministic private materialization must preserve retained proofs after lease cleanup. An archive-only cache is insufficient, and unknown or unconverted legacy receipts remain pinned until their references and verification are understood.

## Acceptance
- After the store lands, one cycle's evidence contains each distinct executable or module once (checked by a digest census), and a second cycle reuses the first one's blobs.
- The retention verb's dry run on the 2026-10-25 tree lists what it would delete, with pinned references excluded, and the owner approves it before the first real run. After pruning, every retained receipt still verifies against the store.
- Evidence and worktree sizes before and after are recorded in `.flow/artifacts/fn-253/disk.md`.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
