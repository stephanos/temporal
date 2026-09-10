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
- Provenance: `Umpire.Provenance.DefinitionKind.behavior` becomes `scenario` and its encoded string `CASE_DEFINITION_KIND_SCENARIO` (`Umpire/Case/Provenance.lean:27` today); the Case fixtures regenerate in this task.
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
- `Property/Scoped/*` and `Umpire/Case/Scoped.lean` are NOT renamed in this task; task .7 owns the `Scoped` to `Correlated` sweep so proto, Lean, and Go change together. The `.partial`/`.final` constructors land here and `Umpire/Case/Scoped.lean` maps them to the still-named proto `SCOPED_ENDPOINT_*` values until task .7; the gate entries for `runtimePrefix` and `deliberatelyClosed` are task .7's.
- Retire `BehaviorDeclaration`, `CheckedBehavior`, `BehaviorSpec`, `ExactSequenceSpec`, `Umpire.Behavior`, `behavior%`, `PropertyDeclaration`, `PropertySpec`, `bounded_response%`, `sameStepCases`, `quiescentWithin`; never the bare word `Behavior`.
- Carried from .2: the retired gate still has no `modelOutcome` / `resultingState` rule. `PropertyTraceField` and `PropertyPredicateField` keep both spellings (and `PropertyTraceField` already has a `.state` constructor, so `.resultingState` needs a name this task chooses), as does `PropertyPredicateInput`. Add `modelOutcome` to the gate here once those constructors are respelled. The Step JSON keys are already `outcome`/`state`/`facts` in every serializer .2 touched.
## Acceptance
- [ ] `Property` and `Scenario` are the only authored records for their languages, each with `check` and `checked`, and `CheckedProperty`/`CheckedScenario` are their checked forms
- [ ] `Umpire/Behavior*` no longer exists; `Umpire/Property/{Language,Fields,Authoring,Trace}.lean` are merged away; `property%`, `scenario%`, `correlated_response%` elaborate
- [ ] Clause, branch, and endpoint names match the spec; `Provenance.DefinitionKind.scenario` replaces `behavior` and the Case fixtures are regenerated; a Property with an ambiguous branch group still reports the same finding under `Overlap*` naming once task .5 lands (unchanged here)
- [ ] `make lint-model`, the Makefile layout check, `lake build Umpire UmpireTests Temporal TemporalModelTests`, and regenerated goldens pass
- [ ] The gate rejects the retired compounds and passes on the tree
## Done summary
Blocked:
Not started. No code was changed for this task: the file moves I began were reverted and the tree
is green at the `.3` receipt. This is a session-budget stop after `.1`, `.2` and `.3`, not a
technical blocker — the task is startable exactly as written.

Sequencing notes for whoever picks it up, from the survey done before stopping:

- The Scenario record merge is the expensive part. `BehaviorDeclaration`, `BehaviorSpec` and
  `ExactSequenceSpec` must collapse into one `Scenario` record whose only construction operations
  are `check` and `checked`. The two sugar records are consumed as record literals at 74 references
  across 11 files (`Nexus2/Authoring.lean`, `Nexus3/Authoring.lean`, `Nexus3/Syntax.lean`,
  `Nexus3/Tests.lean`, `Nexus2/AuthoringTests.lean`, the three `Nexus/Operations` walkthroughs,
  `Operations/PlanningTests.lean`, `Umpire/Behavior/ImportTests.lean`,
  `Umpire/Behavior/Tests/Authoring.lean`), so each `def x : ExactSequenceSpec := { ... }` becomes a
  named-argument call to `Scenario.exactly` (or its constraint counterpart for `BehaviorSpec`).
- `model/Umpire/Behavior/Language.lean` splits cleanly at its first `private def quote` line:
  everything above is the `Umpire/Scenario.lean` types module (through `CheckedBehavior`, which has
  no private constructor), everything below is `Umpire/Scenario/Check.lean`.
- `NamedOccurrence`/`OccurrenceBound`/`OccurrenceOrder`/`ResourceRole` cannot take the bare names
  `Step`/`Count`/`Order`/`Role`: `Umpire.Step` is the model step since `.2`. Namespace them under
  `Scenario`.
- `BehaviorTrace` should become `Scenario.Trace`, not the bare `Trace` the R2 table reserves:
  neither `.2` nor `.3` renamed `ModelTrace`/`ModelCoordinate`, so `Trace`/`TraceAddress` are still
  owed by an R2 sweep and would collide.
- Carried from `.2`: `modelOutcome` can enter the retired gate in this task once
  `PropertyTraceField` and `PropertyPredicateField` are respelled. `PropertyTraceField` already
  owns a `.state` constructor, so `.resultingState` needs a name this task chooses.
## Evidence
- Commits:
- Tests:
- PRs:

### R2 debt carried out of .2 and .3 (found by the spec completion review)

Both tasks shipped without three renames the R2 table names. They are byte-neutral (type names and
type parameters, not canonical JSON keys), so they need no golden regeneration:

- `ModelTrace` -> `Trace` and `ModelCoordinate` -> `TraceAddress` (`model/Umpire/Core.lean:154,163`).
  `ModelTraceStep` also retires into Step per the table, but `Umpire.Step` is already the
  transition result, so it wants a namespaced name such as `Trace.Step`.
- The `Fact` type parameter is still spelled `Observation` on `Machine`, `MaybeVocabulary`,
  `Vocabulary`, `ModelTrace` and `FiniteMachine` (`Core.lean:163,304,346,369,391`,
  `Model/Table.lean:194`). Renaming it collides textually with the `Umpire.Observation` module
  namespace that R5 (task .6) owns, so sequence it with that task or rename binders only.
