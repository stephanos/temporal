---
satisfies: [R3]
---
# fn-82-unify-the-umpire-and-testpilot.4 Property and Scenario records, clauses, and module layout

## Description
Property and Scenario (R3, spec §R3): one authored record per language with `check` and
`checked`, `Umpire.Behavior` becomes `Umpire.Scenario`, Property clauses and branches take the
spec's names, and the per-language module layout collapses. The `Scoped` directory keeps its
name here; task .7 renames it to `Correlated` across the whole tree in one pass.

**Size:** M (mechanical sweep across many files)
**Files:** `model/Umpire/Property.lean`, `model/Umpire/Property/{Check,Evaluate,Elab}.lean`, `model/Umpire/Scenario.lean`, `model/Umpire/Scenario/{Check,Elab}.lean`, deletions of `Property/{Language,Fields,Authoring,Trace}.lean` and `Behavior/*`, `model/ModelLint/ImportGraph.lean`, `Makefile`, importers, docs, gate
**Touches:** [model/Umpire/Property*, model/Umpire/Scenario*, model/Umpire/Behavior*, model/Umpire/Examples/**, model/Temporal/**, model/ModelLint/**, model/Umpire/ARCHITECTURE.md, Makefile, tools/umpire/internal/retiredvocabulary/check.go, .flow/specs/*.md]

### Approach
- Property: merge `Language`+`Fields`+`Authoring` into `Umpire/Property.lean` (`Property` replaces `PropertyDeclaration`+`PropertySpec`; drop `PropertyAuthoring` but keep the `opaqueDeclaration` error); merge `Trace.lean` into `Check.lean`; rename `Evaluation.lean` to `Evaluate.lean`; move the three `%` elaborators into `Property/Elab.lean` (fn-80 R2 depends on their located-diagnostic path, so they stay). Fold `Resolved*` into `Checked*`. Clause changes per spec: optional guard replaces the `guarded*` constructors, `sameStepCases` to `branches`, `quiescentWithin` to `neverWithin`, `PropertyCase`/`PropertyCaseGroup`/`PropertyException` to `Branch`/`Branches`/`Unless`, `PropertyPredicateContext` to `before`/`after`, delete `PropertyLimit`, `PropertyLimitProfile`, `PropertyScopedClock`. Rename `bounded_response%` to `correlated_response%` and its endpoints to `.partial`/`.final`.
- Scenario: `Umpire/Behavior/{Language,Authoring}.lean` become `Umpire/Scenario.lean` + `Scenario/Check.lean` + `Scenario/Elab.lean`; `BehaviorDeclaration`/`BehaviorSpec`/`ExactSequenceSpec` collapse to `Scenario` with `Scenario.exactly`; `CheckedBehavior` to `CheckedScenario`; `BehaviorTrace` to `Trace`; `NamedOccurrence`/`OccurrenceBound`/`OccurrenceOrder`/`ResourceRole` to `Step`/`Count`/`Order`/`Role`; `behavior%` to `scenario%`. JSON keys in `Behavior/Language.lean:669-783` follow, so query goldens regenerate.
- Lint and Makefile: `semanticRoots` entries for `Umpire.Property.*` and `Umpire.Behavior.Language`; the layout loop now asserts `Property/Check.lean` and `Scenario/Check.lean`.
- Regenerate goldens; add compound old names to the gate; respell scanned docs (`Property/COMPATIBILITY.md`, `Umpire/ARCHITECTURE.md`, Nexus docs) and open specs.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Property/Language.lean:289-395` — `PropertyClause` and the declaration record
- `model/Umpire/Property/Check.lean:160-260,1200-1258` — resolved types and the two entry points
- `model/Umpire/Property/Authoring.lean:93-137,269-289` — `bounded_response%`, `PropertySpec`, `property%`
- `model/Umpire/Behavior/Language.lean:134-215,669-783,803-900` — declaration, JSON keys, entry points
- `model/Umpire/Behavior/Authoring.lean:9-100,292-314` — the two sugar records and `behavior%`

**Optional** (reference as needed):
- `model/Temporal/Feature/Nexus/Operations/SuccessfulCompletion.lean` — production consumer of `PropertySpec`/`ExactSequenceSpec`
- `model/Umpire/Property/Tests/GuardedCases.lean` — branch tests to re-baseline

### Key context
- `Property/Scoped/*` and `Umpire/Case/Scoped.lean` are NOT renamed in this task; task .7 owns the `Scoped` to `Correlated` sweep so proto, Lean, and Go change together.
- Retire `BehaviorDeclaration`, `CheckedBehavior`, `BehaviorSpec`, `ExactSequenceSpec`, `Umpire.Behavior`, `behavior%`, `PropertyDeclaration`, `PropertySpec`, `bounded_response%`, `sameStepCases`, `quiescentWithin`; never the bare word `Behavior`.

## Acceptance
- [ ] `Property` and `Scenario` are the only authored records for their languages, each with `check` and `checked`, and `CheckedProperty`/`CheckedScenario` are their checked forms
- [ ] `Umpire/Behavior*` no longer exists; `Umpire/Property/{Language,Fields,Authoring,Trace}.lean` are merged away; `property%`, `scenario%`, `correlated_response%` elaborate
- [ ] Clause, branch, and endpoint names match the spec; a Property with an ambiguous branch group still reports the same finding under `Overlap*` naming once task .5 lands (unchanged here)
- [ ] `make lint-model`, the Makefile layout check, `lake build Umpire UmpireTests Temporal TemporalModelTests`, and regenerated goldens pass
- [ ] The gate rejects the retired compounds and passes on the tree


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
