# Primary-index cache investigation

The two research reports below are preserved verbatim. They record observations before the root agent continued the pending cherry-pick. The separate continuation record identifies later root-reported evidence and the limited independent verification performed during artifact capture.

## First report, verbatim

Evidence favors a virtiofs pathname-cache inconsistency. This remains an inference until checked on the native host.

- Workspace mount is `virtiofs host`, device `0:58`.
- Directory enumeration repeatedly shows only `.git/index`.
- Explicit lookup and reading of `.git/index.lock` succeed, reporting the index’s inode `417856692`, link count **1**, size `7728713`, and identical timestamps/content. Both SHA-256 hashes are `467970417543d0faa4cdb5b7c22d0012d758a5970c2a691768270c8108739ede`.
- No guest Git process or open descriptor appeared in `ps`/`lsof`. Guest inspection cannot exclude host-side writers.
- HEAD and CHERRY_PICK_HEAD remain exactly as supplied. No repository mutation performed.

Git 2.53 creates an exclusive `.lock` file and commits it by `rename()` to `index`; its normal protocol does not create a hardlink pair. [Git lockfile implementation](https://raw.githubusercontent.com/git/git/v2.53.0/lockfile.h)

Docker explicitly documents Git index corruption/unexpected content with virtiofs caching and recommends recreating the sandbox with `DOCKER_SANDBOXES_ENABLE_VIRTIOFS_CACHE=0`. The setting applies at sandbox creation; exporting it inside this running guest will not repair the mount. [Docker troubleshooting](https://docs.docker.com/ai/sandboxes/troubleshooting/#filesystem-operations-are-slow-in-large-repositories)

Recommended continuation:

1. Inspect the same two paths and active Git processes from native macOS. Compare directory listing against explicit `stat`/`ls`, and revalidate HEAD, CHERRY_PICK_HEAD, staged paths, and owner-file preservation.
2. If native macOS sees a coherent index, no lock, and no writer, the documented operation to resume is `git cherry-pick --continue`, subject to the parent’s existing integration authority. [Git documentation](https://git-scm.com/docs/git-cherry-pick#_sequencer_subcommands)
3. If native inspection also disagrees, keep the primary checkout paused. Sandbox recreation requires preserving in-VM work/state first.

There is no supported basis here for deleting, renaming, or truncating either index path.

## Second report, verbatim

`GIT_INDEX_FILE` provides a supported workaround for this single pending commit. It can avoid `.git/index.lock` while leaving the primary index and working files untouched. It does **not** repair the virtiofs inconsistency or establish that host-side writers are absent.

Git 2.53’s source supports the full path:

- `sequencer_continue()` refreshes the configured index. With no `.git/sequencer/todo`, it invokes `continue_single_pick()`, which launches ordinary `git commit` with the inherited environment. [Sequencer source](https://github.com/git/git/blob/v2.53.0/sequencer.c)
- Plain commit uses the selected index as-is. Its index lock targets `repo->index_file`; it does not stage working-tree content. [Commit source](https://github.com/git/git/blob/v2.53.0/builtin/commit.c), [repository source](https://github.com/git/git/blob/v2.53.0/repository.c)
- `GIT_INDEX_FILE` is documented for core Git commands, and commit hooks receive that selected index explicitly. [Environment documentation](https://git-scm.com/docs/git/2.53.0#Documentation/git.txt-GITINDEXFILE), [hook invocation source](https://github.com/git/git/blob/v2.53.0/commit.c)

Current read-only checks found 75 staged paths, zero unmerged entries, no split-index backing file, no `MERGE_AUTOSTASH`, no configured hooks path, and no active commit, index-change, or reference-transaction hooks. `MERGE_RR` is empty. HEAD and CHERRY_PICK_HEAD remain as supplied.

The least reconstructive preparation is a **byte copy of the current primary index into a unique directory outside virtiofs**. Do not initialize from worker commit `8d8131…`. Before continuation, compare primary/copy bytes, complete `ls-files --stage -z` output, and the exact staged diff. Current fingerprints are:

| Read-only representation | SHA-256 |
|---|---|
| Complete stage entries | `0e53b7da8816e0b38e842619c4464558a063112d2a5f6a8f7780337fb09897df` |
| Cached raw diff, full object IDs | `9e7373147118b432be76a7e230f0f865f2a971f3c86ff516913941d437781172` |

An authorized continuation would use the copy’s absolute `GIT_INDEX_FILE` with `cherry-pick --continue`; disabling automatic maintenance for that invocation would bound ancillary work. The primary index should then already match the new commit’s tree, so no restore or index replacement should be necessary.

Remaining conditions are exclusivity against other writers, preservation checks for the 26 owner files, and post-commit verification of the new parent/tree and cleanup state. Ref/object writes and cherry-pick cleanup still occur on virtiofs and could fail independently. The alternate index also does not acquire the primary index’s normal lock, so it must not run alongside another writer.

No mutation or alternate-index creation was performed.

## Root-reported continuation outcome

The root agent subsequently reported successful alternate-index continuation, with no deletion or rename of the primary index. The root reported the following evidence.

| Observation | Reported value |
|---|---|
| Primary HEAD | `8b13b026302e06b0d356065443b579f35b2006ca` |
| Parent | `acbfaa2f308aaefb338626eadcb31beda50e3b25` |
| Tree | `43232590c5f3e453ca473c9a4bc20eea6710ef50` |
| Cherry-pick state | Cleared |
| Cached diff | Empty |
| Primary/copy stage entries and cached raw diff before continuation | Exact comparisons returned `0` |
| Owner-file preservation | All 26 files unchanged, verified by SHA |
| Original index after continuation | SHA-256 `467970417543d0faa4cdb5b7c22d0012d758a5970c2a691768270c8108739ede`, unchanged |
| Alternate index after continuation | Same SHA-256 as original index |
| Continuation control | `/tmp/gomad-primary-index-continuation.BlVj82wF/continue.sh` |
| Control SHA-256 | `56628e436750144ce4551c7bd28f6fd80927b99a47efcf19dff6013f94e7f4b3` |
| Command/tool session | `74912`, exit `0` |
| Last root observation of primary `index.lock` | `stat` returned `ENOENT` |

The root supplied the owner-file evidence digest only as the abbreviated `eaa772...`; this artifact does not reconstruct or claim its full value. The absent lock observation does not establish that the cache inconsistency was cured.

During artifact capture, this research agent independently ran read-only `git --no-optional-locks rev-parse HEAD HEAD^ HEAD^{tree}` in the primary repository and `sha256sum` on the continuation control. Those checks matched the HEAD, parent, tree, and control SHA-256 recorded above. The other continuation checks remain attributed to the root agent. This artifact capture made no index, repository-state, commit, or product-code changes.
