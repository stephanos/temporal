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
Blocked:
BLOCKED: SCOPE_EXCEEDED - session budget, not a technical obstacle.

Two of the task's five acceptance bullets are landed, green and committed; the other three
are a coherent unit I did not want to start and abandon mid-flight. Every gate is at baseline
at HEAD, so the tree is safe to pick up from.

Landed (commits 6db43bf3a7, ff7bb6c3cd, a73a492c96, f5452b241d, 67edfd3851):
- `Umpire.Search` replaces `Umpire.Planning`. `Planning/{Engine,Types,CaseAnalysis}.lean` are
  `Search.lean` and `Search/{Types,Branches}.lean`; the tests moved with them.
  `IncrementalPlannerKernel` is `SearchView`, `PlannerInstrumentation` is `SearchStats`,
  `PlannerRun` is `PlanResult`, `FinitePlannerAdmissionError` is `FiniteSearchAdmissionError`,
  the case-analysis vocabulary is `Branch*`, the joint-obligation vocabulary is `Overlap*`,
  `analyzeCases` is `analyzeBranches`, and the entry point `plan` is `search`. The
  `PlanningOutcome` constructors are `noneFound`, `neverTriggered` and `stillPending` with
  matching canonical names. `Umpire.ExecutionHandoff` and its tests are deleted.
  `Temporal.Feature.Nexus.Operations.Planning` is `.Search` so the feature layer stops naming
  a retired module.
- `ExperimentSpec` is `Plan`, `DrivePlan` is `Plan.Steps`, `ArtifactIntent`/`ArtifactFaultIntent`
  are `PlanRequest`/`RequestedFault`, and `artifactv2`'s Go types follow with every JSON tag
  untouched. `umpire-experiment/v2` and `umpire-drive-plan/v2` are byte-identical and
  `umpire-check-regression-views` passes.
- The gate rejects `Umpire.Planning`, `Umpire.ExecutionHandoff`, `PlannerRun`,
  `PlannerInstrumentation`, `IncrementalPlannerKernel`, `FinitePlannerAdmissionError`,
  `ExecutionHandoff`, `ExperimentSpec`, `DrivePlan`, `ArtifactIntent`, `ArtifactFaultIntent`
  and `analyzeCases`; the scanned Umpire4 documents and open downstream spec records are
  respelled.

Remaining, in the order I would do it:
1. `LimitUnit` in `Umpire/Core.lean:109-123`: `semanticTransitions`->`steps`,
   `selectedActions`->`actions`, `candidateEvaluations`->`search`, `experimentSpecs`->`plans`,
   with the canonical strings following. `observationPositions` is DELETED, and that is the
   only semantic edit in the task: it has three live consumers
   (`Property/Evaluate.lean:846` coordinate lookup, `Evaluate.lean:2191` totals,
   `Property/Check.lean:671` field validation) plus six test fixtures that use it as a clause
   limit; each of those clauses must move to `steps` and be re-baselined. Wire strings appear
   in ten JSON fixtures and in `tools/umpire/internal/artifactv2/artifact.go:731`, so the
   Lean ones regenerate through `lake exe umpire-goldens` and the Go reader list must follow
   in the same commit (this is exactly the class of miss the `.2`/`.3` review caught).
2. Query: `QueryDeclaration`+`QuerySpec` collapse to one `Query` with `check`/`checked`;
   delete `QueryAuthoringInput` and its second `query%` elaborator, `QueryQuantifier`,
   `QueryClaim`, `TieBreakPolicy`; `QueryForm` becomes `Query.Form` with
   `verify`/`find`/`findViolation`/`pick` and matching JSON at `Query/Language.lean:487-490`;
   `QueryEndpoint` becomes `Query.Ending` with `partial`/`final`/`terminal`;
   `QueryExercisePolicy` becomes `requireFiring : Bool`; `CheckedQueryTarget` becomes
   `ModelCompleteness`; `QueryLimits`+`BehaviorPhaseLimits` flatten to one
   `Limits {steps, actions, search}` and `QueryLimitSpec` becomes `Limits`. Note that
   `.partial` needs French quotes (`«partial»`) because `partial` is a Lean keyword - task .4
   already spells `PropertyScopedEndpoint.«partial»` that way.
3. Query's own gate entries (`QueryDeclaration`, `QuerySpec`, `QueryAuthoringInput`,
   `QueryLimitSpec`, `QueryQuantifier`, `QueryClaim`, `TieBreakPolicy`, `find-witness`,
   `find-counterexample`, `select-behavior`, `semantic-transitions`, `selected-actions`,
   `candidate-evaluations`, `experiment-specs`) plus the Makefile Query layout arm, which
   still asserts the old `Umpire/Query/Language.lean` shape.

Method notes for the next session, all verified here:
- Restrict every sweep to `model/`, `tools/umpire/`, `common/testing/testpilot/`, `tests/`,
  `.plans/UMPIRE4_*.md` and the gate's `downstreamSpecs` records. A repo-wide walk rewrote an
  unrelated `BehaviorTrace` in `tools/common/formal/trace` on the first attempt.
- A rename that lands inside a `#guard_msgs` docstring's embedded line/column numbers must be
  re-baselined from the build log, not by hand; the same is true of every inline fingerprint.
- `make lint-code` must be re-checked after any Go error-string respelling: staticcheck ST1005
  fires on a message that starts with a single capitalized word.
## Evidence
- Commits: 6db43bf3a771db96fc2a94bbc10f8be11a0e7a9b, ff7bb6c3cdef82fabc52911b2e34623d2fdaf1d4, a73a492c962279e7f90a12e8cc73859792e03cef, f5452b241dd244df44a7ea5cba4b7a3f7141c3be, 67edfd3851b7f3d1ddb4de41038b94f7ca8efd35
- Tests: cd model && lake build Umpire UmpireTests Temporal TemporalModelTests TestpilotTests Testpilot TemporalExperimentalTests +Umpire.PromotionTests (pass, 402 jobs); make lint-model (169 diagnostics, all generated Temporal/API; equals baseline); make umpire-check-goldens / -regression-views / -case-runtime-conformance / -semantic-inventory / -retired-vocabulary / -lean-api / -testpilot-protocol / -testpilot-authoring (pass); make umpire-check-live-tests (pass); TMPDIR=<physical> CGO_ENABLED=0 go test -count=1 -tags test_dep ./tools/umpire/... ./common/testing/testpilot/... ./tests/testcore/testpilot/... (pass); make lint-code GOLANGCI_LINT_FIX=false (128 findings, equals baseline); CC=/usr/bin/cc go vet -tags test_dep ./... (15 diagnostics, equals baseline)
- PRs:

stage: plan-sync - skipped(config: planSync.enabled != true)
