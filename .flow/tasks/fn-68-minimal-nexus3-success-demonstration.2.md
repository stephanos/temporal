---
satisfies: [R3]
---
# fn-68-minimal-nexus3-success-demonstration.2 Lower the checked success witness into a Case

## Description
Implement the single checked lowering in R3. Keep Nexus-specific execution/evidence wiring in the requested Nexus3 directory.

**Size:** M
**Files:** model/Temporal/Feature/Nexus3/Integration.lean (new), model/Temporal/Feature/Nexus3/CaseSupport.lean (only if needed to share existing physical construction helpers), model/Temporal/Feature/Nexus3/Tests.lean, model/Temporal/CaseRuntime.lean.
**Touches:** [model/Temporal/Feature/Nexus3/Integration.lean, model/Temporal/Feature/Nexus3/CaseSupport.lean, model/Temporal/Feature/Nexus3/Tests.lean, model/Temporal/CaseRuntime.lean]

### Approach
- Expose a narrow Case-production result from Integration.lean. Consume the checked Query, its referenced checked Target/Property/Behavior and its selected witness; validate the exact supported structure before constructing Compiler.Input. IDs/fingerprints identify checked inputs but are not substitutes for checking semantic fields.
- Move/reuse the existing async Program's physical SDK activation/completion/history mechanics from Temporal.CaseRuntime. If shared helpers are needed by its other examples, put only those helpers in Nexus3/CaseSupport.lean. Preserve get-system-info and six conformance behaviors; do not copy an independently maintained finished async Case.
- Derive Case definition bindings from checked declarations and construct the success monitor from the supported Property and declared history mapping. Keep one scheduled capture followed by correlated started/completed evidence. Do not inherit the old independently authored liveness requirement or its violation deadline as the meaning of the final-success Property.
- Keep execution timeouts bounded separately. Absence of correlated terminal evidence remains inconclusive; no new runtime horizon unit, evaluator, or opcode. Reuse the closed Case compiler only as the final assembler.
- Make Temporal.CaseRuntime.asyncNexusCase delegate to the Nexus3 producer; preserve the existing Tool/CaseRuntime async-nexus selector. Keep imports Integration -> Nexus, never Nexus -> Integration, and do not relax ModelLint. The requested co-location is limited to this prototype integration module.
- Reject raw/unmatched inputs and any unsupported Property/Query/Action shape before emitting a Case. Cancellation and scoped progress remain design documentation, not additional implemented declarations.

### Investigation targets
**Required:**
- model/Temporal/Feature/Nexus3/Integration.md:36-106 — action/evidence and unsupported-lowering design
- model/Temporal/CaseRuntime.lean:144-405 — existing private async Program/monitor and Case entry point
- model/Umpire/Case/Compiler.lean:4-95 — assembler input and typed rejection
- model/Umpire/Case/Contract.lean:64-90 — closed monitor and horizon vocabulary
- model/Temporal/CaseRuntimeTests.lean:27-39 — retained async Case structural assertions
**Optional:**
- model/Temporal/Tool/CaseRuntime.lean:9-12 — renderer entry point
- .flow/memory/bug/integration/keep-raw-semantics-behind-checked-input-2026-09-05.md — checked-only facade lesson

### Quick commands
`cd model && mise exec -- lake build Temporal.Feature.Nexus3.Tests temporal-case-runtime`
`cd model && mise exec -- lake build TemporalModelTests`

## Acceptance
- [ ] Default checked completion produces one Case with Nexus3 source IDs/fingerprints; Temporal.CaseRuntime.asyncNexusCase equals that producer result rather than an independent relabeled constant.
- [ ] Same-ID semantic mutations (remove/change the success row, change the Property expectation) change the generated meaning or return a typed failure with no Case. Wrong target, missing/mismatched witness, unsupported clause/action/query form, and altered sequence are rejected.
- [ ] Generated monitor expectations come from the checked Property through the explicit evidence mapping; execution deadlines add no semantic-step/time conversion or hidden stronger liveness requirement.
- [ ] Focused Lean tests and existing CaseRuntimeTests pass; other existing Case producers retain their semantics.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
