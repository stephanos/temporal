---
satisfies: [R5, R8]
---
# fn-27-hermetic-ci-execution-and-qualification.8 Add ArtifactSet v3 and the CI publication closure

## Description
Complete R5/R8 by adding the strict derived CI set and immutable publication path over Task `.4`'s receipt.

**Size:** M
**Files:** `model/Umpire/Artifact/Set.lean`, `model/Umpire/Artifact/Tests/Set.lean`, `tools/umpire/artifact/set.go`, `tools/umpire/artifact/set_test.go`, `tools/umpire/artifact/publication_test.go`
**Touches:** [model/Umpire/Artifact/Set.lean, model/Umpire/Artifact/Tests/Set.lean, tools/umpire/artifact/set.go, tools/umpire/artifact/set_test.go, tools/umpire/artifact/publication_test.go]

### Approach

- Add ArtifactSet v3 with exactly the six CI v1 source members, one v2 receipt, and one qualification-result relation; reconstruct the source set and cross-check every semantic/configuration/run/profile/provenance/Result binding.
- Keep ArtifactSet v1/v2 codecs, readers, token/cardinality ceilings, relationship vocabulary, fixtures, and publication destinations byte-for-byte unchanged; every reader rejects other versions.
- Reuse the existing pathless Result reference and atomic publisher; pass only an admitted v3 set and extend the lock-guarded root revalidation seam needed by the CI controller without adding CI vocabulary to generic validation.
- Prove source-member byte preservation, strict seven-member closure, relationship order, identity formulas, atomic/idempotent/conflict-safe publication, interruption recovery, and no migration/repair path.

### Investigation targets

**Required** (read before coding):
- `.flow/specs/fn-18-versioned-umpire-artifact-boundary.md` — canonical set and publisher invariants
- `.flow/tasks/fn-18-versioned-umpire-artifact-boundary.8.md` — ArtifactSet v1 closure
- `.flow/tasks/fn-18-versioned-umpire-artifact-boundary.10.md` — immutable publication and recovery
- `.flow/specs/fn-26-local-qualification-receipts-and-staged.md` — local v2 closure to preserve
- `.flow/tasks/fn-26-local-qualification-receipts-and-staged.3.md` — prior set-version seam
- Task `.4` — exact admitted CI receipt and identity projection

### Acceptance

- [ ] V3 round-trips byte-for-byte across Lean/Go and accepts only the exact seven-member CI closure.
- [ ] Every missing/extra/duplicate/crossed member, relationship, identity, source-set, receipt/Result, and version mutation rejects.
- [ ] All six source members remain byte-identical; v1/v2 fixtures/readers remain byte-identical and reject v3.
- [ ] Publication and lock-guarded root revalidation are atomic/idempotent/conflict-safe with no migration, repair, or partial visibility.

## Acceptance
- [ ] R5/R8 ArtifactSet v3 and immutable publication closure are complete.
- [ ] Cross-language set/version/relation and publication/recovery matrices pass.
- [ ] Existing artifact comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
