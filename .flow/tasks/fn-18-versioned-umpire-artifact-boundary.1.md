---
satisfies: [R2, R8]
---
# fn-18-versioned-umpire-artifact-boundary.1 Complete the executable ExperimentSpec and canonical artifact package

## Description
### Umpire4 reconciliation (normative)

The current writer must define the complete executable `ExperimentSpec` version required by Umpire4: setup, participant programs, typed symbolic references, actions, faults, ordering/concurrency/causality, observations, expectation program, termination/convergence, phase bounds, omissions, and cleanup. Preserve legacy `umpire-experiment/v1` bytes as read-only compatibility fixtures; do not freeze that incomplete shape as the current writer.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Prepare R2/R8's Lean-owned schema boundary by moving the existing artifact implementation behind vertical modules and adding shared canonical/binding vocabulary without aliases.

**Size:** M
**Files:** `model/Umpire/Artifact.lean`, `model/Umpire/Artifact/Canonical.lean`, `model/Umpire/Artifact/Binding.lean`, `model/Umpire/Artifact/Experiment.lean`, `model/Umpire/Artifact/Tests/Codecs.lean`, `model/UmpireTests.lean`
**Touches:** [model/Umpire/Artifact.lean, model/Umpire/Artifact/**, model/UmpireTests.lean]

### Approach
- Move current DrivePlan/ExperimentSpec declarations, identity functions, lowering, and comments intact into `Artifact/Experiment.lean`; keep `Umpire.Artifact` as the public import facade, not a compatibility alias layer.
- Define exact persisted-byte, SHA-256 binding, provenance-digest, family, limit, and structured schema-error vocabulary reused by later artifact modules.
- Preserve existing semantic JSON and public declaration behavior byte-for-byte; add the canonical persisted projection by appending exactly one LF without changing semantic identity.
- Add Lean fixtures for non-empty fn-16 intent arrays and exact canonical field/list ordering.
- Preserve all existing comments while moving code.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Artifact.lean:36-91,93-241,311-382` — existing structures, comments, encoders, and lowering
- `model/Umpire/Core.lean:28-109` — shared identities/provenance
- `model/Umpire/Space/Compilation.lean` — fn-16 populated artifact contract after dependency lands
- `model/Umpire/ARCHITECTURE.md:207-235` — current public artifact API

### Acceptance
- [ ] Existing Switch and Nexus canonical document/fixture bytes and semantic identities remain unchanged.
- [ ] Persisted bytes are exactly the canonical document plus one LF.
- [ ] The facade exposes vertical modules with no duplicate structure or compatibility alias.
- [ ] Non-empty choice/variant/fault fixtures preserve request-only semantics and existing comments.
## Acceptance
- [ ] R2's existing artifact contract remains byte-identical behind the new package layout.
- [ ] Shared binding/canonical vocabulary is domain-neutral and inert.
- [ ] UmpireTests and focused Lean codec tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
