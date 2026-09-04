---
satisfies: [R1, R2, R8]
---
# fn-62-make-ordinary-temporal-model-authoring.1 Deepen finite Target authoring and migrate Nexus lifecycle

## Description
Deepen the existing `FiniteMachine` adapter for R1, R2, and R8, then migrate the ordinary Nexus lifecycle Target through it. Keep every AUT-08 semantic input explicit while eliminating repeated assembly around the existing checked Target boundary.

**Size:** M
**Files:** `model/Umpire/Target/FiniteMachine.lean`, `model/Umpire/Target/Tests/FiniteMachine.lean`, `model/Umpire/Target/ImportTests.lean`, `model/Temporal/Feature/Nexus/Lifecycle/Target.lean`, `model/Temporal/Feature/Nexus/Lifecycle/TargetTests.lean`
**Touches:** [model/Umpire/Target/FiniteMachine.lean, model/Umpire/Target/Tests/FiniteMachine.lean, model/Umpire/Target/ImportTests.lean, model/Temporal/Feature/Nexus/Lifecycle/Target.lean, model/Temporal/Feature/Nexus/Lifecycle/TargetTests.lean]

### Approach
- Preserve `FiniteMachine` as the sole ordinary complete-finite adapter and `TransitionKernel` as the expert alternative; follow the authority split in `model/Umpire/Target/FiniteMachine.lean:7-110`.
- Group explicit ordered domains, encoders, enumerators, coverage, and Action-executability evidence into the narrowest semantic substructures or constructors; derive record glue only, never a required AUT-08 input.
- Reuse `TargetDefinition`, `TargetComposition`, `AuthoredTarget.make`, and `checkTarget` from `model/Umpire/Target/Language.lean:157-234`; do not add a parallel Target representation or checker.
- Migrate the repeated descriptor/provider/metadata assembly in `model/Temporal/Feature/Nexus/Lifecycle/Target.lean:198-403` and retain all existing comments.
- Compare checked Target identity, canonical metadata, Behavior Fingerprint, transitions, capability graph, and completeness before and after migration.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Target/FiniteMachine.lean:7-110` — ordinary finite adapter and required proofs.
- `model/Umpire/Target/Language.lean:157-234` — explicit composition and checked Target assembly.
- `model/Temporal/Feature/Nexus/Lifecycle/Target.lean:198-403` — repeated ordinary authoring surface to reduce.
- `model/Umpire/Target/Tests/FiniteMachine.lean` — generic positive and invalid finite-machine patterns.
- `model/Temporal/Feature/Nexus/Lifecycle/TargetTests.lean` — migrated Target regressions.

**Optional** (reference as needed):
- `.flow/specs/fn-41-finitemachine-target-authoring.md:86-119` — accepted ordinary/expert boundary.
- `.flow/specs/fn-51-shorten-ordinary-model-authoring.md:68-85` — prior simplification and intentional residual literals.

### Key context
AUT-08 requires authors to supply the semantic domains, encoders, enumerators, and proof evidence. The helper may organize and reuse those inputs, not infer or omit them.

### Acceptance
- [ ] Nexus lifecycle Target is authored from the deepened finite adapter without direct ordinary construction of expert kernel/completeness internals.
- [ ] Every state, Action, Model Outcome, encoder, enumerator, coverage/executability proof, provider, connector, and metadata choice remains explicit.
- [ ] Generic negative tests cover incomplete/out-of-domain enumeration, non-executable Actions, and missing/conflicting capability composition with existing typed diagnostics.
- [ ] Existing lifecycle IDs, canonical metadata, Behavior Fingerprint, transition relation, completeness, imports, and comments are preserved.
- [ ] Focused Target and Nexus lifecycle Lake targets pass with `-tags test_dep` where applicable.

## Acceptance
- [ ] R1, R2, and R8 are satisfied for finite Target authoring.
- [ ] `cd model && mise exec -- lake build Umpire.TargetTests Umpire.Target.ImportTests Temporal.Feature.Nexus.LifecycleTests` passes.
- [ ] No new axiom, `sorry`, `admit`, warning, dependency, or public authoring language is introduced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
