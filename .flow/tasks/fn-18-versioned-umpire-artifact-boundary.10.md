---
satisfies: [R7, R8]
---
# fn-18-versioned-umpire-artifact-boundary.10 Atomically publish and load immutable Artifact sets

## Description
Publish and load one immutable admitted Artifact set without exposing partial or mixed state.


**Size:** M
**Files:** `tools/umpire/artifact/publish.go`, focused tests, and `tools/common/artifactio/**`
**Touches:** [tools/umpire/artifact/publish.go, tools/umpire/artifact/publish_test.go, tools/common/artifactio/**]

### Approach
- Reuse the existing safe-path, lock, private staging, validation, fsync, install, and recovery
  primitives while preserving comments, but do not use multi-root sequential replacement as the
  visibility boundary.
- Validate the complete candidate before installation and re-open/revalidate before success.
- Reject symlinked or escaping paths, non-regular files, conflicting bytes, permission failures, and interruption without damaging a prior set.
- Install one previously absent immutable `root/sets/<manifestSha256-hex>` directory by sibling
  rename after file/directory fsync; expose no mutable current pointer. `LoadSet` accepts the exact
  digest directory, opens without following symlinks, rehashes/revalidates the manifest and every
  member, and returns only the complete admitted value.

### Investigation targets
**Required:** `tools/common/artifactio/**` and task `.8`'s admitted-set API.

## Acceptance
- [ ] Concurrent readers observe one complete old or new set, never a mixture.
- [ ] Failed or interrupted publication preserves the prior complete set and leaves no false success.
- [ ] Identical publication is idempotent; conflicting content rejects.
- [ ] Existing artifactio behavior and comments are preserved.
- [ ] Reader/writer concurrency and injected interruption tests prove readers observe absence or one
  complete digest directory, while identical publish revalidates idempotently and conflict rejects.

### Quick command

`mise exec -- go test -count=1 ./tools/umpire/artifact/... ./tools/common/artifactio/...`

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
