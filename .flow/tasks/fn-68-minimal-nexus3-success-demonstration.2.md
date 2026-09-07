---
satisfies: [R3]
---
# fn-68-minimal-nexus3-success-demonstration.2 Lower the checked success witness into a Case

## Description
Implement the single checked lowering in R3. Keep Nexus-specific execution/evidence wiring in the requested Nexus3 directory.

**Size:** M
**Files:** model/Temporal/Feature/Nexus3/Testpilot.lean (new feature adapter), model/Temporal/Feature/Nexus3/Tests.lean, model/Temporal/Testpilot/CaseSupport.lean (new, only reusable construction mechanics used by multiple producers), model/Temporal/Testpilot/GetSystemInfo.lean and Conformance.lean (split generic owners as needed), model/Temporal/Testpilot.lean (thin generic facade), model/Temporal/TestpilotTests.lean, model/Temporal/Tool/Testpilot.lean.
**Touches:** [model/Temporal/Feature/Nexus3/Testpilot.lean, model/Temporal/Feature/Nexus3/Tests.lean, model/Temporal/Testpilot/*.lean, model/Temporal/Testpilot.lean, model/Temporal/TestpilotTests.lean, model/Temporal/Tool/Testpilot.lean]

### Approach
- Expose a narrow Case-production result from `Temporal.Feature.Nexus3.Testpilot`. Consume the checked Query, its referenced checked Target/Property/Behavior and its selected witness; validate the exact supported structure before constructing Compiler.Input. IDs/fingerprints identify checked inputs but are not substitutes for checking semantic fields.
- Extract from `Temporal.Testpilot` only Case-building mechanics used by multiple producers into `Temporal.Testpilot.CaseSupport`. Keep Nexus history paths, correlation rules, Property binding, Program, and monitor construction in the Nexus3 adapter. Split generic get-system-info and conformance producers into their own submodules when needed so `Temporal/Testpilot.lean` becomes an import-only or equivalently thin generic facade. Generic Testpilot modules must not import Nexus3.
- Derive Case definition bindings from checked declarations and construct the success monitor from the supported Property and declared history mapping. Keep one scheduled capture followed by correlated started/completed evidence. Do not inherit the old independently authored liveness requirement or its violation deadline as the meaning of the final-success Property.
- Keep execution timeouts bounded separately. Absence of correlated terminal evidence remains inconclusive; no new runtime horizon unit, evaluator, or opcode. Reuse the closed Case compiler only as the final assembler.
- Convert the checked Query's two authored Known Gaps exactly once with `KnownGapSet.toCaseKnownGaps` and pass them to `Compiler.Input.knownGaps`; assert exact Case metadata and prove the gaps do not alter or waive the monitored success Property.
- Remove `Temporal.Testpilot.asyncNexusCase`; update `Temporal.Tool.Testpilot` and model tests to call the Nexus3 Producer directly while preserving the existing `async-nexus` selector. Keep imports `Nexus3.Testpilot` → `Nexus3.Nexus`, never the reverse, and do not relax ModelLint.
- Reject raw/unmatched inputs and any unsupported Property/Query/Action shape before emitting a Case. Cancellation and scoped progress remain design documentation, not additional implemented declarations.

### Investigation targets
**Required:**
- model/Temporal/Feature/Nexus3/Integration.md:36-106 — action/evidence and unsupported-lowering design
- model/Temporal/Testpilot.lean:144-405 — existing private async Program/monitor and Case entry point
- model/Umpire/Case/Compiler.lean:4-95 — assembler input and typed rejection
- model/Umpire/Case/Contract.lean:64-90 — closed monitor and horizon vocabulary
- model/Temporal/TestpilotTests.lean:27-39 — retained async Case structural assertions
**Optional:**
- model/Temporal/Tool/Testpilot.lean:9-12 — renderer entry point and direct feature-adapter call
- .flow/memory/bug/integration/keep-raw-semantics-behind-checked-input-2026-09-05.md — checked-only facade lesson

### Quick commands
`cd model && mise exec -- lake build Temporal.Feature.Nexus3.Tests temporal-testpilot`
`cd model && mise exec -- lake build TemporalModelTests`

## Acceptance
- [ ] Default checked completion produces one Case with Nexus3 source IDs/fingerprints and the exact checked cancellation/progress Known Gaps from `Temporal.Feature.Nexus3.Testpilot`; no independent or aliased async Nexus Case remains in `Temporal.Testpilot`.
- [ ] Same-ID semantic mutations (remove/change the success row, change the Property expectation) change the generated meaning or return a typed failure with no Case. Wrong target, missing/mismatched witness, unsupported clause/action/query form, and altered sequence are rejected.
- [ ] Generated monitor expectations come from the checked Property through the explicit evidence mapping; execution deadlines add no semantic-step/time conversion or hidden stronger liveness requirement.
- [ ] Generic Testpilot support imports no Nexus3 module; Nexus-specific Property, history correlation, Program, and monitor definitions live beside Nexus3; `Temporal/Testpilot.lean` is a thin generic facade and existing generic/conformance Case behavior is unchanged.
- [ ] Focused Lean tests and existing TestpilotTests pass; other existing Case producers retain their semantics.

## Done summary
Implemented checked Nexus3 completion lowering in `Temporal.Feature.Nexus3.Testpilot`: the Producer validates the exact checked Target, Property, Behavior, Query, and optional selected witness, binds their IDs and fingerprints, converts the two authored Known Gaps once, and emits the existing async Program with a Property-derived, no-horizon correlated success monitor. Generic Case mechanics, get-system-info, and six conformance Producers now have separate Nexus-free modules; the top-level facade is import-only, the renderer calls Nexus3 directly, and `Temporal.Testpilot.asyncNexusCase` is removed.

Focused tests cover checked provenance and Known Gaps, same-ID Property mutation, Behavior/action mutation, unsupported Query form, wrong Target, absent/mismatched witness, exact correlation requirements, and the lack of a semantic time horizon. Get-system-info and all six conformance renderings remain byte-identical to their checked-in fixtures.

Baseline: green (`cd model && mise exec -- lake build Temporal.Feature.Nexus3.Tests temporal-testpilot`; `cd model && mise exec -- lake build TemporalModelTests`). TDD RED: `/tmp/fn68-task2-red.log` records the missing Nexus3 Testpilot module before production edits.

stage: impl-review - SHIP(receipt: /tmp/impl-review-receipt-fn68-task2.json)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: cd model && mise exec -- lake build Temporal.Feature.Nexus3.Tests temporal-testpilot, cd model && mise exec -- lake build Temporal.TestpilotTests, cd model && mise exec -- lake build TemporalModelTests, make lint-model, model/.lake/build/bin/temporal-testpilot async-nexus > /tmp/fn68-task2-async-nexus.json, cmp rendered get-system-info and all six conformance Cases with checked-in fixtures, git diff --check, trust audit: no axiom/unsafe/#eval/sorry/admit in task files, import audit: generic Temporal/Testpilot modules contain no Nexus3 imports, alias audit: no Temporal.Testpilot.asyncNexusCase remains
- PRs: