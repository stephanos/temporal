---
satisfies: [R7]
---
# fn-29-bounded-production-canary-execution-and.7 Add ArtifactSet v6 and the canary publication closure

## Description
### Umpire4 reconciliation (normative)

All canary-specific policy, profiles, claims, approvals, production authority, credentials, leasing, fencing, recovery, cleanup, rate/concurrency/blast-radius controls, audit, commands, workflows, and documentation belong to the independently owned `tools/canary` module. Umpire supplies stable generic artifact, runner, participant, Run Evaluation, and Claim Assessment interfaces only; it never imports `tools/canary` and gains no canary-specific types. The Lean model may define and verify the eligible trace subset, while the standalone canary owns operational policy and consumes the same complete `ExperimentSpec`. Replace legacy `tools/umpire` canary paths and Umpire-specific canary schema extensions accordingly.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Complete R7 by adding the exact seven-member production-canary Claim Assessment closure over Task `.6`'s v5 receipt.

**Size:** M
**Files:** `model/Umpire/Artifact/Set.lean`, `model/Umpire/Artifact/Tests/Set.lean`, `tools/umpire/artifact/set.go`, `tools/umpire/artifact/set_test.go`, `tools/umpire/artifact/publication_test.go`
**Touches:** [model/Umpire/Artifact/Set.lean, model/Umpire/Artifact/Tests/Set.lean, tools/umpire/artifact/set.go, tools/umpire/artifact/set_test.go, tools/umpire/artifact/publication_test.go]

### Approach
- Add ArtifactSet v6 with the six ordinary v2 source members, one v5 receipt, and one evaluation-receipt-result relation; reconstruct and cross-check every semantic/configuration/run/profile/provenance/Result binding and fixed false release eligibility.
- Preserve v2-v5 set bytes, limits, readers, relations, and destinations; every reader rejects descendant/sibling versions and no migration or repair path is added.
- Reuse the admitted-set publisher and pathless Result reference; extend only generic closure checks needed by v6, without Temporal, canary target, or credential vocabulary.
- Prove source-member byte preservation, strict member/order/identity/relation closure, symlink/alias/root races, concurrent identical/conflicting writers, rollback/interruption recovery, and secret scans.

### Investigation targets
**Required** (read before coding):
- `.flow/specs/fn-18-versioned-umpire-artifact-boundary.md` — admitted set/publication invariants
- `.flow/tasks/fn-28-authorized-remote-staging-black-box.7.md` — v5 evolution and source-byte preservation
- `.flow/tasks/fn-29-bounded-production-canary-execution-and.6.md` — exact receipt/provenance contract
- `tools/common/artifactio/set.go` — lock, path identity, and interruption handling
- `tools/common/artifactio/set_test.go` — publication mutation/recovery pattern

### Acceptance
- [ ] V6 round-trips across Lean/Go and admits only the exact canary seven-member closure.
- [ ] Six source members and all prior set fixtures/readers remain unchanged.
- [ ] Missing/extra/duplicate/crossed/stale/version/relation/release-eligibility/secret mutations reject.
- [ ] Atomic/idempotent/conflict-safe publication and root-revalidation matrices pass.
## Acceptance
- [ ] R7 ArtifactSet v6 and immutable production-canary publication closure are complete.
- [ ] Cross-language set/version/relation/publication suites pass.
- [ ] Existing publication comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
