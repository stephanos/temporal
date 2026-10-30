---
satisfies: [R2]
---
# fn-257-extract-the-standalone-umpire.2 Produce and verify a history-preserving local export

## Description
Implement the reviewed revision-aware extraction only in a disposable clone; retain the original working repository unchanged.

**Size:** M
**Files:** this spec's extraction tool/configuration and evidence; disposable export repository.
**Touches:** [.flow/artifacts/fn-257-extract-the-standalone-umpire/**, .flow/tmp/fn257-export/**]

### Approach
- Use git-filter-repo's documented callback interface in an independent clone with no shared Git directory. Preserve original messages, authors and dates.
- Project complete historical trees or equivalently prove move/merge deletion handling; reject mapped-path collisions. Preserve mixed historical support under a separate provenance namespace rather than overwriting current root files.
- Preserve the old-to-new commit map, representative DSL/checker/macro/fixture/host histories, and exact final source comparison.
- Keep export and portable-layout changes as separate commits.

### Investigation targets
**Required:** task 1's source/history map; MIGRATE.md execution step 2; git-filter-repo primary documentation.

### Equivalence harness
Compare every selected source blob, mode and symlink target with the projected export at the mapped checkpoint; compare representative original/rewritten commit metadata and ancestry.

## Acceptance
- [ ] The local export matches the approved selected tree exactly and contains no unrelated active Temporal/Scala source.
- [ ] Historical paths and both archive transitions are covered with a retained commit map and representative metadata proof.
- [ ] The working source repository and all candidate worktrees remain unchanged by filtering.

## Done summary
Produced available-history export `cf8bec4d6231302df54b57fccdc5bdb9d11552fc` from frozen source `c88fdc550694020eb1c6a2e09577813d906c9ffe`, only in independently created clones. Two fresh clones reproduced the exact same export commit and commit map. The independent verifier compared all 7,149 available source snapshots: 581 retained commits preserve raw author/committer/date/message/encoding metadata and parent frontiers/order; 6,568 zero-pruned commits are projection-neutral. All 1,178 selected source files match exactly in bytes, modes and checked-out content. Both sibling archive moves, earlier framework/host/native-tool paths and current snapshot groups remain covered.

Source refs/index/product-state observations across 36 original worktrees and fingerprints of 56 dirty paths match before/after filtering and clone-only cleanup. Root migration metadata/docs are declared exceptions; ignored cache bytes were neither inventoried nor modified. The later provisional consumer worktree was deliberately created after this preservation proof and retains separate ownership.

The source itself is shallow at 2026-02-05, before every recorded Rust/host/tool birth; both clones have the same available ancestry and no selected Rust prefix exists at the cutoff. Earlier ancestry, including older LICENSE history, is unavailable and is not claimed recovered. Rewritten history is complete within the supplied input. Clone-only cleanup removed stale shallow metadata, local origin/reflogs and unreachable monorepo objects; strict integrity checks pass. The committed 3,112,032-byte recovery bundle restores the exact verified checkout and all 581 rewritten commits without Temporal.

Negative controls reject the original repository, its registered worktree, exported-file corruption and a tampered source map. Review found a verifier ancestry gap; a real disposable-graph mutant preserves snapshots/metadata while adding an extra sibling parent. The legacy checker accepted it; the corrected checker rejects it, while the actual export passes all parent comparisons. The three-axis panel and focused follow-up returned SHIP with the finding resolved.

Thirty historical support files remain under `.history-support/` for separate portable-layout adaptation. No Model, protocol, import or semantic-golden adaptation happened here; no Rust/Go build/runtime test ran under the owner hold. Standalone adaptation, consumer generation, actual execution and publication/cutover remain tasks 3–6. Destination access is still unknown: gh HTTP401 and connected-app SAML HTTP403. Original source/branches were not removed or published.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 63f5c334c0, 35efd6422e, bfd362e34c
- Tests: export-history.py executed twice on independent --no-local --revision=c88fdc5506 clones; identical cf8bec4d6231302df54b57fccdc5bdb9d11552fc and commit-map SHA256, verify-export.py: all 7149 available snapshots, 581 retained metadata/parent frontiers, 6568 neutral prunings and 1178 active files pass; verification-with-parent-edges.json, capture-preservation.py: all 36 original worktrees and 56 dirty paths match before/after export and clone-only cleanup, Original/registered-worktree refusal, file-corruption and commit-map negative controls pass; negative-controls.json, check-parent-control.py: legacy verifier accepts extra sibling edge; corrected verifier rejects it with rewritten-parent-order, Clone-only cleanup: strict git fsck passes, complete rewritten parent graph, original unfiltered source commit absent from both private object stores, git bundle verify and a fresh clone recovery reproduce the verified export and all 581 rewritten commits, Three-axis implementation review and focused post-fix review: SHIP
- PRs: