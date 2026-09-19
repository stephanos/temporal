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
Implemented immutable admitted Artifact-set publication and loading behind a deep artifactio module. Publication validates the complete canonical set, stages an exact private tree, fsyncs files and directories, and exposes it with one sibling rename at `root/sets/<manifest-sha256>`; loading opens that exact digest directory without following symlinks and revalidates the manifest and every member before returning an admitted value. There is no mutable current pointer, multi-root replacement, compatibility normalization, or partial-success path.

Coverage includes byte-identical idempotence, conflict and permission rejection, symlink/non-regular/escape rejection, stale-stage recovery, interruption before install, concurrent publishers, and readers observing only absence or one complete set. The configured implementation review found a manifest A→B→A race; manifest reads are now pinned to the initially verified bytes and a synchronized regression test proves the fix.

stage: impl-review - ran [2026-08-29T09:59:28Z..2026-08-29T10:02:48Z] (model: gpt-5.6-sol)
## Evidence
- Commits: e70aaea77c77629ef44f0bf6debf7d6cbe8c9b81, df217e3c6949fca9907c2895c5c0345e84c6aa4b
- Tests: baseline: mise exec -- go test -count=1 ./tools/umpire/artifact/... ./tools/common/artifactio/... (pass), TDD RED: focused tests failed because artifact.PublishSet, artifact.LoadSet, and artifactio.ImmutableDirectory were absent, mise exec -- go test -count=1 ./tools/umpire/artifact/... ./tools/common/artifactio/... (pass at reviewed HEAD), mise exec -- go test -race -count=1 ./tools/common/artifactio/... ./tools/umpire/artifact/... (pass), manifest A-to-B-to-A synchronized regression: behaviorally red before fix, green after pinning the first verified manifest, mise exec -- go test -race -count=1 ./tools/common/artifactio/... -run 'TestImmutableDirectory(RejectsManifestABA|Interruption|RejectsConcurrentWriter|Recovers)' (pass), mise exec -- go vet ./tools/common/artifactio/... ./tools/umpire/artifact/... (pass), scoped golangci-lint (0 issues), gofmt and git diff --check (pass), make lint-model (pass), make umpire-check-legacy-vocabulary (pass at reviewed HEAD), make umpire-check-regression (pass at reviewed HEAD: generated views, Go regression tests, active vocabulary, 226-job Lean build), flowctl codex impl-review fn-18-versioned-umpire-artifact-boundary.10 --base c8844972d --receipt /tmp/impl-review-receipt-fn-18-versioned-umpire-artifact-boundary.10.json (SHIP)
- PRs:

stage: plan-sync - skipped(config: planSync.enabled != true)
