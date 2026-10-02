# Retained bytes of the representative qualification set (fn-114 task 10, R7)

Host darwin/arm64, 2026-10-02, revision `b12b15b1c`, toolchain build key `245141dc…`.

## Result

The unpruned representative set (28 workloads, 2 seeds, 112 artifacts) retains
1.17 GB on disk with shared targets. The same artifacts with a private target
each are 11.44 GB.

| Quantity | Bytes | How it was measured |
| --- | --- | --- |
| After: on disk | 1,177,358,336 | `du -sk` of the artifacts root (1,149,764 KiB); a hard-linked file counts once |
| After: distinct files | 1,173,782,154 | each file once, by inode |
| Before: every artifact owns its target | 11,442,111,182 | every path counts its file (12,293,693,418) minus the 14 pool paths (851,582,236) |
| Sum of per-artifact stored bytes | 11,421,193,338 | `artifact_bytes` of the set report; equals the sum over the 112 manifests |
| Earlier run, before targets were shared | 11,418,746,156 | `artifact_bytes` of the set report this host retained from 2026-10-01 |

The 112 artifacts hold 14 distinct targets (851,582,236 bytes). The `./tests`
binary (154,593,154 bytes) is one file with 49 links: 48 artifacts of 12
workloads and the pool entry.

Per-artifact stored bytes did not change, by the stored-bytes rule
(`artifact.RetainedBytes`): the set report's `artifact_bytes` is 11.42 GB before
and after. That field sums artifacts and is not a disk measurement.

## What the before number is

No run was made at a revision before task 9. The set needs 11.4 GB plus the
2 GiB the set keeps free, and this host had 5 to 10 GiB free during the task.
The before number is therefore the same run counted as if no file were shared.
It is not an estimate: before task 9 every artifact wrote these same bytes as
private files. Two independent figures agree with it: the set report's sum of
stored bytes (0.2% lower, because it leaves out campaign journals and
qualification reports) and the retained report of the 2026-10-01 run on the
earlier toolchain (11,418,746,156).

## Commands

Run from the repository root with the pinned stock go1.27.1 first on `PATH`.
`ARTIFACTS` and `REPORT` were paths in the session scratch directory.

```bash
make -C tools/gomad3 qualification-set \
  GOMAD3_QUALIFICATION_MANIFEST="$PWD/tools/gomad3integration/qualification/temporal.json" \
  GOMAD3_QUALIFICATION_WORKDIR="$PWD" \
  GOMAD3_QUALIFICATION_ARTIFACTS="$ARTIFACTS" \
  GOMAD3_QUALIFICATION_OUTPUT="$REPORT"
.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/task-10/measure-retained-bytes.sh "$ARTIFACTS"
```

This is `make gomad3-qualification` without its compatibility-pack step and with
the artifacts outside `.toolchain`. The run exited 0 in 34 minutes, most of it
building targets for the new toolchain key: `expectations-met=true supported=28
failed=0 infrastructure-errors=0 completed=28/28`, 56 seeds replayed, none
pruned. The report (13,789,050 bytes, SHA-256 `5f44808a…fe07a4a7ae`) and the
artifacts stayed in scratch and were deleted after measuring.

Script output:

```
on_disk_kib (du -sk; a hard-linked file counts once): 1149764
distinct_file_bytes (each file once, by inode): 1173782154
every_path_bytes (each path counts its file; the bytes without sharing): 12293693418
artifact_directories: 112
target_paths_in_artifacts: 112
pool_entries: 14
pool_bytes: 851582236
```

`gomad inspect` on one of the artifacts printed `sharing=shared`, and its target
had 5 links.

## Corpus cap for `./tests` cases

The corpus cap is unchanged: 1,024 cases and 1 GiB. With a private target each,
6 `./tests` cases fit. With the shared target counted once, the count depends on
what a case holds besides the target:

| `./tests` artifact of this run | Stored bytes | Without the target | Cases before | Cases after |
| --- | --- | --- | --- | --- |
| Smallest | 155,482,615 | 889,461 | 6 | 1,024 (the case cap) |
| Median | 158,761,878 | 4,168,724 | 6 | 220 |
| Largest | 174,824,055 | 20,230,901 | 6 | 45 |

All 48 `./tests` artifacts of this run together are 450,399,464 bytes under the
rule; the first 6 by size are 933,727,906 bytes without it.

These are the cap's arithmetic applied to the stored bytes of real `./tests`
success artifacts from the run above. No guided campaign filled a corpus with
`./tests` cases; `TestCorpusByteCapCountsTheSharedTargetOnce` runs the same
arithmetic through the corpus selection.

## Not measured

- linux/amd64 (no native host).
- A pruned set run. That pruning removes a pool entry once no retained artifact
  links to it is covered by
  `TestPruneQualifiedCampaignsRemovesASharedTargetOnlyWithItsLastArtifact` only.
- The complete after number on the qualified candidate is task 14's, which
  runs the set again after the runtime changes.
