---
satisfies: [R7]
---
# fn-29-bounded-production-canary-execution-and.7 Add ArtifactSet v5 and the canary publication closure

## Description
Complete R7 by adding the exact seven-member production-canary qualification closure over Task `.6`'s v4 receipt.

**Size:** M
**Files:** `model/Umpire/Artifact/Set.lean`, `model/Umpire/Artifact/Tests/Set.lean`, `tools/umpire/artifact/set.go`, `tools/umpire/artifact/set_test.go`, `tools/umpire/artifact/publication_test.go`
**Touches:** [model/Umpire/Artifact/Set.lean, model/Umpire/Artifact/Tests/Set.lean, tools/umpire/artifact/set.go, tools/umpire/artifact/set_test.go, tools/umpire/artifact/publication_test.go]

### Approach
- Add ArtifactSet v5 with the six ordinary v1 source members, one v4 receipt, and one qualification-result relation; reconstruct and cross-check every semantic/configuration/run/profile/provenance/Result binding and fixed false release eligibility.
- Preserve v1-v4 set bytes, limits, readers, relations, and destinations; every reader rejects descendant/sibling versions and no migration or repair path is added.
- Reuse the admitted-set publisher and pathless Result reference; extend only generic closure checks needed by v5, without Temporal, canary target, or credential vocabulary.
- Prove source-member byte preservation, strict member/order/identity/relation closure, symlink/alias/root races, concurrent identical/conflicting writers, rollback/interruption recovery, and secret scans.

### Investigation targets
**Required** (read before coding):
- `.flow/specs/fn-18-versioned-umpire-artifact-boundary.md` — admitted set/publication invariants
- `.flow/tasks/fn-28-authorized-remote-staging-black-box.7.md` — v4 evolution and source-byte preservation
- `.flow/tasks/fn-29-bounded-production-canary-execution-and.6.md` — exact receipt/provenance contract
- `tools/common/artifactio/set.go` — lock, path identity, and interruption handling
- `tools/common/artifactio/set_test.go` — publication mutation/recovery pattern

### Acceptance
- [ ] V5 round-trips across Lean/Go and admits only the exact canary seven-member closure.
- [ ] Six source members and all prior set fixtures/readers remain unchanged.
- [ ] Missing/extra/duplicate/crossed/stale/version/relation/release-eligibility/secret mutations reject.
- [ ] Atomic/idempotent/conflict-safe publication and root-revalidation matrices pass.

## Acceptance
- [ ] R7 ArtifactSet v5 and immutable production-canary publication closure are complete.
- [ ] Cross-language set/version/relation/publication suites pass.
- [ ] Existing publication comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
