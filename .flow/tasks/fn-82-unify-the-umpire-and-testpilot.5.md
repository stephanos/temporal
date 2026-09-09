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
- Query: `QueryDeclaration`+`QuerySpec` to `Query`; delete `QueryAuthoringInput` and its second `query%` elaborator, `QueryQuantifier`, `QueryClaim`, `TieBreakPolicy`; `QueryForm` to `Query.Form` with `verify`/`find`/`findViolation`/`pick` and JSON spellings at `Query/Language.lean:487-490`; `QueryEndpoint` to `TraceEnding` with `.partial`/`.final`/`.terminal`; `QueryExercisePolicy` to `requireFiring : Bool`; `CheckedQueryTarget` to `ModelCompleteness`; `QueryLimits`+`BehaviorPhaseLimits` to one `Limits {steps, actions, search}`; `QueryLimitSpec` to `Limits`.
- `LimitUnit` in `Core.lean:118-133`: `steps`, `actions`, `logicalTime`, `search`, `plans`; delete `observationPositions`. Wire strings (`"semantic-transitions"` etc.) appear in ten JSON fixtures under `model/Temporal/Feature/Nexus/Fixtures`, `model/Umpire/Examples/Fixtures`, `model/Umpire/Artifact/Tests/Fixtures` and in `tools/umpire/internal/artifactv2/artifact.go:725`; regenerate the Lean ones and update the Go reader.
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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
