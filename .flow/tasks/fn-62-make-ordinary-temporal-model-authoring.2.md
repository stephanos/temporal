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
Migrated the established Nexus Lifecycle to `Temporal.Shared.definitionFamily` and the proof-carrying `FiniteMachine.authoredTarget` seam while preserving exact IDs, sources, metadata, canonical JSON, fingerprint, ordered domains, relations, provider selection, checked admission, and public compatibility aliases. Independent old-shape assembly and actual missing/conflicting-provider fixtures prove exact compatibility; only the downstream planning proof script changed to unfold the new constructor.

Baseline and final Lifecycle plus Operations builds passed. `make lint-model` passed 258 targets. `make lint-code GOLANGCI_LINT_FIX=false` retained the inherited exit 2 with exactly 1,316 normalized diagnostic headers, byte-identical to the approved baseline and SHA-256 `aee7770bec1fe01dab8826427cc89e9ffa7e764fbac25ce6b68bf5f2e3c0b077`. The historical Lifecycle native proof dependencies remain visible and unchanged; no additional native/compiler/custom trust was introduced.

No commit was created under the user's standing commit policy; HEAD remains `7774fdc7ac751ac959816c9829516ce54af57194`. The official staged-overlay review returned SHIP with zero findings; reviewed tree `7825f4540f908466144b9436c4ee637ab525d6cc` equals the final staged tree and all frozen owned source hashes match.

stage: impl-review - ran (codex:gpt-5.6-sol:medium; SHIP; receipt timestamp 2026-09-06T03:18:42.136928Z)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: baseline: cd model && mise exec -- lake build Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.OperationsTests, cd model && mise exec -- lake build Temporal.Feature.Nexus.LifecycleTests, cd model && mise exec -- lake build Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.OperationsTests, make lint-model, make lint-code GOLANGCI_LINT_FIX=false (inherited exit 2; 1316 normalized headers; SHA-256 aee7770bec1fe01dab8826427cc89e9ffa7e764fbac25ce6b68bf5f2e3c0b077; byte-identical to approved baseline), impl-review codex:gpt-5.6-sol:medium SHIP; receipt /tmp/impl-review-receipt-fn-62-make-ordinary-temporal-model-authoring.2.json, reviewed staged tree 7825f4540f908466144b9436c4ee637ab525d6cc equals final staged tree; frozen owned source hashes match
- PRs: