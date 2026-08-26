---
satisfies: [R1, R5, R7]
---
# fn-19-bounded-local-temporal-execution-and.5 Compose Nexus System execution programs and configuration

## Description
### Umpire4 reconciliation (normative)

Move all Nexus execution/program/configuration ownership and public facades from `Temporal.Feature` to `Temporal.System`. Feature retains product-visible semantics only. Compose the complete current `ExperimentSpec`; do not reconstruct participant programs or other omitted meaning from legacy v1.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Complete R1/R5's model-owned Nexus-specific binding and canonical two-member input set for the one live program.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/Execution.lean`, `model/Temporal/Feature/Nexus/ExecutionTests.lean`, `model/Temporal/Feature/Nexus.lean`, `model/Temporal/Feature.lean`, `model/TemporalModelTests.lean`, `tools/umpire/temporal/nexus/testdata/caller-closure-input-set/**`
**Touches:** [model/Temporal/Feature/Nexus/Execution.lean, model/Temporal/Feature/Nexus/ExecutionTests.lean, model/Temporal/Feature/Nexus.lean, model/Temporal/Feature.lean, model/TemporalModelTests.lean, tools/umpire/temporal/nexus/testdata/caller-closure-input-set/**]

### Approach
- Compose the exact local profile, fn-4 evidence/profile/program/mapping references, one participant protocol descriptor, exact capabilities, and fixed budgets into a canonical RuntimeConfiguration for the existing caller-closure ExperimentSpec.
- Define checked inert participant-program metadata for the four phase commands and exact target/action/occurrence; add no callbacks or new persisted family.
- Emit/check in the canonical RuntimeConfiguration plus exact fn-18 manifest/bindings for the two-member input set.
- Prove target/action/occurrence/fault/participant/protocol/capability/program/ref changes fail composition or set admission.
- Wire the Nexus sub-facade through the existing `Temporal.Feature` facade and aggregate tests while keeping reusable Umpire modules domain-neutral.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Feature/Nexus/CallerClosure.lean:462-525`
- fn-4 Temporal evidence profile/program/mapping values
- Task `.1` profile and fn-18 Runtime/Set encoders
- canonical caller-closure ExperimentSpec fixture

### Acceptance
- [ ] The two-member input set is canonical, strictly admitted, and binds the exact current caller-closure artifact.
- [ ] Program/config identities change on every meaning-bearing mutation and remain insensitive only to declared provenance exclusions.
- [ ] Any unsupported fault, extra participant/action/target, or semantic-reference drift rejects before Go execution.
- [ ] Public Temporal facades and focused Lean tests pass.
## Acceptance
- [ ] R1/R5 exact Nexus configuration/program binding is model-owned and inspectable.
- [ ] Cross-language fixture bytes pass fn-18 strict set admission.
- [ ] No new semantic or artifact authority is introduced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
