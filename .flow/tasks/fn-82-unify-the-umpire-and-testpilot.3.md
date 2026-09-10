---
satisfies: [R2]
---
# fn-82-unify-the-umpire-and-testpilot.3 Model core layout, capability names, and dead-code deletion

## Description
Finish the model core (R2, spec §R2): `Umpire.Target` becomes `Umpire.Model` with the
four-module layout, the capability and source-reference names from the vocabulary table, exactly
two checking entry points, and the dead modules and declarations deleted.

**Size:** M (mechanical sweep across many files)
**Files:** `model/Umpire/Model/{Types,Canonical,Check,Elab,Table}.lean` (from `Target/{Data,Projection,Semantics,Frontend,FiniteTable+FiniteMachine}.lean`), `model/Umpire/Model.lean`, `model/Umpire/Id.lean`, `model/Umpire/Operation/Parameterized.lean` (moved), `model/Umpire/Core.lean`, `model/Shared.lean` (facade that imports the two deleted Shared modules), `model/ModelLint/ImportGraph.lean` + tests, `Makefile`, every importer and doc, gate
**Touches:** [model/**, tools/umpire/internal/retiredvocabulary/check.go, Makefile, .flow/specs/*.md]

### Approach
- Move and rename modules per the spec table; delete `model/Umpire/Target/Language.lean` (zero declarations), `model/Shared/Transition.lean`, `model/Shared/TraceReplay.lean`, the `model/Umpire/OutcomeClassification/` directory (fold its ImportTests), and `TargetBehaviorClosure` (`Core.lean:401-425`).
- Types: `TargetDefinition`+`TargetDeclaration` into `ModelSpec` with `providers`/`connectors` defaulting to empty (`AuthoredTarget.make` at `Target/Semantics.lean:77-87` copies field by field today); `AuthoredTarget` to `DraftModel`; `CheckedTarget` to `CheckedModel`; `TargetComposition` to `Providers`; `TargetBehaviorDomain*` to `Vocabulary`/`MaybeVocabulary`; `TargetBehaviorDescription` to `BehaviorTable`; `FiniteTargetDefinition` to `TableModelSpec`; `Validated*` to `Checked*`; capability layer per table; `AuthoringOccurrence*`/`AuthoringDiagnostic` to `SourceRef`/`SourceSpan`/`LocatedError`; field `canonicalBehavior` to `behaviorVersion` on all five records (JSON keys at `Target/Projection.lean:129-175` follow).
- Entry points: keep `checkModel` (Except) and `model` (proof); `composeTarget` becomes the private `composeModel`; collapse `FiniteTable.checkTarget`, `checkModelTarget`, `ParameterDomain.checkTarget`, `validateModel` into one `checkModel` per owner namespace.
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
- Deviation recorded during implementation: `FiniteTable` keeps two checkers, not one. `checkModel` lowers a table through its `FiniteModelIdentity` to the ModelValue carriers; `checkTypedModel` keeps the author's typed carriers. They return different types, so they cannot merge; `validateModel` became the total `CheckedTable.withIdentity` instead of a third checker.
## Acceptance
- [ ] `Umpire.Model` with `Types`, `Canonical`, `Check`, `Elab`, `Table` replaces `Umpire.Target`; `Target/Language.lean`, `Shared/Transition.lean`, `Shared/TraceReplay.lean`, and the `OutcomeClassification/` directory are gone
- [ ] `checkModel` and `model` are the only public construction entry points; `composeModel` is private and no `composeTarget` remains
- [ ] Capability, source-reference, and vocabulary names match the spec table; `behaviorVersion` replaces `canonicalBehavior`
- [ ] `make lint-model` passes with the updated `semanticRoots` and Model prefix rule; the Makefile layout check passes
- [ ] Goldens regenerated; the gate rejects `CheckedTarget`, `AuthoredTarget`, `QueryTarget`, `TargetBehaviorDomain`, `MeaningProvision`, `LawWitness`, `Umpire.Target` and passes on the tree
## Done summary
`Umpire.Target` is `Umpire.Model`. `Target/{Data,Projection,Semantics,Frontend,FiniteTable +
FiniteMachine}` became `Model/{Types,Canonical,Check,Elab,Table}`; `Target/Language.lean`,
`Shared/{Transition,TraceReplay}.lean`, `OutcomeClassification/ImportTests.lean` and
`TargetBehaviorClosure` are deleted; `DefinitionFamily` moved to `Umpire/Id.lean` and
`Parameterized.lean` under `Umpire/Operation`. `TargetDefinition` and `TargetDeclaration` merged
into one `ModelSpec` whose `providers`/`connectors` default to empty, which deleted the two
field-by-field copies the old split forced. `AuthoredTarget`/`CheckedTarget`/`TargetComposition`
are `DraftModel`/`CheckedModel`/`Providers`; the behavior domain is a `Vocabulary` and its
description a `BehaviorTable`; the capability layer is `Capability`, `Provider`, `Law`, `LawProof`,
`Meaning`, `Connector`; `AuthoringOccurrence*`/`AuthoringDiagnostic` are `SourceRef`, `SourceSpan`,
`LocatedError`; `canonicalBehavior` is `behaviorVersion`; the `CheckedModel` field is `machine`.
`checkModel` and `model` are the public construction entry points and `composeModel` is private.
`ModelLint` pins the `Umpire.Model` prefix with `model-isolation` rules, the Makefile package check
asserts `Model/Check.lean`, and `Shared.lean` now exposes the modules `Shared` actually owns.

The review caught a real defect the gates could not: task .2's `DefinitionKind` rename and this
task's `KnownGapKind` rename never reached `tools/umpire/internal/artifactv2`, whose validators
still accepted `observation`, `kernel` and `capability-contract`. Any produced artifact carrying a
capability gap decoded as invalid. Both vocabularies are now named slices, `knownGapKindRank`
derives its order from one of them, and a new `vocabulary_test.go` reads
`Umpire.KnownGapKind.name` and `Umpire.DefinitionKind.name` out of the Lean source and pins the Go
lists against them — verified to fail when the Go list drifts.

Deviation, recorded in the task file: `FiniteTable` keeps two checkers. `checkModel` lowers through
a `FiniteModelIdentity` to the ModelValue carriers; `checkTypedModel` keeps the author's typed
carriers. Their return types differ, so they cannot merge. `validateModel` was not a checker at all
and became the total `CheckedTable.withIdentity`.

Note: commit `ebb94a44e` carries this task's 207-file change set. A parallel session ran
`git add -A && git commit -m wip` while the work was in flight and swept it up; only the message
was amended.

stage: impl-review - ran (model: claude-fable-5-1) - NEEDS_WORK then SHIP; 1 P1 (the Go decoder
vocabularies) and 7 P3 findings, all addressed
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: ebb94a44e4c2299cbf401e36937b47701a57047a, 3d2478fe0c6a84be5bac5514ebee1db4f6dc1179, aeee2c7c75c3fe3644a9bd18027ead37ca376bec
- Tests: cd model && lake build Umpire UmpireTests Temporal TemporalModelTests TestpilotTests Testpilot TemporalExperimentalTests (pass), make lint-model (169 diagnostics, equals baseline; modelLint and modelLintTests pass), make umpire-check-goldens / -regression-views / -case-runtime-conformance / -semantic-inventory / -retired-vocabulary (pass), make umpire-check-lean-api / -testpilot-protocol / -testpilot-authoring (pass), make umpire-check-live-tests (pass, 6 passing identities), TMPDIR=<physical> CGO_ENABLED=0 go test -count=1 -tags test_dep ./tools/umpire/... ./common/testing/testpilot/... ./tests/testcore/testpilot/... (pass), go test -run TestKindVocabulariesMatchLean: proven to fail when the Go kind list drifts from Lean, make lint-code GOLANGCI_LINT_FIX=false (128 findings, equals baseline)
- PRs:
stage: plan-sync - skipped(config: planSync.enabled != true)
