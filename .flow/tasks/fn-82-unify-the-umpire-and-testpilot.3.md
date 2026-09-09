---
satisfies: [R2]
---
# fn-82-unify-the-umpire-and-testpilot.3 Model core layout, capability names, and dead-code deletion

## Description
Finish the model core (R2, spec §R2): `Umpire.Target` becomes `Umpire.Model` with the
four-module layout, the capability and source-reference names from the vocabulary table, exactly
two checking entry points, and the dead modules and declarations deleted.

**Size:** M (mechanical sweep across many files)
**Files:** `model/Umpire/Model/{Types,Canonical,Check,Elab,Table}.lean` (from `Target/{Data,Projection,Semantics,Frontend,FiniteTable+FiniteMachine}.lean`), `model/Umpire/Model.lean`, `model/Umpire/Id.lean`, `model/Umpire/Operation/Parameterized.lean` (moved), `model/Umpire/Core.lean`, `model/ModelLint/ImportGraph.lean` + tests, `Makefile`, every importer and doc, gate
**Touches:** [model/**, tools/umpire/internal/retiredvocabulary/check.go, Makefile, .flow/specs/*.md]

### Approach
- Move and rename modules per the spec table; delete `model/Umpire/Target/Language.lean` (zero declarations), `model/Shared/Transition.lean`, `model/Shared/TraceReplay.lean`, the `model/Umpire/OutcomeClassification/` directory (fold its ImportTests), and `TargetBehaviorClosure` (`Core.lean:401-425`).
- Types: `TargetDefinition`+`TargetDeclaration` into `ModelSpec` with `providers`/`connectors` defaulting to empty (`AuthoredTarget.make` at `Target/Semantics.lean:77-87` copies field by field today); `AuthoredTarget` to `DraftModel`; `CheckedTarget` to `CheckedModel`; `TargetComposition` to `Providers`; `TargetBehaviorDomain*` to `Vocabulary`/`MaybeVocabulary`; `TargetBehaviorDescription` to `BehaviorTable`; `FiniteTargetDefinition` to `TableModelSpec`; `Validated*` to `Checked*`; capability layer per table; `AuthoringOccurrence*`/`AuthoringDiagnostic` to `SourceRef`/`SourceSpan`/`LocatedError`; field `canonicalBehavior` to `behaviorVersion` on all five records (JSON keys at `Target/Projection.lean:129-175` follow).
- Entry points: keep `checkModel` (Except) and `model` (proof); make `composeTarget` private; collapse `FiniteTable.checkTarget`, `checkModelTarget`, `ParameterDomain.checkTarget`, `validateModel` into one `checkModel` per owner namespace.
- Lint: update `semanticRoots` (`ImportGraph.lean:155-173`), `isTargetForbiddenDestination` (`:252`, now the `Umpire.Model` prefix), and `ImportGraphTests.lean`; rewrite the Makefile layout loop at `Makefile:1225-1239` for `Model` (it stays a per-package assertion, now on `Model/Check.lean`).
- Regenerate goldens as in task .2; add compound old names to the gate; respell scanned docs and open specs.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Target/Semantics.lean:15-100,419-543` — the checked/authored records and entry points
- `model/Umpire/Target/Data.lean:11-152` — records and occurrence types
- `model/Umpire/Target/FiniteMachine.lean:43-70,424-513` — table path entry points to collapse
- `model/ModelLint/ImportGraph.lean:145-173,245-263` — pinned module names
- `model/Umpire/ARCHITECTURE.md:40-75` — the Target ownership table that must be respelled

**Optional** (reference as needed):
- `model/Umpire/Target/Authoring.lean` — `DefinitionFamily`, moving to `Umpire/Id.lean`
- `model/Umpire/Target/Parameterized.lean` — declares `namespace Umpire.Operation`; moves, no rename

### Key context
- `Umpire.Target.Semantics` may not reach `Umpire.Target.Frontend` or `Lean.Elab.Term`; the same rule must hold for `Model.Check` versus `Model.Elab` after the move.
- `Makefile:1421` asserts one exact lint diagnostic that names `Umpire.Core`; it does not change here.
- Retire compounds only (`CheckedTarget`, `AuthoredTarget`, `Umpire.Target`, `TargetBehaviorDomain`, `MeaningProvision`, ...), never `Target`.

## Acceptance
- [ ] `Umpire.Model` with `Types`, `Canonical`, `Check`, `Elab`, `Table` replaces `Umpire.Target`; `Target/Language.lean`, `Shared/Transition.lean`, `Shared/TraceReplay.lean`, and the `OutcomeClassification/` directory are gone
- [ ] `checkModel` and `model` are the only public construction entry points; `composeTarget` is private
- [ ] Capability, source-reference, and vocabulary names match the spec table; `behaviorVersion` replaces `canonicalBehavior`
- [ ] `make lint-model` passes with the updated `semanticRoots` and Model prefix rule; the Makefile layout check passes
- [ ] Goldens regenerated; the gate rejects `CheckedTarget`, `AuthoredTarget`, `QueryTarget`, `TargetBehaviorDomain`, `MeaningProvision`, `LawWitness`, `Umpire.Target` and passes on the tree


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
