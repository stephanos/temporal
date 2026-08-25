---
satisfies: [R1, R5, R8]
---
# fn-3-umpire-semantic-authoring-and-planning.1 Establish semantic vocabulary and capability composition

## Description
Build the shared semantic substrate for R1/R5: stable typed identities and kinds, canonical metadata, pure trace values, proof-backed capabilities, target composition, explicit connectors, and structured declaration diagnostics. Keep this as a deep module so Property, Behavior, Query, and downstream evidence work consume one narrow interface.

**Size:** M
**Files:** `model/Temporal/Experiment/Semantics.lean`, `model/Temporal/Experiment/SemanticsTests.lean`
**Touches:** [model/Temporal/Experiment/Semantics.lean, model/Temporal/Experiment/SemanticsTests.lean]

### Approach
- Add one semantic-foundation module that owns namespaced declaration identities, declaration kinds, source/provenance metadata, typed bound units, canonical ordering/digest inputs, and the minimal pure `SemanticTrace` shape.
- Model capabilities and connectors as explicit checked records with stable identity/version and Lean proof fields for required laws; portable projections retain law identities and semantic digests rather than proof terms.
- Define the pure target transition-kernel contract: finite resolved-setup/initial-state enumeration and state/action step enumeration returning only valid outcome/result-state/model-observation tuples, with soundness and completeness proofs against the target's authoritative relation.
- Compose targets from the transition kernel, declared providers, and explicit connectors, rejecting incomplete kernels and ambiguity independently of declaration/type-class order.
- Replace prose-only compiler errors in the new path with stable kinds plus declaration identity, source path, offending value, and canonically ordered related identities.
- Give vocabulary/capability/target metadata a deterministic canonical JSON projection and semantic digest; readers remain out of scope.
- Provide typed Lean constructors/combinators and lightweight notation only; do not introduce a custom parser/elaborator.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Experiment/DSL.lean:5-138` — legacy identities, bounds, target, and error records being superseded
- `model/Temporal/Experiment/Compiler.lean:6-40` — existing deterministic identity validation pattern
- `.plans/UMPIRE_DSL.md:151-242` — vocabulary and capability-composition contract
- `model/NexusAutoClose.lean:508-584` — checked model configuration/transition concepts the target adapter must expose later

**Optional** (reference as needed):
- `model/Temporal/ExperimentTests.lean:116-368` — current negative and canonical-order fixtures

### Key context
- `SemanticTrace` contains model-emitted semantic observations only; qualification, raw evidence, and derivations belong to fn-4.
- Preserve existing comments in every modified file.

### Quick commands
```bash
cd model && mise exec -- lake env lean Temporal/Experiment/SemanticsTests.lean
```
## Acceptance
- [ ] Stable identities reject empty, duplicate, unknown, and wrong-kind declarations with all structured diagnostic fields populated.
- [ ] Capability composition accepts proof-backed compatible providers, rejects missing laws/providers, and rejects conflicting providers unless one explicit connector reconciles them.
- [ ] A checked target exposes sound-and-complete finite initial-state and semantic-step enumerators; absent proofs or a step outside the authoritative relation reject composition.
- [ ] Provider and declaration input order do not change checked composition, digests, diagnostics, or rendered values.
- [ ] Metadata projections are byte-identical on repetition and change when a meaning-bearing identity, capability contract, law, connector, or target-kernel digest changes.
- [ ] The exported pure trace retains initial state plus selected actions, model outcomes, resulting states, and model-emitted semantic observations without evidence qualification fields.
- [ ] Minimal Workflow/Nexus-shaped and independent switch-shaped target kernels compile against the same semantic interface.
- [ ] The focused Lean test command passes and the R8 exclusion audit is clean.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
