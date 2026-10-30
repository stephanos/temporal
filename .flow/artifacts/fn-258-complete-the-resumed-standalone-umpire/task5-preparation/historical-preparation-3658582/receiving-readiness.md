# Local receiving-ref readiness

Revalidation passes: 88 entries, 81 originals and 15 refusal controls. All 717 frozen R1 framework/historical-framework paths appear in the 48 receiving plans, with 5,044 candidate-path occurrences, zero missing/extra paths and zero per-candidate destination collisions. All 48 names are valid, absent and free of prefix/case collisions. Every planned source ref still names its pinned source commit. Exact full source OIDs and mapping pointers for each receiver are in [receiving-readiness.json](receiving-readiness.json); full mappings remain in [receiving-ref-creation-work.json](receiving-ref-creation-work.json).

The map is complete for changed-path accounting, **not** a complete snapshot or hunk recipe. It contains 1,931 committed deletion observations. Deletions need pinned before-images and verified after-absence; copying only present changed files loses that meaning. Historical paths also need explicit layout handling. No receiving target commits exist yet.

## Next local actions, after root admission

1. Preserve source custody first: bundle the exact overlay recovery refs, independently recover all original/observed objects, and archive raw dirty/untracked bytes, modes, symlinks, index stages and staged/unstaged binary patches. Verify an out-of-tree restore. Inventory unique ignored inputs; protect shared caches. Do not stash, stage, reset or clean original checkouts.
2. Prepare 46 committed custody snapshots in isolated object stores. Hold `candidate-039` and `candidate-075` until dirty custody is verified. `039` has nine dirty framework paths and an unresolved three-stage index for `model/umpire/backends/process/src/support.rs`; retain stages and working bytes without resolving it. `075` has **only** dirty framework contributions: five modified and 40 untracked files. A ref to its committed HEAD would lose all 45 planned framework paths. Both dirty checkouts currently report zero ignored directory/file boundaries; that is an observation, not an archive.
3. Construct complete framework-only recovery snapshots at the exact pinned source OIDs, applying only explicitly admitted dirty custody. Verify full projected tree/blobs/modes, deletion before-images, path accounting and archive links. Keep source/server/tracker/shared/Scala history under its original recovery. Do not import monorepo commits or interpret a recovery snapshot as feature acceptance. Use explicitly identified new custody commits; do not invent original author/parent history. If authored candidate history must move too, separately filter and verify it.
4. Emit reviewed target OIDs and a create-only transaction containing exactly the 48 existing names. Root may then run `git -C /Users/stephan/Workspace/skunkworks/umpire/umpire update-ref --stdin < reviewed-receiver-transaction.txt`, whose body is `start`, 48 `create <exact-ref> <verified-custody-oid>` lines, `prepare`, `commit`. **This is a later mutation template; verified target OIDs are not available now.** Never update/force existing refs or renumber stable ledger-index identities.
5. Verify and bundle/recover all receiver targets, prove canonical `05473210` HEAD/tree/index/worktree unchanged, and update preparation evidence to recovery-only verified. Keep transfer acceptance, outstanding backups and overall R6 open.

## Six source-only freezes

| Source ref under `refs/heads/` | Exact preserved commit |
| --- | --- |
| `agent/attempt-fencing-prototype` | `bac2d44038362121939b0904e1993e2e8e25f787` |
| `checklocks-matcher-data` | `a325f98d350dff75ff16c82a9b1ab29dd9bda83b` |
| `w-1d-core` | `243f9dd525b16b7ab15b7f1fa5d486e0f7ce822a` |
| `w-1d-shared` | `86e4ee2505d9c3f379d820fb1d5bb1a658629287` |
| `w-perf` and `w-perf-dev` | `f4fba65eb634f11f547679279597bf48a69fa797` |

All six refs/objects are verified and have no active registered checkouts. They need no standalone receiving refs or deferred feature implementation. Their committed recovery is established; historical unique ignored/untracked absence is not established merely because an old checkout is gone. Retain original repository/artifacts, locate any prior custody manifests, and keep that uncertainty frozen until explicitly reviewed. No full-preservation claim is made.

Authoritative action/tracker locations remain `.flow/tasks/fn-258-complete-the-resumed-standalone-umpire.5.{json,md}` and each overlay entry's existing Temporal Flow links. No tracker history or system-named metadata files belong in standalone active source. Recovery trees are not automatically admitted for checkout/activation; normal source-boundary review still applies.

Consumer `3658582d` currently has seven tracked missing-sibling/docs/wiring edits pending root's fix. Preserve them before retirement and explicitly admit its new consumer pin/affected proof; do not silently replace the prepared pair. Destination SSO/access, publication, fetchable CI activation, source-removal authorization and final gates remain independent blockers. No source/ref/Flow mutation or external write was performed by this readiness pass.
