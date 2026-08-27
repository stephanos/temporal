---
satisfies: [R1, R2, R5, R6]
---
# fn-31-deepen-umpire-target-and-simplify.7 Derive Query and Planning views from checked targets

## Description
Add the downstream derivation seam that keeps Target below Query and Planning while removing proof-record assembly from ordinary examples.

This task establishes and directly verifies that seam in Query/Planning. The dependent Tasks `.3`
and `.4` own adopting it in Switch and Temporal respectively, including their end-to-end
compatibility fixtures; this task does not edit those example sources ahead of that migration wave.

**Size:** M
**Files:** `model/Umpire/Query/Language.lean`, `model/Umpire/Query/Tests/Completeness.lean`, `model/Umpire/Query/Tests/Validation.lean`, `model/Umpire/Planning/Engine.lean`, `model/Umpire/Planning/Tests/Fixtures.lean`, `model/Umpire/Planning/Tests/Enumeration.lean`, `model/Umpire/Query.lean`, `model/Umpire/Planning.lean`
**Touches:** [model/Umpire/Query/Language.lean, model/Umpire/Query/Tests/**, model/Umpire/Planning/Engine.lean, model/Umpire/Planning/Tests/**, model/Umpire/Query.lean, model/Umpire/Planning.lean]

### Approach

- Preserve the dependency direction `Target -> Property/Behavior -> Query -> Artifact/Planning`; Target imports neither downstream module.
- Keep `QueryBounds`, `FiniteCompletenessEvidence`, and the `invalidBound`, `unitMismatch`, `missingFiniteCompleteness`, and `targetKernelMismatch` failures owned by Query.
- Consume Target's additive finite-planning capability. When it is unavailable, an exhaustive Query returns the existing deterministic `missingFiniteCompleteness`; this does not invalidate or partially check the Target. When available, derive Query role assignments from `CheckedTarget.resolvedSetups` and the Query action domain from the target-owned action list and focused `actionSound`/`actionComplete` proofs. Copy the target's stable role/action-domain compatibility tokens verbatim into the canonical Query completeness view so existing strings such as the Switch and Nexus domain tokens remain byte-identical. Return a complete `CheckedQueryTarget` or the existing deterministic Query error. Family maintainers supply the finite-domain proof obligations and tokens once, while ordinary query authors never assemble `FiniteCompletenessEvidence`.
- Keep `IncrementalPlannerKernel` and `FiniteKernelOrder` owned by Planning, and adapt `IncrementalPlannerKernel.ofFinite` behind one public checked-query derivation rather than introducing another enumerator or proof system.
- Preserve Query identities, exact role/action-domain token strings, completeness claims, planner ordering, bounds, outcomes, canonical JSON, and existing comments byte-for-byte.

### Investigation targets

**Required** (read before coding):
- `model/Umpire/Query/Language.lean:111-217,276-447`
- `model/Umpire/Query/Tests/Completeness.lean`
- `model/Umpire/Planning/Engine.lean:7-105`
- `model/Umpire/Planning/Tests/Fixtures.lean:126-205`
## Acceptance
- [ ] An explicitly planning-unavailable checked Target remains usable by non-exhaustive semantic consumers, while an exhaustive Query fails with `missingFiniteCompleteness`; an opted-in Target provides explicit finite actions plus soundness/completeness proofs.
- [ ] Query derives its role/action completeness view without manufacturing or weakening evidence, copies target-owned stable role/action-domain compatibility tokens verbatim, and remains the sole owner of bounds and query-level completeness errors.
- [ ] Planning remains the sole owner of the indexed kernel and ordering proofs, and its public derivation produces byte-/result-identical traversal from the same checked inputs.
- [ ] The public derivation reuses the established finite-kernel implementation; no parallel planner enumerator or duplicate ordering authority exists.
- [ ] The checked adapters are sufficient for Switch and Temporal examples to consume without constructing `FiniteCompletenessEvidence`, `FiniteKernelOrder`, or `IncrementalPlannerKernel` records directly; Tasks `.3` and `.4` perform those concrete source migrations.
- [ ] Query/Planning tests preserve stable role/action-domain tokens and traversal through the derivation seam; Tasks `.3` and `.4` verify the existing Switch, Nexus Lifecycle, and Experimental CallerClosure tokens and canonical Query JSON byte-for-byte when adopting it.
- [ ] Target imports no Query, Planning, Artifact, Temporal, runtime, or verification module; facade and import tests enforce the direction.
- [ ] Focused Query/Planning suites plus `UmpireTests` pass with existing comments preserved.
## Done summary
Added the Target-to-Query derivation that projects resolved role assignments, target-owned finite actions, proofs, and stable compatibility tokens without introducing a second evidence authority. Added the checked-Query-to-Planning derivation that accepts only canonical ordering obligations and delegates to the existing `IncrementalPlannerKernel.ofFinite` enumerator; Query/Planning fixtures now exercise the public seams without assembling downstream proof records.

Planning-unavailable targets remain valid for non-exhaustive Query consumers and fail exhaustive Query checking with the existing deterministic `missingFiniteCompleteness`. Focused traversal, facade visibility, import direction, aggregate model builds, regression projections, and model lint all pass. Concrete Switch and Temporal adoption remains intentionally owned by dependent Tasks `.3` and `.4`, as clarified in this task after Codex withdrew its sequencing finding.

The implementation stayed within the declared Umpire Query/Planning surface. External commit `7105d4d29f9b0f9f3430869fb6202da3d245d243` absorbed the first six task files together with unrelated `.plans/UMPIRE4_ORDER.md`; worker-authored commits are `51604b51fe7bd44f85e6d24a0777660df16917da` and `4f8ef3a3ea3047aceb3dbc1c55b14f0202bea969`. Unrelated dirty `.plans/UMPIRE4_SPEC.md` and `.flow/memory/declined/generated-api-drift-verification.md` were preserved and excluded from task commits.

baseline: green
stage: impl-review - ran [2026-08-27T03:57:45Z..2026-08-27T04:04:04Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 7105d4d29f9b0f9f3430869fb6202da3d245d243, 51604b51fe7bd44f85e6d24a0777660df16917da, 4f8ef3a3ea3047aceb3dbc1c55b14f0202bea969
- Tests: cd model && mise exec -- lake build Umpire.TargetTests Umpire.Query.Tests Umpire.Planning.Tests, cd model && mise exec -- lake build Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.OperationsTests Temporal.Feature.Nexus.Experimental.CallerClosureTests, cd model && mise exec -- lake build UmpireTests TemporalModelTests, make umpire-check-regression, make lint-model, rg -n ^import (Umpire.(Query|Planning|Artifact)|Temporal) model/Umpire/Target model/Umpire/Target.lean -g *.lean (no matches), git diff --check
- PRs:
