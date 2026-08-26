---
satisfies: [R7]
---
# fn-28-authorized-remote-staging-black-box.7 Add ArtifactSet v4 and the remote publication closure

## Description
Complete R7 by adding the exact seven-member remote qualification closure over Task `.6`'s receipt.

**Size:** M
**Files:** `model/Umpire/Artifact/Set.lean`, `model/Umpire/Artifact/Tests/Set.lean`, `tools/umpire/artifact/set.go`, `tools/umpire/artifact/set_test.go`, `tools/umpire/artifact/publication_test.go`
**Touches:** [model/Umpire/Artifact/Set.lean, model/Umpire/Artifact/Tests/Set.lean, tools/umpire/artifact/set.go, tools/umpire/artifact/set_test.go, tools/umpire/artifact/publication_test.go]

### Approach
- Add ArtifactSet v4 with exactly the six ordinary v1 source members, one v3 receipt, and one qualification-result relation; reconstruct and cross-check every semantic/configuration/run/profile/provenance/Result binding.
- Preserve v1/v2/v3 set bytes, limits, readers, relationships, and destinations; each reader rejects descendant/sibling versions and no migration or repair path is added.
- Reuse the existing admitted-set publisher and pathless Result reference; extend only generic closure checks needed by v4, without Temporal, staging, target, or credential vocabulary.
- Prove source-member byte preservation, strict closure/order/identity, symlink/alias/root races, concurrent identical/conflicting writers, rollback, interruption recovery, and secret scans.

### Investigation targets
**Required** (read before coding):
- `.flow/specs/fn-18-versioned-umpire-artifact-boundary.md` — admitted set and immutable publication invariants
- `.flow/tasks/fn-26-local-qualification-receipts-and-staged.3.md` — v2 qualification set seam
- `.flow/tasks/fn-27-hermetic-ci-execution-and-qualification.8.md` — v3 set evolution and source-byte preservation
- `tools/common/artifactio/set.go:475-645` — lock, path identity, and interruption handling
- `tools/common/artifactio/set_test.go:13-215` — publication mutation/recovery test pattern

### Acceptance
- [ ] V4 round-trips across Lean/Go and admits only the exact remote seven-member closure.
- [ ] All six source members remain byte-identical and all prior set fixtures/readers remain unchanged.
- [ ] Missing/extra/duplicate/crossed/stale member, relation, identity, version, and secret mutations reject.
- [ ] Atomic/idempotent/conflict-safe publication and root revalidation matrices pass.

## Acceptance
- [ ] R7 ArtifactSet v4 and immutable publication closure are complete.
- [ ] Cross-language set/version/relation/publication suites pass.
- [ ] Existing publication comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
