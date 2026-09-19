---
satisfies: [R3]
---
# fn-82-unify-the-umpire-and-testpilot.5 Query forms, flat Limits, Search, and Plan

## Description
Query, Limits, Search, and Plan (R3, spec §R3): `Query` becomes the one authored record with
`verify`/`find`/`findViolation`/`pick` forms, limits flatten to one record with five units,
`Umpire.Planning` becomes `Umpire.Search`, `ExperimentSpec`/`DrivePlan` become `Plan`/`Plan.Steps`
with unchanged wire identifiers, and `Umpire.ExecutionHandoff` is deleted.

**Size:** M (mechanical sweep across many files)
**Files:** `model/Umpire/Query.lean`, `model/Umpire/Query/{Check,Elab}.lean`, `model/Umpire/Core.lean` (`LimitUnit`), `model/Umpire/Search.lean`, `model/Umpire/Search/{Types,Branches}.lean`, `model/Umpire/Artifact/{Types,Codecs,Planning}.lean`, `model/Umpire/Promotion.lean`, deletions of `Umpire/ExecutionHandoff*.lean` and `Umpire/Planning/*`, `model/ModelLint/ImportGraph.lean`, `Makefile`, `tools/umpire/internal/artifactv2/*.go` (type names only), docs, gate
**Touches:** [model/Umpire/Query*, model/Umpire/Planning*, model/Umpire/Search*, model/Umpire/Artifact/**, model/Umpire/Core.lean, model/Umpire/Promotion*, model/Umpire/ExecutionHandoff*, model/Umpire/Examples/**, model/Temporal/**, model/ModelLint/**, Makefile, tools/umpire/internal/artifactv2/**, tools/umpire/cmd/umpire-gen-regression-views/**, .flow/specs/*.md]

### Approach
- Query: `QueryDeclaration`+`QuerySpec` to `Query`; delete `QueryAuthoringInput` and its second `query%` elaborator, `QueryQuantifier`, `QueryClaim`, `TieBreakPolicy`; `QueryForm` to `Query.Form` with `verify`/`find`/`findViolation`/`pick` and JSON spellings at `Query/Language.lean:487-490`; `QueryEndpoint` to `Query.Ending` with `.partial`/`.final`/`.terminal` (distinct from the two-valued `TraceEnding` task .7 introduces on correlated rules); `QueryExercisePolicy` to `requireFiring : Bool`; `CheckedQueryTarget` to `ModelCompleteness`; `QueryLimits`+`BehaviorPhaseLimits` to one `Limits {steps, actions, search}`; `QueryLimitSpec` to `Limits`.
- `LimitUnit` in `Core.lean:118-133`: `steps`, `actions`, `logicalTime`, `search`, `plans`; delete `observationPositions`. Wire strings (`"semantic-transitions"` etc.) appear in ten JSON fixtures under `model/Temporal/Feature/Nexus/Fixtures`, `model/Umpire/Examples/Fixtures`, `model/Umpire/Artifact/Tests/Fixtures` and in `tools/umpire/internal/artifactv2/artifact.go:725`; regenerate the Lean ones with `lake exe umpire-goldens` (task .2's writer covers all three directories) and update the Go reader.
- Search: `Planning/Engine.lean` to `Search.lean`, `Planning/Types.lean` to `Search/Types.lean`, `Planning/CaseAnalysis.lean` to `Search/Branches.lean` with `analyzeBranches`, `Branch*`, `Overlap*` (was `Joint*`); `IncrementalPlannerKernel` to `SearchView`, `PlannerInstrumentation` to `SearchStats`, `PlannerRun` to `PlanResult`, `PlanningOutcome` constructors per spec (`noneFound`, `neverTriggered`, `stillPending`), `plan` to `search`.
- Plan: `ExperimentSpec` to `Plan`, `DrivePlan` to `Plan.Steps`, `ArtifactIntent`/`ArtifactFaultIntent` to `PlanRequest`/`RequestedFault`; keep `umpire-experiment/v2` and `umpire-drive-plan/v2` byte-identical; rename Go types in `artifactv2` without touching JSON tags.
- Delete `ExecutionHandoff.lean` and its tests (the `SwitchExperimentSpecV2.json` golden it includes stays for Artifact tests). Update `semanticRoots`, the Makefile `Planning/Engine.lean` assertion (`Makefile:1235-1239`), and `Promotion.lean:148` (renders `import Umpire.Planning.Engine` into generated source; the rendered bytes and `promotionSourceSha256` change).

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Query/Language.lean:11-60,90-140,259-300,334-360,487-530,556-612` — limits, forms, endpoint, checked record, JSON, entry points
- `model/Umpire/Planning/Engine.lean:35-80,384-420,485-540,663-700,1092-1140` — kernel view, outcome, run, entry points
- `model/Umpire/Artifact/Types.lean:39-100` — `DrivePlan`, `ExperimentSpec`, intents
- `tools/umpire/internal/artifactv2/artifact.go:17-65,700-730` — format ids, JSON tags, limit unit strings
- `model/Umpire/Core.lean:118-137` — `LimitUnit` and `Limit`

**Optional** (reference as needed):
- `model/Umpire/Planning/CaseAnalysis.lean:14-200` — types to rename `Branch*`/`Overlap*`
- `tools/umpire/regression/ci_workflow_test.go:622` — asserts the rendered `import Umpire.Planning.Engine`

### Key context
- `verify` stays as the Query form because `check` is the construction method on `Query`.
- Retire `QueryDeclaration`, `QuerySpec`, `QueryAuthoringInput`, `QueryLimitSpec`, `QueryQuantifier`, `QueryClaim`, `TieBreakPolicy`, `Umpire.Planning`, `PlannerRun`, `IncrementalPlannerKernel`, `ExperimentSpec`, `DrivePlan`, `ExecutionHandoff`, `find-witness`, `find-counterexample`, `select-behavior`, `semantic-transitions`, `selected-actions`, `candidate-evaluations`, `experiment-specs`.
## Acceptance
- [ ] `Query` with `check`/`checked` and `Query.Form.{verify,find,findViolation,pick}` replaces the four old records and forms; `QueryQuantifier`, `QueryClaim`, `TieBreakPolicy`, `QueryAuthoringInput` are gone
- [ ] `Limits` is one flat record and `LimitUnit` has exactly `steps`, `actions`, `logicalTime`, `search`, `plans`
- [ ] `Umpire.Search` replaces `Umpire.Planning` with `PlanResult`, `SearchView`, `SearchStats`, `Branch*`, `Overlap*`, and `search`; `Umpire.ExecutionHandoff` no longer exists
- [ ] `Plan` and `Plan.Steps` replace `ExperimentSpec` and `DrivePlan`; `umpire-experiment/v2` and `umpire-drive-plan/v2` bytes are unchanged in every fixture and `make umpire-check-regression-views` passes
- [ ] `make lint-model`, the Makefile layout check, Go tests under `tools/umpire`, and the gate pass
## Done summary
All five acceptance bullets are met and every gate is at or under its baseline.

Landed in this session (commits f0d01b7c8b, 3e3e291b6b, f8c430fd94, 928ab49ac7),
completing the work started in 6db43bf3a7, ff7bb6c3cd, a73a492c96, f5452b241d, 67edfd3851:

- `LimitUnit` is exactly `steps`, `actions`, `logicalTime`, `search`, `plans`, with the
  canonical strings following. `observationPositions` is deleted, not renamed: its three
  consumers (the occurrence coordinate lookup, the endpoint total, the field validation)
  all reduce to `steps`, and the `PropertyOccurrence.observationPosition` field plus the
  observation-offset threading through nine functions and four agreement theorems, which
  existed only to answer it, go with them. Every clause that used it now bounds `steps`;
  the Nexus2 race test's terminal trigger coordinate moves from 3 to 2 accordingly.
- `QueryDeclaration` + `QuerySpec` collapse into one authored `Query` whose only
  construction operations are `check` and `checked`. `QueryAuthoringInput` and its second
  `query%` elaborator are deleted; the one remaining elaborator takes the authored Query
  and the Model it asks it of, and the Nexus2 frontend tests build concrete values for it.
  `QueryQuantifier`, `QueryClaim` and `TieBreakPolicy` are deleted -- the form constructor
  already says what the Query claims, so `Query.Form.name` replaces all three -- and
  `QueryExercisePolicy` becomes the `requireFiring : Bool` it always was.
- `QueryForm` is `Query.Form` with `verify`/`find`/`findViolation`/`pick` and JSON
  `verify`/`find`/`find-violation`/`pick`; `QueryEndpoint` is `Query.Ending` with
  `partial`/`final`/`terminal`; `CheckedQueryModel` is `ModelCompleteness`.
- `QueryLimits`, `BehaviorPhaseLimits` and `QueryLimitSpec` flatten into one
  `Limits {steps, actions, search}`, so the query and artifact JSON lose the nested
  `behavior` object and Go's `artifactv2.Limits` follows.
- Module layout follows Property and Scenario: `Umpire/Query.lean` (types),
  `Query/Check.lean` (canonicalization and checking), `Query/Elab.lean` (the
  located-diagnostic elaborator). `Query/Language.lean` and `Query/Authoring.lean` are
  deleted, the Makefile package-layout arm folds Query into the Property/Scenario loop,
  and `ModelLint` semanticRoots plus its tests follow.
- `umpire-experiment/v2` and `umpire-drive-plan/v2` remain byte-identical as identifiers.
  All seventeen goldens, the Case conformance fixtures, the Generated Views and every
  inline fingerprint literal regenerated through their owning targets; no golden was
  hand-edited.
- The gate rejects `QueryDeclaration`, `QuerySpec`, `QueryAuthoringInput`, `QueryLimitSpec`,
  `QueryQuantifier`, `QueryClaim`, `QueryExercisePolicy`, `BehaviorPhaseLimits`,
  `TieBreakPolicy`, `CheckedQueryTarget`, `CheckedQueryModel`, `checkQuery`,
  `Umpire.Query.Language`, `Umpire.Query.Authoring`, and the retired wire strings
  `find-witness`, `find-counterexample`, `select-behavior`, `semantic-transitions`,
  `selected-actions`, `observation-positions`, `candidate-evaluations`, `experiment-specs`.
  Because the gate also bans the lowerCamel variant, the local `queryDeclaration` values
  are `authoredQuery`, matching `.4`'s `authoredProperty`/`authoredScenario`.
- The P3 the spec completion review left on the landed work is closed: Go's
  `artifactv2.Experiment` (and `DecodeExperiment`, `SealExperiment`, `ValidateExperiment`,
  `CanonicalExperimentBytes`, `ExpectedExperimentChecksum`, `VerifyExperimentChecksums`,
  `ExperimentFormat`) is `Plan`, with every JSON tag and wire identifier untouched.

Review: SHIP with three P3s (a stale `search.candidateEvaluations` fixture string, a
half-rewritten Nexus2 README sentence, and the planning receipt's unchanged `formatVersion`
tag). All three are fixed in 928ab49ac7, along with three identifiers the sweep left
spelled "QueryDeclaration".

Deliberately left to their owning tasks: the Nexus3 `limits` macro still spells its
keywords `transitions`/`selected_actions`/`candidate_evaluations` (task .8 owns the command
keywords), and `Query.Ending`'s docstring forward-references the `TraceEnding` task .7
introduces.

Swept in, not authored here: `.plans/UMPIRE4_ORDER.md` progress prose from a parallel
session. Its "Landed:" paragraph named retired vocabulary the gate scans, so it is
respelled to describe the same renames without the retired identifiers.
## Evidence
- Commits: 6db43bf3a771db96fc2a94bbc10f8be11a0e7a9b, ff7bb6c3cdef82fabc52911b2e34623d2fdaf1d4, a73a492c962279e7f90a12e8cc73859792e03cef, f5452b241dd244df44a7ea5cba4b7a3f7141c3be, 67edfd3851b7f3d1ddb4de41038b94f7ca8efd35, f0d01b7c8b, 3e3e291b6b, f8c430fd94, 928ab49ac7
- Tests: cd model && lake build Umpire UmpireTests Temporal TemporalModelTests TestpilotTests Testpilot TemporalExperimentalTests +Umpire.PromotionTests (pass, 402 jobs), make umpire-check-regression (pass; goldens, regression-views, testpilot-protocol, testpilot-authoring, case-runtime-conformance, semantic-inventory, retired-vocabulary, lean-api, live-tests, the model layout assertions, and go test ./tools/umpire/... ./common/testing/testpilot/... ./tests/testcore/testpilot/...), make umpire-build-model (pass, includes the Makefile package-layout check), make lint-model (169 diagnostics, all generated Temporal/API; equals baseline), make lint-code GOLANGCI_LINT_FIX=false (126 findings; baseline 128), CC=/usr/bin/cc go vet -tags test_dep ./... (15 diagnostics; equals baseline), go run ./tools/planindex (48 lines; equals baseline)
- PRs:
stage: plan-sync - skipped(config: planSync.enabled != true)
