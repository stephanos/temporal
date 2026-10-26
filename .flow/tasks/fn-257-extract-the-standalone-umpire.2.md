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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
