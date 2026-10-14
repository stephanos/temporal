---
satisfies: [R3, R5, R7]
---
# fn-39-make-the-temporal-nexus-feature-model.2 Split Operations by walkthrough and planning concern

## Description
Move each complete operation walkthrough and the shared planner machinery into focused child modules behind the existing Operations facade (R3, R5, R7). This is a mechanical decomposition with no declaration or artifact change.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/Operations.lean`, `model/Temporal/Feature/Nexus/Operations/Internal.lean`, `model/Temporal/Feature/Nexus/Operations/AsyncStart.lean`, `model/Temporal/Feature/Nexus/Operations/Cancellation.lean`, `model/Temporal/Feature/Nexus/Operations/SuccessfulCompletion.lean`, `model/Temporal/Feature/Nexus/Operations/Planning.lean`
**Touches:** [model/Temporal/Feature/Nexus/Operations.lean, model/Temporal/Feature/Nexus/Operations/**]

### Approach
- Keep only documented aggregation in `Operations.lean`; place the existing shared source/role/declaration mechanics in a narrowly named internal child without creating a new authoring language.
- Move the incremental kernel and other shared deterministic-planning machinery into `Operations/Planning.lean`; it imports only Lifecycle/Internal support and no walkthrough or aggregate module.
- Move each complete Property → Behavior → Query → deterministic run slice into its matching child while retaining the current `Temporal.Feature.Nexus.Operations.<Walkthrough>` namespace and declaration order; each walkthrough imports the lower Planning seam.
- Preserve every current public declaration, `Operations.source`, Definition ID, canonical behavior, intended/negative trace, deterministic run, and comment.
- Keep child imports directed toward Lifecycle/Internal/Planning; no child imports the Operations facade, and Planning does not import a walkthrough.
- Add concise module and read-next documentation to the Operations facade and children while preserving every existing declaration comment.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Feature/Nexus/Operations.lean:10-45` — shared source and authoring mechanics.
- `model/Temporal/Feature/Nexus/Operations.lean:47-162` — Async Start walkthrough.
- `model/Temporal/Feature/Nexus/Operations.lean:164-198` — shared incremental planner currently between concerns.
- `model/Temporal/Feature/Nexus/Operations.lean:200-447` — Cancellation and Successful Completion walkthroughs.
- `.flow/tasks/fn-38-consolidate-layered-model-helpers.4.md:13-32` — predecessor helper and exact-source constraints.

**Optional** (reference as needed):
- `model/Temporal/Tool/Inspect.lean:79-90` — all three planned artifact consumers.

### Acceptance
- [ ] Operations facade aggregates the focused concerns and all existing qualified declarations remain available.
- [ ] Each walkthrough is independently readable in one file through Property → Behavior → Query → deterministic run; Planning owns only shared machinery.
- [ ] Operations facade and child module docs explain the walkthrough and lower Planning seam without removing or rewriting existing declaration comments.
- [ ] Existing IDs, source, fingerprints, Query JSON, planner outcomes, Artifact bytes, traces, and comments are unchanged.
- [ ] `cd model && mise exec -- lake build Temporal.Feature.Nexus.OperationsTests Temporal.Tool.InspectTests TemporalModelTests` passes.

## Acceptance
- [ ] R3, R5, and R7 task-scoped checks pass.
- [ ] No unrelated worktree file is modified.

## Done summary
Split the stable Nexus Operations facade into focused Async Start, Cancellation, Successful Completion, Internal, and Planning modules while retaining every existing qualified declaration, comment, source identity, plan, Query, and pretty-canonical Artifact byte. Each walkthrough now keeps its complete Property → Behavior → Query → deterministic run slice together, and the lower Planning seam imports no walkthrough or aggregate module.

The focused acceptance build, complete Umpire regression, and model lint are green. Gate receipt writes were non-blockingly unwarrantable because the protected inherited false-symlink stat entry kept the worktree dirty.

stage: impl-review - ran (SHIP)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 43118f9175637c50d475134b8d4912199c3340c4
- Tests: baseline: green, git diff --check, cd model && mise exec -- lake build Temporal.Feature.Nexus.OperationsTests Temporal.Tool.InspectTests TemporalModelTests, make umpire-check-regression, make lint-model, import boundary scan: child modules import only Planning; Planning imports only Internal; no child imports the Operations facade; Planning imports no walkthrough, gate receipt: NO_RECEIPT (protected inherited false-symlink stat entry kept worktree dirty; receipt failure is non-blocking)
- PRs:
