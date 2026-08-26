---
satisfies: [R7, R8]
---
# fn-18-versioned-umpire-artifact-boundary.10 Atomically publish and recover immutable artifact sets

## Description
Implement R7's safe immutable filesystem publication and interruption recovery over already admitted current-version sets.

**Size:** M
**Files:** `tools/umpire/artifact/publish.go`, `tools/umpire/artifact/publish_test.go`, `tools/common/artifactio/**`
**Touches:** [tools/umpire/artifact/publish.go, tools/umpire/artifact/publish_test.go, tools/common/artifactio/**]

### Approach
- Reuse and deepen only necessary `artifactio` safe-path, lock, private staging, fsync, install, rollback, and recovery primitives while preserving its comments.
- Publish an admitted current-version set to an immutable directory keyed by SHA-256 of exact canonical manifest bytes; never overwrite a set or create mutable `current` state.
- Reject absolute/traversal/colliding paths, symlinks, non-regular files, unsafe modes, stale bindings, and target conflicts before visibility.
- Fully validate staged bytes before install, atomically install, then reopen and strictly revalidate the installed set before success.
- Prove identical concurrent publication idempotent and inject failure/interruption at every phase while preserving all prior complete sets.

### Investigation targets
**Required** (read before coding):
- `tools/common/artifactio/artifact.go:10-40`
- `tools/common/artifactio/set.go:16-110,475-645`
- `tools/common/artifactio/set_test.go:13-253`
- `tools/umpire/internal/generate/regression/generate.go:116-176`
- Tasks `.8` and `.9` admitted/migrated set contracts

### Acceptance
- [ ] Unsafe filesystem inputs and conflicting existing bytes reject without changing prior sets.
- [ ] Failure injection at every stage leaves no visible partial set and recovery converges.
- [ ] Identical concurrent publish is idempotent and successful installs are strictly reopened/revalidated.
- [ ] Publication remains retention-neutral and accepts only admitted current-version sets.

## Acceptance
- [ ] R7 immutable atomic publication and recovery are implemented.
- [ ] Existing artifactio behavior and comments are preserved.
- [ ] Artifact and artifactio focused tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
