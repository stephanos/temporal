---
satisfies: [R4, R6, R8, R9]
---
# fn-77-typed-operations-parameterized-actions.8 Lower field requirements with checked complete Case coverage

## Description
Lower field requirements with checked complete Case coverage for the referenced parent requirements.

**Size:** M
**Files:** model/Umpire/Case/**; model/Umpire/Case.lean; model/Umpire/Observation/Projection/**; model/Umpire/Property/Scoped/**; model/Temporal/System/Nexus/ImplementationLink.lean
**Touches:** [model/Umpire/Case/**, model/Umpire/Case.lean, model/Umpire/Observation/Projection/**, model/Umpire/Property/Scoped/**, model/Temporal/System/Nexus/ImplementationLink.lean]

### Approach
- Introduce checked input-field construction, result/event Observation and clause-lowering coverage linked to operation/schema/source identities; validate complete selected coverage before compiling any Case.
- Lower typed request assignments and same-step/captured operands using task7 capability; prove emitted executable field denotation against task5/6 source meaning and existing fn78 lowering.
- Separate projection/correlation correspondence conditions from independent product clauses. Missing support/correlation fails Link admission; field mismatches fail only their authored requirement.
- Reject any missing/unsupported requested mapping as a whole-Case source-owned error. Add removal/mutation controls and field-specific online/offline/model equivalence fixtures.

### Investigation targets
**Required:**
- model/Umpire/Case/Compiler.lean:42 — lowering diagnostics; line93 whole-Case assembly.
- model/Umpire/Case/Scoped.lean:69 — checked decoded-meaning Lowered evidence.
- model/Umpire/Case/ScopedProofs.lean — actual-history source correspondence.
- model/Umpire/Observation/Projection/Declaration.lean:33 — submission versus confirmed evidence.

### Quick commands
`cd model && mise exec -- lake build Umpire.Case.CompilerTests Umpire.Property.Tests.Scoped.Evidence Temporal.Feature.Nexus3.Tests`
`go test -tags test_dep ./common/testing/testpilot/internal/verification ./common/testing/testpilot`
`make umpire-check-case-runtime-conformance`

`cd model && mise exec -- lake build Umpire.Case.Tests.FieldLowering`

### Execution constraints
Read the full parent spec. Preserve existing comments and unrelated dirty source. New paths listed here are proposed owners; reuse an established equivalent before creating one. Run Lean jobs serially; fn76 lint recovery is complete. Preserve fn75 semantic import isolation and fn78 obligation/evidence authority. No operation cancellation, new dependency/toolchain, or implicit fixture promotion. Do not stage, commit, or push; the user owns commits. Capture task-local before/after trust for changed load-bearing declarations; task11 additionally compares to the original task1 substrate.

Create and wire the proposed test module Umpire.Case.Tests.FieldLowering into its existing aggregate test root; if an existing equivalent is reused, record its exact name and run it explicitly. The new-module Quick command applies after creation; baseline existing roots before editing instead of treating an absent module as an environmental failure.

## Acceptance
- [ ] Every selected input/result/event field and requested clause has inspected checked coverage; removal/unsupported mappings reject before Driver I/O.
- [ ] Actual emitted expressions/requests/results/captures have source-denotation correspondence, not only serialized round-trip evidence.
- [ ] Correlation corruption reports Link failure; authored field mutation yields its clause violation or supported admission rejection, never missing-evidence success.
- [ ] Model/online/offline field fixtures agree and old scoped/Nexus3 fixture bytes remain unchanged.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
