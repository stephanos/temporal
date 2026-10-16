# WASM worktree consolidation

The owner requested preservation and merging of work outside the main checkout
after stopping implementation. This operation consolidates saved source and
history; it does not resume the WASM specifications or claim qualification.

## Active source integration

| Saved work | Integration into `gomad` |
| --- | --- |
| `fn180-evidence` through `012d434b06` | Merge `69d7c9b25e`; preserves the new Runner tests and the current inspection, operator and host-status fields |
| `gomad-fn181-operator-20261015` through `dd5c3789f8` | Merge `43eb5d0729`; preserves operator fixtures, queued regressions and documentation alongside cleanup documentation |
| `gomad-fn182-final-20261015` through `98833acfb5` | Merge `dfa0d11f79`; includes slim guest source, cache/replay tests, dependency analysis and interrupted measurement evidence |
| Runtime, library, routing and identity-test checkpoints | `git cherry` confirmed their patches were already present; their original commits are now reachable through the preservation merge |
| Cleanup through `bc2412de28` | Already merged as `cc0fd53dd4` before the owner stop |

Conflict resolution retained the current combined source inventories,
`ClusterResult` host flags and inspection metadata, `Result.DebugOutput`, and
both the cleanup and installed-scenario qualification sections. The additional
replay test file was the only active-source addition from the evidence merge.

## Historical source preservation

Preservation merge `ccb453cadddd9756beb377f9fb1abb3964390183` adds 19 checkpoint
parents without changing the active source tree. It retains original
cherry-picked histories and older worktree variants, including detached HEADs,
in main's ancestry. Their source can be recovered even if those worktree
directories are later removed. No worktree or branch was removed here.

The worktree inventory also found two older dirty fn-176 checkouts. Alternate
Git indexes captured their source, including the untracked
`wasi/storage_selection.go`, without changing their original HEADs, indexes or
working files. Each changed file's Git blob was compared with the working file.
Both exact snapshots are parents of the preservation merge:

| Original worktree | Permanent ref | Snapshot |
| --- | --- | --- |
| `gomad-fn176-catalogue-admission-20261015` | `preserved/2026-10-16/fn176-catalogue` | `70f8c2909298e1aaf9821d4ba6448b6e15d9c9dd` |
| `gomad-fn176-rust-admission-20261015` | `preserved/2026-10-16/fn176-rust` | `58ea199fcbfcd2fbafab03d1467b7727e3a7e340` |

Historical variants are preserved as history, rather than replacing the newer
integrated fn-176/fn-178 implementations. For example, `git show
preserved/2026-10-16/fn176-catalogue:tools/gomad/internal/backends/wasm/wasi/storage_selection.go`
recovers the previously untracked source.

All 14 temporary worker handovers were copied into
[wasm-preserved-handovers-2026-10-16](wasm-preserved-handovers-2026-10-16/).
`cmp` verified every copy against its `/tmp` original. Existing tracked batch
records and partial measurement collectors are also in main. Raw build outputs,+binaries and large test artifacts retain their original local locations; this
operation did not delete or relocate them.

## Verification and remaining state

- All original worktree HEADs and the two dirty snapshots are reachable from
  `gomad`; the preservation merge's tree equals its first parent's tree.
- Merge whitespace checks passed. The focused Go check compiled the merged
  Runner and installation packages. The installation selector-refusal regression
  passed.
- `TestReplayInspectionUsesDedicatedProviderAfterPreflight` and
  `TestReplayInspectionRejectsMissingWitnessAndPreservesPreflightRefusals` fail
  while constructing their fixture with `artifact replay mode does not match
  its backend capability`. Both fail identically on the original
  `fn180-evidence` checkpoint, so these are inherited unfinished tests rather
  than a claim of passing merged acceptance. Both test commands exited 1.
- No full WASM test, runtime qualification, installed scenario matrix, performance
  comparison, lint or independent review was claimed. All fn-180 through fn-183
  tasks remain incomplete as recorded at the stop checkpoint.
- Concurrent fn-184/fn-185 planning files and their MILESTONES edits were left
  untouched. No push, PR or CI operation was performed.
