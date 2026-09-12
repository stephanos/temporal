---
satisfies: [R3, R6, R7]
---
# fn-84-deepen-five-shallow-module-clusters-in.3 Umpire.Search.admit and the AdmittedQuery

## Description
Give Umpire one `admit` operation (R3) that owns the Property, Scenario, Query, search-view and search chain and returns either one `AdmissionDiagnostic` or an `AdmittedQuery`; migrate the six Temporal callers and the Switch example onto it; move the two search-view transports (`SearchView.retarget`, `AdmittedQuery.withQuery`) inside `Umpire.Search` and switch the five production sites and thirteen fixtures onto them; give the four `.checked` constructors the `native_decide` auto-param.

**Size:** M
**Files:** `model/Umpire/Search.lean` and `Search/` (post-fn-82 home of the engine; `admit`, `AdmittedQuery`, `AdmissionDiagnostic`, `SearchView.retarget`, `AdmittedQuery.withQuery` land here), `model/Umpire/Query.lean`, `Property.lean`, `Scenario.lean` (the `.checked` auto-params), `model/Umpire/Variations/*.lean` (the compiler's per-point transport), `model/Umpire/Exploration/*.lean` (engine, candidate universe, session), `model/Umpire/Promotion.lean` (takes an `AdmittedQuery`), `model/Umpire/Examples/Switch.lean`, `model/Temporal/Feature/Nexus/Success/{Authoring,TypedNexus,TypedUnary}.lean`, `model/Temporal/Feature/Nexus/Race/{Authoring,Cancellation,Race}.lean`, `model/Temporal/Feature/Nexus/Operations.lean` and `Operations/*.lean`, `model/Temporal/Feature/Nexus/Experimental/{VariationSpace,Exploration}.lean`, `model/Umpire/Search/Tests/*.lean`, `Variations/Tests/*.lean`, `Exploration/Tests/*.lean`, `model/UmpireTests.lean`, `model/Umpire/ARCHITECTURE.md`, `model/ARCHITECTURE.md`, `model/README.md`, `tools/umpire/CONTEXT.md`
**Touches:** [model/Umpire/Search.lean, model/Umpire/Search/**, model/Umpire/Query.lean, model/Umpire/Query/**, model/Umpire/Property.lean, model/Umpire/Scenario.lean, model/Umpire/Variations/**, model/Umpire/Exploration/**, model/Umpire/Promotion.lean, model/Umpire/Examples/Switch.lean, model/Temporal/Feature/Nexus/**, model/UmpireTests.lean, model/Umpire/ARCHITECTURE.md, model/ARCHITECTURE.md, model/README.md, tools/umpire/CONTEXT.md]

### Approach
- Read `.plans/LEAN_GUIDELINES.md` first (mandatory for Lean work).
- Baseline: record the `PlanResult` bytes, `CheckedQuery.id` values, search-run scopes, goldens and fingerprints for the Switch example, every Nexus Model and the Variations fixtures before any change; the task's pin is that none of them moves.
- Model `admit` on `Umpire.Case.Correlated.lower` (one admission call returning a proof-carrying record with the obligations discharged inside). `admit` takes the checked Model, the Property, an optional Scenario, the authored `Query` record (form, limits, and the identity and policy fields that `CheckedQuery.id`, the search-run scope and the fingerprints read) and Known Gaps. `AdmittedQuery` is indexed by the checked Model, like the search view, and carries the checked Property, Scenario, Query and `SearchView`; `AdmissionDiagnostic` is one union with one constructor per stage, each carrying that stage's typed error unchanged.
- `admit` lives in `Umpire.Search` because Search imports Query (the search view is Search's type); putting it in Query would invert the `lint-model` import direction.
- Scenario is optional (Property-only Queries admit), because two of the six callers admit that way today.
- Two transports, both inside `Umpire.Search`: `SearchView.retarget` moves a view across a proved Model equality and is the only place an `Eq.mpr (congrArg …)` over a view may remain; `AdmittedQuery.withQuery` re-pairs an admitted base Query's view with another checked Query over the same Model. The five production sites (Variations compiler `planPoint`, Exploration engine and candidate universe, the Nexus operations index, the two experimental Nexus modules) and Promotion all have one view and many Queries, so they take an `AdmittedQuery` and call `withQuery` per point; the thirteen fixture transports go the same way. If a site needs a transport neither operation expresses, record it in the summary rather than keeping an `Eq.mpr` there.
- Add `(ok : … := by native_decide)` to `Property.checked`, `Scenario.checked`, `Query.checked` (and the exact-sequence form if fn-82 keeps one), the way `Umpire.model` already does; measure `lake build` time of the Switch and Nexus targets before and after and keep the delta inside the noise floor you measured first.
- Keep `search` and `SearchView` public for `Umpire.Promotion`; add a visibility test that the search view's proof fields are unreachable except through `search`.
- Migrate the callers and delete their bespoke admission-error unions and every `_isSome` theorem whose only use was the `checked` argument; the Switch example's simp scripts that name nine internals of four modules should disappear with it.
- Docs: the architecture document's raw/check/checked claims and its lifecycle diagram gain the admission node; the model README's walkthrough of the explicit proof argument and its stale `^TestUmpire` live-selector line are corrected; the `Umpire.Query`/`Umpire.Search` facade rows name `admit`; add an `Admitted Query` glossary entry (new Umpire-authoring section in `CONTEXT.md`).

### Investigation targets
**Required** (paths are post-fn-82; the pre-fn-82 evidence lines are in the spec's scan record):
- the search engine module in `Umpire.Search` — the 15-field search-view record, its constructor from a checked Query, `search`, the result match
- `Umpire.Query`, `Umpire.Property`, `Umpire.Scenario` — the `check`/`checked` pairs and the Query record's identity and policy fields
- `model/Temporal/Feature/Nexus/Success/Authoring.lean` — the fullest hand-written chain (checkModel, contexts, three checks, Known Gap canonicalization, query, search view, search, match)
- `model/Umpire/Variations/Compiler.lean` `planPoint` and `model/Umpire/Exploration/Engine.lean` `explore`/`buildCandidateUniverse` — one view, many Queries; the shape `withQuery` must serve
- `model/Umpire/Promotion.lean` — the search-view consumer that keeps `search` public and switches to `AdmittedQuery`
- `model/Umpire/Examples/Switch.lean` — the ordering-obligation simp scripts
- `model/Umpire/Case/Correlated.lean` `lower` — the admission-record pattern

**Optional:**
- `model/Temporal/Feature/Nexus/Operations.lean`, `Nexus/Experimental/VariationSpace.lean`, `Nexus/Experimental/Exploration.lean` — the other three production transports
- `model/Umpire/Search/Tests/Fixtures.lean`, `Variations/Tests/*`, `Exploration/Tests/*` — fixtures opening with `Eq.mpr` transports
- `model/Umpire/Model/Check.lean` — `Umpire.model`'s auto-param
- `Makefile` structural facade gate (each package root file and verbatim facade import)

### Key context
- Avoid retired tokens (`Kernel*`, `Target*`, `Planning` names) in every new identifier; fn-82's spellings are `Search`, `SearchView`, `PlanResult`, `Branch*`.
- `make lint-model` rebuilds every owned source; a new module must be imported from a root lib.
- fn-31 already deepened the checked Model so examples stopped assembling completeness evidence; this task finishes the chain above it.
- 2026-09-10: start only after fn-83 .15 is done. fn-83 .10 moved the success authoring chain into `Umpire.Command`, and fn-83 .14 and .15 still edit that module; Flow cannot express a cross-spec task dependency, so this note carries it. fn-85 builds on the `admit` this task adds.
## Acceptance
- [ ] `Umpire.Search.admit` takes the checked Model, Property, optional Scenario, the authored Query record and Known Gaps and returns `Except AdmissionDiagnostic AdmittedQuery`; `AdmittedQuery.search`, `.searchWithIntent`, `.analyzeBranches`, `.withQuery`, `SearchView.retarget` and `AdmissionDiagnostic.located` exist; `AdmittedQuery` is indexed by the checked Model
- [ ] the six Temporal callers and the Switch example obtain results through `admit`; their admission-error unions are deleted; the Variations compiler, Exploration engine, candidate universe, Exploration session and Promotion take an `AdmittedQuery` and use `withQuery`; `grep -rn 'Eq.mpr (congrArg' model --include='*.lean'` matches only the body of `SearchView.retarget`; no `_isSome` theorem remains whose only use was a `checked` argument
- [ ] `Property.checked`, `Scenario.checked`, `Query.checked` take the `native_decide` auto-param; measured `lake build` delta on the Switch and Nexus targets recorded and within the noise floor
- [ ] each stage's rejection surfaces as its own `AdmissionDiagnostic` constructor with the stage's typed error unchanged; a Property-only Query admits; a visibility test shows the search view's proof fields are unreachable outside `Umpire.Search` except via `search`
- [ ] `CheckedQuery.id` values, search-run scopes, admit-then-search `PlanResult` bytes, goldens and fingerprints are byte-identical to the baseline for the Switch example and every Nexus Model; the Variations goldens do not move
- [ ] focused: `lake build Umpire.Query.Tests Umpire.Search.Tests Umpire.Search.VisibilityTests` (fn-82's names) and `lake build UmpireTests` green; `make lint-model` green
- [ ] architecture documents, model README (walkthrough and live selector), facade rows and `CONTEXT.md` updated; documentation gate passes; `make umpire-check-regression` green
## Done summary
`Umpire.Search.admit` (new module `Umpire/Search/Admission.lean`) now runs the whole check chain: Property, optional Scenario, Known Gaps, Query, then search view. It returns either one `AdmissionDiagnostic`, with one constructor per stage and that stage's typed error unchanged, or an `AdmittedQuery` indexed by the checked Model. The Command success chain, Race, Race.Cancellation, Race.Authoring and the Switch example admit through it. The Variations compiler, Exploration engine, candidate set, session and Promotion take an `AdmittedQuery` and re-pair Queries with `withQuery`. `Property`/`Scenario`/`Query.checked` default their proof to `native_decide`.

stage: impl-review - ran (claude backend, SHIP first round; receipt /tmp/impl-review-receipt-3684356e61f5-fn-84-deepen-five-shallow-module-clusters-in.3.json)

Equivalence pin (R6): a before/after capture was taken at base and after the change. It covered query ids, canonical metadata, fingerprints, `PlanResult` reprs and branch-analysis scopes for Switch, the Command success Model, Race (9 questions), Race.Cancellation, Race.Authoring, the three Operations, and the experimental VariationSpace batch/metadata and Exploration runs/session. The two captures are byte-identical (`.flow/tmp/fn84.3-pin-{base,after}.txt`). Goldens, Case fixtures and Variations goldens are unchanged under `make umpire-check-regression`.

Gates: `make lint-model` 163 (same inherited count); `make lint-code` 161 after `go clean -cache` (14505 → 161, no Go changes); `make umpire-check-regression` exit 0 (nine passing live identities); retired-vocabulary gate clean. `grep 'Eq.mpr (congrArg'` matches only `SearchView.retarget`.

Build time (`lake env lean`, 3 runs, base → after): Switch 2.57-2.60s → 2.58-2.60s; AsyncStart 1.14-1.15 → 1.14-1.16; Operations Cancellation/SuccessfulCompletion, Race, Race.Cancellation and Success.Model are unchanged within ±0.02s. The noise floor is about 0.04s.

Decisions taken autonomously / deviations:
- TypedNexus and TypedUnary are not migrated, and their `AdmissionError` unions stay. They check a field or correlated Property through `CheckedFieldProperty.check` with field bindings, among Operation/Parameter/Field/Projection stages. They never form a Query or search, so routing them through `admit` would add stricter checks (memory: behavior-neutral refactors must not strengthen validation).
- Model-stage errors stay outside `admit`, because `admit` takes an already checked Model. The Command, Race, Race.Cancellation and Race.Authoring unions shrink to `invalidTarget`, `invalidVocabulary` and one `AdmissionDiagnostic` constructor rather than disappearing. A search Known Gap composition error maps to `.knownGaps`, so Command's diagnostic text is byte-identical.
- The real API differs from the spec's sketch. `admit` takes a `Query.Shape` (authored Query fields minus the Property and Scenario, with `form : CheckedProperty → Query.Form`) and `knownGaps : List KnownGap`. Gap canonicalization is a stage, which keeps Command's check precedence. `AdmittedQuery.search`/`searchWithIntent` return `Except` like `search`/`searchWithPlanRequest`. `located` returns an `AdmissionLocation` (stage, definition id, optional source path), since `LocatedError` is Model-specific. `AdmittedQuery.retarget` was added for the Exploration space-equality transport; it delegates to `SearchView.retarget`.
- Property-only admission searches `Query.Shape.unconstrainedScenario`, named `<query id>.scenario`. It declares the Model's setup roles, because a role-less Scenario admits no candidates.
- `Switch.incrementalKernel` and its simp scripts are deleted. Switch admits `exactActionAdmitted` once and runs the exploratory and exact-trace Queries through `withQuery`, which keeps elaboration time at base. Every Switch-kernel fixture now uses that admission.
- The experimental VariationSpace and Exploration modules admit the AsyncStart Query inside their fallible preparation (`baseAdmission`, new `VariationSpacePreparationError.admission`). This avoids a new production `native_decide` witness and its roughly 0.06s elaboration cost. COVERAGE.md records the trust change.
- Specimens whose contract the auto-param changed were rewritten to pin the new behavior: an invalid declaration now fails the default proof. The earlier specimens had vacuous `#guard_msgs` and passed without any error. Affected: SwitchTests property/behavior, Query/Tests/Validation, and Operations/SearchTests (three specimens merged into one).
- The `CheckedModel.kernel` field was deleted; it had no reader.

Touches extensions: `model/Umpire/Command/Authoring.lean` and `model/Umpire/Command/Syntax.lean` (post-fn-83 home of the success chain; the `query` command now passes the authored gap list), `model/Umpire/Examples/SwitchTests.lean` (auto-param pin), `model/Umpire/PromotionTests.lean` (Promotion signature), `model/Temporal/Feature/Nexus/Success/Tests.lean` (constructor matches), `model/Temporal/Feature/Nexus/COVERAGE.md` (trust inventory). `model/Umpire.lean` was not edited; `Search.Admission` is reachable through Promotion and Variations.

Follow-ups (not done):
- Retiring the deleted public names (`invalidPlanner` family, `Switch.incrementalKernel`) needs the vocabulary list in `tools/umpire/internal/retiredvocabulary`, which is outside Touches.
- Review P3s left open after SHIP:
  - `AdmittedQuery.property` is kept, not re-derived, under `withQuery`.
  - `baseAdmission` copies AsyncStart's query fields; an `AsyncStart.queryShape` would remove the copy.
  - `Query.Shape` could move into `Umpire.Query`.
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 714200935a784c11ddfd0000d4638868c84c38e3
- Tests: cd model && mise exec -- lake build Temporal UmpireTests TemporalModelTests TemporalExperimentalTests +Umpire.PromotionTests (includes Umpire.Query.Tests, Umpire.Search.Tests, Umpire.Search.VisibilityTests), make lint-model (163, inherited, all in Temporal/API/Proto.lean), go clean -cache && make lint-code GOLANGCI_LINT_FIX=false (161 issues, inherited baseline; no Go files touched), make umpire-check-regression (exit 0; 574 Lean jobs; nine passing live TestTestpilot identities), equivalence pin: lake env lean Pin.lean before/after, byte-identical (.flow/tmp/fn84.3-pin-base.txt vs fn84.3-pin-after.txt), baseline: green (regression via receipt eabc8bfc; lint-model 163 measured pre-edit)
- PRs: