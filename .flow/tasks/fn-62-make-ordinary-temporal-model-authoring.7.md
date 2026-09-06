---
satisfies: [R1, R8, R9]
---
# fn-62-make-ordinary-temporal-model-authoring.7 Publish and verify the complete established authoring journey

## Description
Integrate `.1`–`.6` and `.8` into a compiled established Nexus reader path, exact compatibility evidence, public documentation, and final gates.

**Size:** M
**Files:** model/owner docs, established Nexus public tutorial/tests, import tests and aggregate roots
**Touches:** [model/README.md, model/Umpire/ARCHITECTURE.md, model/Umpire/Target/ImportTests.lean, model/Umpire/Property/ImportTests.lean, model/Umpire/Behavior/ImportTests.lean, model/Umpire/Query/Tests/Visibility.lean, model/Umpire/Observation/ImportTests.lean, model/Umpire/Planning/VisibilityTests.lean, model/Temporal/Feature/Nexus.lean, model/Temporal/Feature/Nexus/**, model/TemporalModelTests.lean]

## Approach
- Make the established Nexus journey itself the compiled public-facade walkthrough: finite Target, Properties, Behaviors, Queries, existing planner, Observation, and authored gaps. Add a concise reader-order guide and locally relevant Lean syntax key, not an unrelated duplicate example family.
- Include representative malformed ID/reference, missing author proof, invalid raw Target/transition and Observation examples in runnable tests.
- Document ordinary versus expert authoring, explicit check/checked seams, stable IDs, composition, outcomes, Limits and nonbehavioral gaps. Keep Nexus2's exception and unmeasured usability boundaries explicit; no broader rule change.
- Verify exact Nexus compatibility and named source/gap deltas. Audit all changed load-bearing declarations, public imports and complete first-party module boundaries.
- Freeze source/tests before final serial gates. Recording results in documentation does not itself justify rerunning unchanged-source gates.

## Investigation targets
**Required:**
- `model/README.md:51` — public reader path.
- `model/Umpire/ARCHITECTURE.md:52` — authoring boundaries.
- `model/Temporal/Feature/Nexus.lean:8` — public model entrypoint.
- `model/Temporal/Feature/Nexus/Operations/PlanningTests.lean` — exact artifacts.
- `model/Temporal/Feature/Nexus/ObservationTests.lean` — Observation compatibility.
- `model/Temporal/Feature/Nexus2/EVIDENCE.md:57` — completed prototype comparison, not established migration evidence.

## Acceptance
- [ ] Public compiled walkthrough and negative specimens cover R1, including actual Observation and Known Gap paths, without internal/Experimental/runtime/verification imports.
- [ ] Exact established compatibility, full error payloads, source/gap deltas and trust inventory are recorded as executable evidence; docs accurately distinguish prototype and production.
- [ ] Serial aggregate `cd model && mise exec -- lake build UmpireTests TemporalModelTests TemporalExperimentalTests` and `make umpire-build-model` pass.
- [ ] Full `make umpire-check-regression` uses physical canonical TMPDIR and passes; `make lint-model` passes; `make lint-code GOLANGCI_LINT_FIX=false` has no new diagnostics against the verified baseline. Go tests use `-tags test_dep`.
- [ ] All task evidence is recorded before task review; fn-62 still requires whole-spec completion review before closure.
- [ ] Consolidate the named constructor/composition complexity audits from `.1`, `.3`, `.5`, `.6`, `.8`, checking every added call/traversal and the stated 1×/10× pass conditions; timing alone or repeated cached admission is insufficient.

## Done summary
Published the compiled established Nexus authoring journey through the public facade: proof-carrying finite Target, checked Property/Behavior/Query and planning, typed Observation evaluation, and checked authored Known Gap publication. The facade tests execute representative malformed identity/reference, missing finite proof, incomplete raw Target, invalid transition, and invalid Observation failures; the reader guide and tracked evidence record exact compatibility, source/gap boundaries, transitive trust, Nexus2's separate prototype limits, and every constructor/composition call and 1x/10x structural pass condition from tasks .1/.3/.5/.6/.8.

The follow-up quality pass restored the two original finite-machine behavioral negatives and exact comments alongside the distinct missing-proof guards. AsyncStart, Cancellation, and SuccessfulCompletion still call `IncrementalPlannerKernel.ofCheckedQuery` directly, while a shared semantic proof in Operations.Planning consumes explicit Query Target equality, completeness, and Lifecycle action evidence. Its named transitive trust print retains only the established kernel and Lifecycle baseline, and the tracked quality audit records the accepted findings and resolutions.

Original full verification remains green: focused 86 jobs, aggregate 254 jobs, model build 329 jobs, physical-TMPDIR regression with tagged Go and 324 Lean jobs, and model lint 260 targets. Follow-up verification passed the Target/operations/facade suite (80 jobs), the documented proof/facade suite (59 jobs), and `make lint-model` (captured rc0, 260 targets). Go lint retained the inherited rc2 with exactly 1,316 sorted diagnostic headers, byte-identical to the prior task-7 signature at SHA-256 `aee7770bec1fe01dab8826427cc89e9ffa7e764fbac25ce6b68bf5f2e3c0b077`. The same-session official review returned SHIP with zero findings at round 2; reviewed tree `96a2c992435323ee07ecb9dae4abc3181fe26a93` matches all expanded task-owned staged blobs. No agent commit was created under the user's standing commit policy; HEAD remains the authorized external WIP `8f40772ebb8e56fac4d068c9990b3c1827f8a05f`. The conductor owns the renewed whole-spec completion review before fn-62 closure.

stage: impl-review - ran (SHIP; round 2; actual model codex:gpt-5.6-sol:medium; receipt timestamp 2026-09-06T06:28:56.201234Z; `/tmp/impl-review-receipt-fn-62-make-ordinary-temporal-model-authoring.7.json`)
stage: plan-sync - skipped(config: planSync.enabled != true)
stage: tracker-sync - skipped(config: tracker bridge inactive)
## Evidence
- Commits:
- Tests: baseline: cd model && mise exec -- lake build Umpire.TargetTests Umpire.Property.Tests Umpire.Behavior.Tests Umpire.Query.Tests Umpire.Observation.Tests Umpire.Planning.Tests Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.OperationsTests Temporal.Feature.Nexus.ObservationTests (pass; 130 jobs; /tmp/fn62-task7-baseline.log), TDD red: cd model && mise exec -- lake build Temporal.Feature.NexusTests (expected task7 specimen compilation failures; /tmp/fn62-task7-tdd-nexustests.log), cd model && mise exec -- lake build Temporal.Feature.NexusTests Umpire.Query.Tests Umpire.Planning.Tests Temporal.Feature.Nexus.ObservationTests Temporal.Feature.Nexus.OperationsTests (pass; 86 jobs; /tmp/fn62-task7-focused-final.log), cd model && mise exec -- lake build UmpireTests TemporalModelTests TemporalExperimentalTests (pass; 254 jobs; /tmp/fn62-task7-final-aggregate.log), make umpire-build-model (pass; 329 jobs; /tmp/fn62-task7-final-umpire-build-model.log), TMPDIR=/private/tmp/fn62-task7-regression.a6A6VH make umpire-check-regression (pass; tagged Go packages, exact inherited live identities, 324 Lean jobs; /tmp/fn62-task7-final-regression.log), make lint-model (original pass; 260 targets and complete import graph; /tmp/fn62-task7-final-lint-model.log), make lint-code GOLANGCI_LINT_FIX=false (original inherited rc2; exactly 1316 sorted diagnostic headers; SHA-256 aee7770bec1fe01dab8826427cc89e9ffa7e764fbac25ce6b68bf5f2e3c0b077; byte-identical task8 baseline; /tmp/fn62-task7-final-lint-code.log), quality: cd model && mise exec -- lake build Umpire.TargetTests Temporal.Feature.Nexus.OperationsTests Temporal.Feature.NexusTests (pass; 80 jobs; restored behavioral/missing-proof tests and named transitive trust; /tmp/fn62-task7-quality-focused.log), quality docs: cd model && mise exec -- lake build Temporal.Feature.Nexus.Operations.Planning Temporal.Feature.NexusTests (pass; 59 jobs; /tmp/fn62-task7-quality-doc-focused.log), quality: make lint-model (pass; persisted rc0; 260 targets and complete import graph; /tmp/fn62-task7-quality-lint-model-rc.log; /tmp/fn62-task7-quality-lint-model.rc), quality: make lint-code GOLANGCI_LINT_FIX=false (inherited rc2; exactly 1316 sorted diagnostic headers; SHA-256 aee7770bec1fe01dab8826427cc89e9ffa7e764fbac25ce6b68bf5f2e3c0b077; byte-identical prior task7 signature; /tmp/fn62-task7-quality-lint-code.log; /tmp/fn62-task7-quality-diagnostic-headers.txt), quality audit and accepted resolutions (.flow/artifacts/fn-62-make-ordinary-temporal-model-authoring/quality-audit.md), public facade import audit (no Internal, Experimental, runtime, or verification imports), transitive named trust audit (#print axioms Lifecycle.targetAuthoring, lifecycleIncrementalKernelResult_isSome, AsyncStart.run, Observation.checkedPlan, authoredRun; established baseline retained; no new production proof trust), structural cost audit (tracked model/Temporal/Feature/Nexus/EVIDENCE.md; all task1/3/5/6/8 calls, traversals, proof-only follow-up, and 1x/10x pass conditions), impl-review codex:gpt-5.6-sol:medium (SHIP; round 2; zero findings; same session 01a07548-4d64-7982-8e77-3b97b3cb089e; /tmp/impl-review-receipt-fn-62-make-ordinary-temporal-model-authoring.7.json), reviewed owned-blob equality (13/13 paths; reviewed tree 96a2c992435323ee07ecb9dae4abc3181fe26a93)
- PRs: