---
satisfies: [R2, R3, R9]
---
# fn-77-typed-operations-parameterized-actions.3 Check typed schema field references and value access

## Description
Check typed schema field references and value access for the referenced parent requirements.

**Size:** M
**Files:** model/Umpire/Value/**; model/Umpire/Operation/**; tools/umpire/cmd/umpire-gen-lean-api/**; model/Temporal/API/**
**Touches:** [model/Umpire/Value/**, model/Umpire/Operation/**, tools/umpire/cmd/umpire-gen-lean-api/**, model/Temporal/API/**]

### Approach
- Expose generated typed field references for the full declared selected-operation schema, retaining field-number/containing-schema identity rather than Lean names.
- Build checked nested access, bounded repeated index/cardinality, map lookup, presence and oneof selection over task2 values. Access admission tracks descriptor-specific availability and types.
- Keep implicit scalar defaults distinct from explicit-presence fields; a selected oneof or established presence is required before consuming dependent values. Unsupported structural forms remain discoverable but reject requested evaluation.
- Prove admitted field-path denotation against task2 concrete values; add source-local negative references, wrong schema/oneof and out-of-range selection fixtures.

### Investigation targets
**Required:**
- tools/umpire/cmd/umpire-gen-lean-api/model.go:255 — full descriptor field metadata.
- common/testing/testpilot/internal/ir/path.go:54 — bound descriptor paths.
- common/testing/testpilot/internal/ir/path.go:151 — descriptor-presence restriction.
- model/Umpire/Property/Language.lean:61 — existing predicate contexts.

### Quick commands
`go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-gen-lean-api ./common/testing/testpilot/internal/ir`
`cd model && mise exec -- lake build Umpire.Property.Tests Testpilot.Tests`

`cd model && mise exec -- lake build Umpire.Value.FieldTests`

### Execution constraints
Read the full parent spec. Preserve existing comments and unrelated dirty source. New paths listed here are proposed owners; reuse an established equivalent before creating one. Run Lean jobs serially; fn76 lint recovery is complete. Preserve fn75 semantic import isolation and fn78 obligation/evidence authority. No operation cancellation, new dependency/toolchain, or implicit fixture promotion. Do not stage, commit, or push; the user owns commits. Capture task-local before/after trust for changed load-bearing declarations; task11 additionally compares to the original task1 substrate.

Create and wire the proposed test module Umpire.Value.FieldTests into its existing aggregate test root; if an existing equivalent is reused, record its exact name and run it explicitly. The new-module Quick command applies after creation; baseline existing roots before editing instead of treating an absent module as an environmental failure.

## Acceptance
- [ ] All declared selected-schema fields have stable structural references; exact executable support/unsupported diagnostics are explicit rather than sample-field whitelists.
- [ ] Nested, repeated, map, scalar presence and oneof access enforce schema types and availability, including missing and boundary cases.
- [ ] Checked access correspondence proves returned values/presence match admitted concrete field semantics.
- [ ] Negative source-location tests and unchanged literal/portable tests pass through normal owner roots.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
