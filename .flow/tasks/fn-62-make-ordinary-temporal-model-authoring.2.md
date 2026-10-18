---
satisfies: [R1, R2, R8]
---
# fn-62-make-ordinary-temporal-model-authoring.2 Migrate established Nexus Lifecycle with exact compatibility

## Description
Migrate the established Lifecycle to `.1`'s proof-carrying interface and `.3`'s identities. This repurposes the never-executed planner task; R3 was completed by fn-65 and needs no new adapter.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/Lifecycle/Target.lean`, `model/Temporal/Feature/Nexus/Lifecycle/TargetTests.lean`, `model/Temporal/Feature/Nexus/Lifecycle/SemanticsTests.lean`
**Touches:** [model/Temporal/Feature/Nexus/Lifecycle/Target.lean, model/Temporal/Feature/Nexus/Lifecycle/TargetTests.lean, model/Temporal/Feature/Nexus/Lifecycle/SemanticsTests.lean]

## Approach
- Capture existing exact Target JSON, metadata, fingerprint, domains, initial states and transitions before migration. Use established assertions as independent oracles.
- Replace only repeated assembly; authors retain all semantic choices and proof arguments. Do not route the established model through `ValidatedFiniteTable.machine`.
- Keep public declaration names, sources and explicit composition stable. If source correction is unavoidable, name and test its exact permitted delta separately.
- Reuse the existing planning adapter when checking finite completeness; no changes to `IncrementalPlannerKernel.ofCheckedQuery` are required.

## Investigation targets
**Required:**
- `model/Temporal/Feature/Nexus/Lifecycle/Target.lean` — migration owner.
- `model/Temporal/Feature/Nexus/Lifecycle/TargetTests.lean:16` — exact metadata and identity fixtures.
- `model/Temporal/Feature/Nexus/Lifecycle/SemanticsTests.lean` — independent relation checks.
- `model/Umpire/Target/FiniteMachine.lean` — new author evidence seam.

## Acceptance
- [ ] Established Target uses smaller assembly with explicit author evidence and unchanged checkTarget admission.
- [ ] The ordinary Lifecycle authoring expression calls FiniteMachine.authoredTarget directly with explicit metadata/composition; it contains no hand-built TargetDefinition, dependent planning transport, or repeated machine setup/kernel selections. Existing public targetDefinition/finitePlanning aliases may forward to owner APIs for compatibility but are not part of the new author journey.
- [ ] Exact IDs, metadata, canonical JSON, fingerprints, domains, initial/transition relations, provider choices, and public imports match baseline.
- [ ] Missing capability and competing-provider failures retain complete diagnostics; axiom inventory has no new compiler trust.
- [ ] `cd model && mise exec -- lake build Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.OperationsTests` passes; applicable lint gates pass or match verified inherited output.

## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: