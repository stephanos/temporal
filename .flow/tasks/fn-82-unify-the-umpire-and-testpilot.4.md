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
`Umpire.Behavior` is `Umpire.Scenario` and the Property language is one authored record.

`Property/{Language,Fields,Authoring,Trace,Evaluation}.lean` collapsed into `Umpire/Property.lean`
(types, fields, sugar) plus `Property/{Check,Evaluate,Elab}.lean`; `Behavior/{Language,Authoring}.lean`
became `Umpire/Scenario.lean` plus `Scenario/{Check,Elab}.lean` and the tests moved with them.
`PropertyDeclaration` + `PropertySpec` collapsed into one `Property`; `BehaviorDeclaration` +
`BehaviorSpec` + `ExactSequenceSpec` collapsed into one `Scenario` whose sugar is now the functions
`Scenario.exactly` and `Scenario.constrained`. `check` and `checked` are the construction operations
on both; `PropertyAuthoring` is deleted and `opaqueDeclaration` survives as a plain error kind with
its own test. The occurrence types are namespaced under `Scenario` (`Step`, `Count`, `Order`, `Role`,
`Slot`) because `Umpire.Step` has been the model step since `.2`.

Clause vocabulary: `PropertyCase`/`PropertyCaseGroup`/`PropertyException` are `PropertyBranch`/
`PropertyBranches`/`PropertyUnless`, the `Resolved*` carriers folded into `Checked*`, `sameStepCases`
is `branches`, `quiescentWithin` is `neverWithin`, and `eventuallyWithin`/`neverWithin` absorbed the
two `guarded*` constructors by taking an optional guard, an optional Unless and a clause source.
`PropertyPredicateContext` is `before`/`after`. `PropertyLimit`, `PropertyLimitProfile` and
`PropertyScopedClock` are deleted: clauses carry a plain `Limit`, and `correlated_response%` (was
`bounded_response%`) lost its `on <clock>` phrase. The correlated endpoints are `.final` and
`.«partial»` (guillemets because `partial` is a Lean keyword; the repo already spells `«property»`
that way). `modelOutcome` is `outcome` everywhere in the Property surface and its canonical JSON.

The gate now rejects `BehaviorDeclaration`, `CheckedBehavior`, `BehaviorSpec`, `ExactSequenceSpec`,
`Umpire.Behavior`, `behavior%`, `PropertyDeclaration`, `PropertySpec`, `PropertyAuthoring`,
`PropertyCase(Group)`, `PropertyException`, `sameStepCases`, `quiescentWithin`, `modelOutcome`,
`bounded_response%`, the five deleted `Umpire.Property.*` module paths, and the occurrence,
`Resolved*`, `checkProperty`/`checkBehavior` and `guardedQuiescentWithin` compounds.

Method: goldens, Nexus fixtures and the Case conformance trees regenerated through `umpire-gen-goldens`
and `umpire-gen-case-runtime-conformance`; every inline fingerprint, checksum and expected-JSON pin
was re-baselined by running the owning module and reading the computed value, never by inventing one.
The review's P1 (an `unless` without a guard silently dropped at admission) was fixed with two clause
checks that were shown red before they landed.

Deviations, all recorded here:
- `Property.check`/`Scenario.check` take `(context) (declaration)` rather than the spec sketch's
  `Property -> CheckContext`; dot notation gives authors exactly `property.check context`, and the
  order keeps ~40 free-function call sites unchanged.
- `PropertyTraceField`/`PropertyPredicateField.resultingState` were NOT respelled: `resultingState`
  is still a live Nexus3 `require` keyword (task .8) and the proto `json=resultingState` in scanned
  generated Go (task .7), so it cannot enter the gate here and a half-rename would be worse. The
  `.state` collision the task flagged is therefore still open for whichever task retires the keyword.
- `Umpire.Property` is the types module, so it is no longer a facade. The Makefile package-layout
  check was rewritten to assert that each package's `Check.lean` builds on its types module; the
  Query arm keeps the old `Language.lean` shape until task .5.
- The checked clause carriers keep `guardedEventuallyWithin`/`guardedNeverWithin`: on that side
  `guarded` names a real distinction (trigger-frozen applicability), not authoring sugar, so the
  spec's "constructors lose the guarded prefix" was applied to the authored `PropertyClause` only.
- The Scenario canonical JSON keys and the `"behavior"` Definition-ID segment were kept, so Scenario
  fingerprints are byte-stable. The task text expected the keys to "follow"; none of them contained
  a retired name.
- `modelOutcomes` and `propertyLimit` stay: both are artifact wire keys with a Go decoder, which
  task .6 owns. `PropertyLimit` is therefore not in the gate yet.
- `.plans/UMPIRE4_{DSL,SPEC_COMPS}.md` were respelled (one line each) against the "no historical
  .plans edits" boundary, because the gate's `UMPIRE4_*.md` glob scans them - the same deviation `.2`
  recorded for `UMPIRE4_SPEC_COMPS.md` and `UMPIRE4_SPEC_MODEL_ARCH.md`.

Swept in, not authored here: the fn-67/fn-77/fn-80/fn-81 spec status flips and the fn-80.5 task
record a parallel session left uncommitted in this checkout.

stage: impl-review - ran (model: claude-fable-5-1) - NEEDS_WORK then SHIP; 1 P1, 2 P2 and 5 P3
findings, all addressed
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 747d83daabf449be43027df46849318e13380d40, 55e48bd5eb1ee16ae749c6411894d0f321f16648, a7b7af236c4d0177916495216ff1028834b56c4a, a02b1d2ab9acde8845e4eb6274b19c7700d380b6, 23d674849f8586772f6fb858e335220105e89f6a
- Tests: cd model && lake build Umpire UmpireTests Temporal TemporalModelTests TestpilotTests Testpilot TemporalExperimentalTests +Umpire.PromotionTests (pass, 405 jobs), make lint-model (169 diagnostics, all in generated Temporal/API/{Types,Proto}.lean; equals baseline; modelLintTests, modelLint and the import-graph check pass), make umpire-check-goldens (pass), make umpire-check-regression-views (pass), make umpire-check-case-runtime-conformance (pass), make umpire-check-semantic-inventory (pass), make umpire-check-retired-vocabulary (pass, with 42 new retired compounds), make umpire-check-lean-api / -testpilot-protocol / -testpilot-authoring (pass), make umpire-check-live-tests (pass, 6 passing identities), make buf-breaking (pass), TMPDIR=<physical> CGO_ENABLED=0 go test -count=1 -tags test_dep ./tools/umpire/... ./common/testing/testpilot/... ./tests/testcore/testpilot/... (pass), make lint-code GOLANGCI_LINT_FIX=false (128 findings: errcheck 1, govet 4, revive 106, staticcheck 17; equals baseline), CC=/usr/bin/cc go vet -tags test_dep ./... (15 diagnostics, equals baseline), lake build Umpire.Property.Tests.GuardedTemporal with the two new clause checks disabled: proven red before the fix
- PRs:
stage: plan-sync - skipped(config: planSync.enabled != true)
