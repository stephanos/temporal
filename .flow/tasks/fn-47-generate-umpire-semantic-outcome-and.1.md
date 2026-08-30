---
satisfies: [R1]
---
# fn-47-generate-umpire-semantic-outcome-and.1 Catalog Planning and runtime outcome families

## Description
Define the reusable descriptor vocabulary and owner-local exact-one Planning/runtime constructor classifiers for R1.

**Size:** M
**Files:** `model/Umpire/SemanticInventory/Types.lean`, `model/Umpire/Planning/Engine.lean`, `model/Umpire/Artifact/Runtime.lean`, `model/Umpire/SemanticInventory/Tests/PlanningRuntime.lean`
**Touches:** [model/Umpire/SemanticInventory/Types.lean, model/Umpire/Planning/Engine.lean, model/Umpire/Artifact/Runtime.lean, model/Umpire/SemanticInventory/Tests/PlanningRuntime.lean]

### Approach
- Define documentation descriptors, the five explicit Known Gap source shapes and exact/lossy carry mappings, and catalog validation primitives in the shared module.
- Add canonical constructor descriptors and exact-one classifiers beside PlanningOutcome and each runtime status owner. Payload-free enums may use values; PlanningOutcome.found and .invalid match constructors while ignoring arbitrary payload identity.
- Preserve current constructors, rendered names, behavior, derived instances, and comments.
- Test exact ordered names, representative payload independence, and compile-time constructor exhaustiveness without source scanning or reflection.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Planning/Engine.lean:196-211` — payload-bearing PlanningOutcome and rendered names.
- `model/Umpire/Artifact/Runtime.lean:107-208` — five distinct runtime status families.
- `model/Umpire/ImplementationLink/Language.lean:19-24` and `model/Umpire/Observation/Evaluation.lean:74-77` — distinct non-KnownGap source/projection shapes.
- `model/Umpire/Planning/Tests/KnownGaps.lean:33-54` — compact compile-time assertion style.

### Quick commands
`cd model && mise exec -- lake build Umpire.SemanticInventory.Tests.PlanningRuntime`
## Acceptance
- [ ] Planning and all five runtime families expose canonical constructor descriptors plus exact-one classifier proofs.
- [ ] PlanningOutcome.found and .invalid classify every arbitrary payload without representative-value enumeration.
- [ ] Exact current rendered names and order are pinned.
- [ ] No shared status enum, constructor change, behavior change, or source scanner is introduced.
- [ ] A missing, overlapping, or unclassified constructor fails the proof or focused tests.
- [ ] Existing comments are preserved.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
