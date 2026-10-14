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
Migrated AsyncStart, Cancellation, and SuccessfulCompletion through the established PropertySpec, transition-result, ExactSequenceSpec, QuerySpec, QueryLimitSpec, and checked-query planner APIs. Each operation retains its public declarations and raw checker/admission results, supplies explicit checked evidence, and preserves exact identities, source, metadata, fingerprints, traces, outcomes, artifacts, and compatibility entry points; executable tests cover invalid clauses/capabilities, contradiction, Target mismatch, invalid Limits, omitted proofs, and missing completeness.

The checked-query adapter remains unchanged. A reusable Engine theorem proves successful extraction from its explicit Target, completeness, finite-domain, and canonical-order premises; all three operations call `IncrementalPlannerKernel.ofCheckedQuery` directly. Constructor work is record assembly plus one transition-clause map and the existing checker/admission passes, with no new nested scan, frontend, adapter, registry, I/O, dependency, or Observation/Known-Gap migration.

Verification: the focused 68-job build and `make lint-model` passed. The trust audit reports only kernel axioms for the Engine theorem and the exact historical Nexus native baseline for checked declarations; all three possible `incrementalKernelResult_isSome` native axiom names are absent. Go lint retained inherited exit 2 with exactly 1,316 sorted diagnostic headers, byte-identical to the approved baseline at SHA-256 `aee7770bec1fe01dab8826427cc89e9ffa7e764fbac25ce6b68bf5f2e3c0b077`. The official staged-overlay review returned SHIP with zero findings; review tree and frozen pre-receipt staged tree are both `0dbc3b8614889317a2682f90ddebe2d1ed481b11`, and frozen source blobs match after gates, review, and receipt staging.

No commit was created under the user's standing commit policy; HEAD remains `7774fdc7ac751ac959816c9829516ce54af57194` and all cumulative and unrelated work remains preserved.

stage: impl-review - ran [2026-09-06T03:55:41Z..2026-09-06T04:00:13Z] (SHIP; actual model codex:gpt-5.6-sol:medium; receipt /tmp/impl-review-receipt-fn-62-make-ordinary-temporal-model-authoring.4.json)
stage: plan-sync - skipped(config: planSync.enabled != true)
stage: tracker-sync - skipped(config: tracker.enabled != true; inactive confirmed)
## Evidence
- Commits:
- Tests: baseline: cd model && mise exec -- lake build Temporal.Feature.Nexus.OperationsTests Umpire.Planning.Tests (green; 68 jobs; /tmp/fn62-task4-baseline.log), cd model && mise exec -- lake build Temporal.Feature.Nexus.OperationsTests Umpire.Planning.Tests (green; 68 jobs; /tmp/fn62-task4-final-focused.log), trust audit via PlanningTests #print axioms (Engine ofCheckedQuery_isSome uses only propext/Classical.choice/Quot.sound; operation checked values retain exact historical step_result_exposed/target and property/behavior/query result native axioms; all three incrementalKernelResult_isSome native axiom names absent; /tmp/fn62-task4-final-focused.log), make lint-model (green; 258 jobs; /tmp/fn62-task4-final-lint-model.log), make lint-code GOLANGCI_LINT_FIX=false (inherited exit 2; exactly 1316 sorted diagnostic headers; normalized SHA-256 aee7770bec1fe01dab8826427cc89e9ffa7e764fbac25ce6b68bf5f2e3c0b077; byte-identical to /tmp/fn62-task2-final-lint-code.headers; /tmp/fn62-task4-final-lint-code.log), git diff --cached --check -- task-owned paths (green), flowctl gate classify --base 7774fdc7ac751ac959816c9829516ce54af57194 (FULL; cumulative unmatched .plans/UMPIRE4_ORDER.md); gate receipt non-blocking unavailable because unrelated staged work makes receipt unwarrantable, impl-review codex:gpt-5.6-sol:medium (SHIP; zero findings; /tmp/impl-review-receipt-fn-62-make-ordinary-temporal-model-authoring.4.json), reviewed staged tree 0dbc3b8614889317a2682f90ddebe2d1ed481b11 equals frozen pre-receipt staged tree; owned source blobs equal after gates, review, and receipt staging, tracker inactive confirmed (.flow/config.json tracker.enabled=false); plan-sync skipped (.flow/config.json planSync.enabled=false)
- PRs:
