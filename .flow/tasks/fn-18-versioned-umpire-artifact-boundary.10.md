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
- Reuse the existing safe-path, lock, private staging, validation, fsync, install, rollback, and recovery primitives while preserving comments.
- Validate the complete candidate before installation and re-open/revalidate before success.
- Reject symlinked or escaping paths, non-regular files, conflicting bytes, permission failures, and interruption without damaging a prior set.

### Investigation targets
**Required:** `tools/common/artifactio/**` and task `.8`'s admitted-set API.

## Acceptance
- [ ] Concurrent readers observe one complete old or new set, never a mixture.
- [ ] Failed or interrupted publication preserves the prior complete set and leaves no false success.
- [ ] Identical publication is idempotent; conflicting content rejects.
- [ ] Existing artifactio behavior and comments are preserved.

### Quick command

`mise exec -- go test -count=1 ./tools/umpire/artifact/... ./tools/common/artifactio/...`

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
