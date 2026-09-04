---
satisfies: [R1, R7, R8]
---
# fn-62-make-ordinary-temporal-model-authoring.6 Carry model-owned Known Gaps end to end

## Description
Implement R1, R7, and R8 by placing explicit checked model-owned Known Gaps at the narrowest established authoring boundary and carrying them deterministically through planning and artifacts.

**Size:** M
**Files:** `model/Umpire/Query/Language.lean`, `model/Umpire/Artifact/Planning.lean`, `model/Umpire/Planning/Tests/KnownGaps.lean`, `model/Umpire/SemanticInventory/Tests/KnownGaps.lean`, `model/Temporal/Feature/Nexus/OperationsTests.lean`
**Touches:** [model/Umpire/Query/Language.lean, model/Umpire/Artifact/Planning.lean, model/Umpire/Planning/Tests/KnownGaps.lean, model/Umpire/SemanticInventory/Tests/KnownGaps.lean, model/Temporal/Feature/Nexus/OperationsTests.lean, model/Temporal/Feature/Nexus/Fixtures/**]

### Approach
- First trace the existing `KnownGap`, `KnownGapSet`, canonical planner gaps, semantic inventory, and artifact path; choose the narrowest Target/Query/Observation-owned seam that represents model limitations without changing behavior.
- Reuse `KnownGapSet.ofUnordered`, `checkCanonical`, `union`, and `toList` in `model/Umpire/Planning/Types.lean:8-148`; do not add another gap vocabulary or mutable registry.
- Carry authored gaps alongside checked model data, union them deterministically with phase-owned gaps, and preserve source identity/cardinality through planning and serialization.
- Add a small Nexus-authored gap only as a checked propagation fixture; do not invent unsupported behavior solely to populate it.
- Regenerate only source-bearing fixtures whose intentional Known Gap content changes; broad generated API drift/CI remains out of scope.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Planning/Types.lean:8-148` — opaque checked Known Gap set and canonical operations.
- `model/Umpire/Artifact/Types.lean:131-170` — current phase-owned planner gaps.
- `model/Umpire/Artifact/Planning.lean:300-350` — planning artifact construction and gap injection.
- `model/Umpire/Planning/Tests/KnownGaps.lean` — deterministic gap regressions.
- `model/Umpire/SemanticInventory/KnownGaps.lean:346-505` — source/catalog lineage and validation.

**Optional** (reference as needed):
- `.flow/memory/bug/integration/portable-model-plans-need-exact-2026-09-03.md:16-25` — exact artifact and explicit-obligation constraint.
- `.flow/memory/bug/integration/portable-schemas-must-preserve-source-2026-09-03.md:15-24` — source shape and cardinality constraint.

### Acceptance
- [ ] Model authors can declare explicit capability/input/interpretation/claim gaps as checked existing Known Gap data without affecting allowed traces or Property truth.
- [ ] Checked Query/planning/artifact/result paths retain every authored gap exactly once and deterministically union phase-owned gaps.
- [ ] Malformed/duplicate codes, invalid categories, crossed bindings, noncanonical external order, and missing required gaps reject before runtime I/O.
- [ ] A gap never establishes success, changes behavior, or silently disappears in partial/failure output.
- [ ] Semantic inventory, canonical ordering, artifact goldens, and Nexus propagation regressions cover the intended delta.

## Acceptance
- [ ] R1, R7, and R8 are satisfied with one checked Known Gap vocabulary.
- [ ] `cd model && mise exec -- lake build Umpire.Planning.Tests Umpire.SemanticInventory.Tests Umpire.Artifact.Tests Temporal.Feature.Nexus.OperationsTests` passes.
- [ ] Reviewed fixture deltas are limited to intentional source/Known Gap content and all exact artifact checks pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
