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
- Define documentation descriptors, the initial shared Known Gap source shapes and exact/lossy carry mappings, and catalog validation primitives in the shared module; task fn-47.4 owns any additional lineage shape proved necessary by the complete current flow inventory.
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
Added typed semantic-inventory descriptors, closed Known Gap source/carry vocabulary, and owner-local exact-one classifiers for Planning plus all five runtime status families. Focused tests pin current name order and payload-independent Planning classification without changing constructors, rendered behavior, or comments.

Baseline: expected greenfield red (future semantic-inventory Lake executable/test and Make targets were absent); `make lint-model` was green at 200/200.

stage: impl-review - ran [SHIP at 2026-09-01T11:50:37Z; session 01a05ccb-f00a-7cd0-a2bf-3243759aa4d9; 0 open findings]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 09f35d0612a94bc3ebaaf59a1374b42bab213479
- Tests: baseline: expected greenfield red (future semantic-inventory Lake executable/test and Make targets absent); make lint-model green 200/200, TDD RED: cd model && mise exec -- lake build Umpire.SemanticInventory.Tests.PlanningRuntime (missing task-owned classifier/catalog API), cd model && mise exec -- lake build Umpire.SemanticInventory.Tests.PlanningRuntime, make lint-model, Lean #print axioms for all six exact-one classifier proofs (no axioms)
- PRs:
