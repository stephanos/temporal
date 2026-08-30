---
satisfies: [R1]
---
# fn-47-generate-umpire-semantic-outcome-and.1 Catalog Planning and runtime outcome families

## Description
Define the reusable descriptor vocabulary and owner-local exhaustive Planning/runtime catalogs for R1.

**Size:** M
**Files:** `model/Umpire/SemanticInventory/Types.lean`, `model/Umpire/Planning/Engine.lean`, `model/Umpire/Artifact/Runtime.lean`, `model/Umpire/SemanticInventory/Tests/PlanningRuntime.lean`
**Touches:** [model/Umpire/SemanticInventory/Types.lean, model/Umpire/Planning/Engine.lean, model/Umpire/Artifact/Runtime.lean, model/Umpire/SemanticInventory/Tests/PlanningRuntime.lean]

### Approach
- Define only documentation descriptors, Known Gap lineage/scope/source-shape vocabulary, and catalog validation primitives in the shared module.
- Add typed canonical lists and total membership proofs beside PlanningOutcome and each runtime status owner.
- Preserve current constructors, rendered names, behavior, derived instances, and comments.
- Test exact ordered names and compile-time exhaustiveness without source scanning or reflection.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Planning/Engine.lean:196-211` — PlanningOutcome and rendered names.
- `model/Umpire/Artifact/Runtime.lean:107-208` — five distinct runtime status families.
- `model/Umpire/Planning/Tests/KnownGaps.lean:33-54` — compact compile-time assertion style.
- `model/Umpire/Observation/Verdict.lean:11-17,62-80` — separate-family precedent.

### Quick commands
`cd model && mise exec -- lake build Umpire.SemanticInventory.Tests.PlanningRuntime`

## Acceptance
- [ ] Planning and all five runtime families expose canonical typed values plus total membership proofs.
- [ ] Exact current rendered names and order are pinned.
- [ ] No shared status enum, constructor change, behavior change, or source scanner is introduced.
- [ ] A missing/duplicate/unclassified constructor fails compile-time proof or focused tests.
- [ ] Existing comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
