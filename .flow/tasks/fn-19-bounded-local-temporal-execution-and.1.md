---
satisfies: [R1, R4, R7]
---
# fn-19-bounded-local-temporal-execution-and.1 Define the model-owned ephemeral local execution profile

## Description
### Umpire4 reconciliation (normative)

All model-owned execution profiles, participant programs, configuration interpretation, and evidence-source contracts live under `Temporal.System`. This task must keep Feature imports absent and state whether each adapter uses the in-memory Temporal test suite or a developer-server lifecycle, including cleanup and evidence limitations.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Prepare R1/R4/R7's Lean-owned portable profile values without putting authority material into artifacts.

**Size:** M
**Files:** `model/Temporal/System/Execution.lean`, `model/Temporal/System/Execution/LocalProfile.lean`, `model/Temporal/System/Execution/LocalProfileTests.lean`, `model/Temporal/System.lean`, `model/TemporalModelTests.lean`
**Touches:** [model/Temporal/System/Execution.lean, model/Temporal/System/Execution/LocalProfile.lean, model/Temporal/System/Execution/LocalProfileTests.lean, model/Temporal/System.lean, model/TemporalModelTests.lean]

### Approach
- Define the exact ephemeral-local profile SemanticReference, non-self-referential profile digest Generated View, generic required capabilities, five canonical phase budgets, seed/attempt policy, and closed participant/program requirements as inert Lean values.
- Validate the fixed 120-second/4096-record/16-MiB aggregate and fn-18 RuntimeConfiguration invariants without adding endpoints, namespaces, credentials, executable names, or callbacks.
- Expose the narrow execution facade through the existing `Temporal.System` import and canonical fixture helpers consumed by the later Nexus composition.
- Pin reordering/drift/limit/authority-field negative cases and preserve existing comments.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Artifact/Runtime.lean` after fn-18
- `model/Temporal/System/Configuration/Core.lean` checked-value patterns
- parent spec exact input/authority and phase-budget contracts

### Acceptance
- [ ] Profile Definition ID/Behavior Fingerprint/capabilities and all five budgets have exact canonical bytes and pass fn-18 validation.
- [ ] Altered profile fields, capability union, time/attempt/record/byte totals, or authority-like data reject before a checked value exists.
- [ ] The module imports no Nexus feature and defines no runtime IO.
- [ ] `Temporal.System` and aggregate test imports expose/check the profile without importing a feature adapter.
## Acceptance
- [ ] R1/R4 portable local profile values are exact and authority-free.
- [ ] Focused Lean profile tests and `TemporalModelTests` pass.
- [ ] Reusable Umpire artifact modules remain Temporal-independent.

## Done summary
Defined the sealed, authority-free ephemeral-local execution profile under Temporal.System with exact pretty Generated View identity, canonical capabilities/budgets/policies, closed participant requirements, fn-18 RuntimeConfiguration checks, mutation coverage, and aggregate exposure.

baseline: red (parent final-spec Quick targets assigned to later fn19 tasks were absent before fn19.1; the pre-existing temporaltest suite passed)
stage: impl-review - ran (SHIP at 2026-08-29T11:46:28.933655Z)
## Evidence
- Commits: 0a43d66f181edea90ed504f17a543d26d87056c3, 741fccfd7bd8e9264d0fbcd4a3c1459ed227a131
- Tests: baseline: red (go test -count=1 ./tools/umpire/runtime/...; go test -count=1 ./tools/umpire/temporal/local/...; go test -count=1 ./tools/umpire/temporal/nexus/...; go test -count=1 ./tools/umpire/cmd/umpire-local-run/...; LocalProfileTests and Nexus ExecutionTests targets absent before implementation/later tasks), go test -count=1 ./temporaltest/..., cd model && mise exec -- lake build Temporal.System.Execution.LocalProfileTests TemporalModelTests, mise exec -- make lint-model, git diff --check fd5075998e7b58f6e0be0815a873af2219caea5e..HEAD plus Temporal.System.Execution Feature-import/runtime-IO guards, flowctl codex impl-review fn-19-bounded-local-temporal-execution-and.1 --base fd5075998e7b58f6e0be0815a873af2219caea5e (SHIP)
- PRs: