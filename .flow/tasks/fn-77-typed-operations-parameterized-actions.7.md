---
satisfies: [R2, R6, R8, R9]
---
# fn-77-typed-operations-parameterized-actions.7 Support exact portable field and capture evaluation

## Description
Support exact portable field and capture evaluation for the referenced parent requirements.

**Size:** M
**Files:** model/Testpilot/**; proto/internal/temporal/server/api/testpilot/v1/**; api/testpilot/v1/**; common/testing/testpilot/internal/ir/**; common/testing/testpilot/internal/verification/**; common/testing/testpilot/internal/execution/**; model/Umpire/Case/Tests/**
**Touches:** [model/Testpilot/**, proto/internal/temporal/server/api/testpilot/v1/**, api/testpilot/v1/**, common/testing/testpilot/internal/ir/**, common/testing/testpilot/internal/verification/**, common/testing/testpilot/internal/execution/**, model/Umpire/Case/Tests/**]

### Approach
- Map the required task3/5/6 fragment onto existing portable values, Program/Contract paths, expressions and captures; inventory demonstrably missing forms before adding any versioned generic capability.
- Implement only necessary generic schema/value/capture support in existing codec/admission/evaluator owners, preserving disjoint Program/Contract contexts and declared Observation-only Contract access.
- Range-check Lean numbers and schema identities before encoding, reject unknown/stale capability fields and enforce typed projected-value/capture work. Preserve existing Case1.0 and default-empty meaning where unchanged.
- Add independently derived Lean/Go fixtures for exact bytes, optional/default, oneof, nested/repeated/maps, wrong schemas, malformed/incomplete evidence and bounded failures; demonstrate online/offline parity. Regenerate complete protocol surfaces if annotated/proto inputs change.

### Investigation targets
**Required:**
- model/Testpilot/Authoring.lean:35 — existing portable concrete value constructors.
- model/Testpilot/Protocol.lean — generated protocol authority.
- common/testing/testpilot/internal/ir/expression.go:78 — typed expression binding.
- common/testing/testpilot/internal/ir/runtime_value.go — codec validation.
- common/testing/testpilot/internal/verification/scoped.go — fn78 run-local scoped evaluation.

### Quick commands
`go test -tags test_dep ./common/testing/testpilot/internal/ir ./common/testing/testpilot/internal/verification ./common/testing/testpilot/internal/execution`
`cd model && mise exec -- lake build Testpilot.Tests Umpire.Case.CompilerTests`
`make umpire-check-testpilot-protocol umpire-check-testpilot-authoring umpire-check-case-runtime-conformance`

`cd model && mise exec -- lake build Testpilot.Tests.Fields`

### Execution constraints
Read the full parent spec. Preserve existing comments and unrelated dirty source. New paths listed here are proposed owners; reuse an established equivalent before creating one. Run Lean jobs serially; fn76 lint recovery is complete. Preserve fn75 semantic import isolation and fn78 obligation/evidence authority. No operation cancellation, new dependency/toolchain, or implicit fixture promotion. Do not stage, commit, or push; the user owns commits. Capture task-local before/after trust for changed load-bearing declarations; task11 additionally compares to the original task1 substrate.

Create and wire the proposed test module Testpilot.Tests.Fields into its existing aggregate test root; if an existing equivalent is reused, record its exact name and run it explicitly. The new-module Quick command applies after creation; baseline existing roots before editing instead of treating an absent module as an environmental failure.

## Acceptance
- [ ] Required field/capture forms execute through generic existing owners; any new capability has a closed versioned reason and unknown/stale rejection.
- [ ] Independent expected values pin exact codec/evaluator behavior and Program/Contract availability across positive, violated, incomplete and malformed cases.
- [ ] Numeric/schema/capture/work limits reject atomically, with no private Slot/raw payload access or model-specific runtime branches.
- [ ] Affected Lean/Go and owned-generation staleness checks pass; unchanged protocol/Case fixtures retain bytes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
