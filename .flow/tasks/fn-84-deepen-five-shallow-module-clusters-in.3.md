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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
