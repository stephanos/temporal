---
satisfies: [R1, R4, R7]
---
# fn-19-bounded-local-temporal-execution-and.1 Define the model-owned ephemeral local execution profile

## Description
Prepare R1/R4/R7's Lean-owned portable profile values without putting authority material into artifacts.

**Size:** M
**Files:** `model/Temporal/System/Execution.lean`, `model/Temporal/System/Execution/LocalProfile.lean`, `model/Temporal/System/Execution/LocalProfileTests.lean`, `model/Temporal/System.lean`, `model/TemporalModelTests.lean`
**Touches:** [model/Temporal/System/Execution.lean, model/Temporal/System/Execution/LocalProfile.lean, model/Temporal/System/Execution/LocalProfileTests.lean, model/Temporal/System.lean, model/TemporalModelTests.lean]

### Approach
- Define the exact ephemeral-local profile SemanticReference, non-self-referential profile digest projection, generic required capabilities, five canonical phase budgets, seed/attempt policy, and closed participant/program requirements as inert Lean values.
- Validate the fixed 120-second/4096-record/16-MiB aggregate and fn-18 RuntimeConfiguration invariants without adding endpoints, namespaces, credentials, executable names, or callbacks.
- Expose the narrow execution facade through the existing `Temporal.System` import and canonical fixture helpers consumed by the later Nexus composition.
- Pin reordering/drift/limit/authority-field negative cases and preserve existing comments.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Artifact/Runtime.lean` after fn-18
- `model/Temporal/System/Configuration/Core.lean` checked-value patterns
- parent spec exact input/authority and phase-budget contracts

### Acceptance
- [ ] Profile identity/digest/capabilities and all five budgets have exact canonical bytes and pass fn-18 validation.
- [ ] Altered profile fields, capability union, time/attempt/record/byte totals, or authority-like data reject before a checked value exists.
- [ ] The module imports no Nexus feature and defines no runtime IO.
- [ ] `Temporal.System` and aggregate test imports expose/check the profile without importing a feature adapter.

## Acceptance
- [ ] R1/R4 portable local profile values are exact and authority-free.
- [ ] Focused Lean profile tests and `TemporalModelTests` pass.
- [ ] Reusable Umpire artifact modules remain Temporal-independent.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
