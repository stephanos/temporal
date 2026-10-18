---
satisfies: [R1, R4, R5, R8]
---
# fn-62-make-ordinary-temporal-model-authoring.4 Migrate established Nexus operation constructors and planning

## Description
Migrate AsyncStart, SuccessfulCompletion, and Cancellation through fn-65's existing constructor APIs and checked planner adapter. Consume `.2`'s Lifecycle and `.3`'s shared identities.

**Size:** M
**Files:** established Nexus `Operations` declarations, planning and tests
**Touches:** [model/Temporal/Feature/Nexus/Operations/**, model/Temporal/Feature/Nexus/Operations.lean]

## Approach
- Use `PropertySpec`, `transitionResultClauses`, `ExactSequenceSpec`, `QuerySpec`, and `QueryLimitSpec`, retaining author-supplied `.checked` evidence and existing raw error results.
- Replace local dependent planner transport with `IncrementalPlannerKernel.ofCheckedQuery`; preserve explicit mismatch/absence handling instead of assuming success.
- Keep existing public operation declarations and comments. Use no prototype frontend or new generic wrapper. Preserve source data unless an exact correction is recorded.
- Compare all three operation identities, complete metadata/fingerprints, selected traces, planning outcomes and exact artifacts with pre-migration baselines.
- Use explicit kernel proofs or existing approved proof evidence. Failed fn-65 kernel synthesis is not permission for a new native fallback; audit each migrated checked declaration's transitive dependencies.

## Investigation targets
**Required:**
- `model/Temporal/Feature/Nexus/Operations/Internal.lean` — repeated inputs.
- `model/Temporal/Feature/Nexus/Operations/Planning.lean` — transport to remove.
- `model/Temporal/Feature/Nexus/Operations/PlanningTests.lean:17` — exact operation artifacts.
- `model/Umpire/Property/Authoring.lean:64` — proof-taking constructor.
- `model/Umpire/Behavior/Authoring.lean:54` — sequence checker seam.
- `model/Umpire/Query/Authoring.lean` — named Limits and checked Query.
- `model/Umpire/Planning/Engine.lean:347` — covered adapter.

## Acceptance
- [ ] All three operations use existing constructors with less repeated assembly and explicit checker-success arguments; no new planner adapter or frontend is added.
- [ ] Exact identity, source, metadata, fingerprint, selected trace, outcome, and artifact checks match baseline except specifically named source corrections.
- [ ] Invalid clauses/capabilities, contradiction, Target mismatch, invalid Limits, and omitted proof evidence retain their existing failure boundary; missing finite completeness remains explicit.
- [ ] `cd model && mise exec -- lake build Temporal.Feature.Nexus.OperationsTests Umpire.Planning.Tests` passes; trust and lint checks show no new issues.

## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: