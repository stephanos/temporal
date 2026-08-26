---
satisfies: [R1, R2, R5, R6]
---
# fn-31-deepen-umpire-target-and-simplify.7 Derive Query and Planning views from checked targets

## Description
Add the downstream derivation seam that keeps Target below Query and Planning while removing proof-record assembly from ordinary examples.

**Size:** M
**Files:** `model/Umpire/Query/Language.lean`, `model/Umpire/Query/Tests/Completeness.lean`, `model/Umpire/Query/Tests/Validation.lean`, `model/Umpire/Planning/Engine.lean`, `model/Umpire/Planning/Tests/Fixtures.lean`, `model/Umpire/Planning/Tests/Enumeration.lean`, `model/Umpire/Query.lean`, `model/Umpire/Planning.lean`
**Touches:** [model/Umpire/Query/Language.lean, model/Umpire/Query/Tests/**, model/Umpire/Planning/Engine.lean, model/Umpire/Planning/Tests/**, model/Umpire/Query.lean, model/Umpire/Planning.lean]

### Approach

- Preserve the dependency direction `Target -> Property/Behavior -> Query -> Artifact/Planning`; Target imports neither downstream module.
- Keep `QueryBounds`, `FiniteCompletenessEvidence`, and the `invalidBound`, `unitMismatch`, `missingFiniteCompleteness`, and `targetKernelMismatch` failures owned by Query.
- Extend the checked Target kernel with an explicit finite action list and focused `actionSound`/`actionComplete` proofs. Derive Query role assignments from `CheckedTarget.resolvedSetups` and the Query action domain from that checked kernel evidence; return a complete `CheckedQueryTarget` or the existing deterministic Query error. Family maintainers supply the proof obligations once, while ordinary query authors never assemble `FiniteCompletenessEvidence`.
- Keep `IncrementalPlannerKernel` and `FiniteKernelOrder` owned by Planning, but derive them through one public constructor from admitted Query completeness and target-owned finite action/initial/step lists.
- Preserve Query identities, completeness claims, planner ordering, bounds, outcomes, and existing comments byte-for-byte.

### Investigation targets

**Required** (read before coding):
- `model/Umpire/Query/Language.lean:111-217,276-447`
- `model/Umpire/Query/Tests/Completeness.lean`
- `model/Umpire/Planning/Engine.lean:7-105`
- `model/Umpire/Planning/Tests/Fixtures.lean:126-205`
## Acceptance
- [ ] The checked Target kernel provides explicit finite actions plus soundness/completeness proofs; Query derives its role/action completeness view without manufacturing or weakening evidence and remains the sole owner of bounds and query-level completeness errors.
- [ ] Planning remains the sole owner of the indexed kernel and ordering proofs, and its public derivation produces byte-/result-identical traversal from the same checked inputs.
- [ ] Switch and Temporal examples can consume the checked adapters without constructing `FiniteCompletenessEvidence`, `FiniteKernelOrder`, or `IncrementalPlannerKernel` records directly.
- [ ] Target imports no Query, Planning, Artifact, Temporal, runtime, or verification module; facade and import tests enforce the direction.
- [ ] Focused Query/Planning suites plus `UmpireTests` pass with existing comments preserved.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
