---
satisfies: [R2, R8, R9]
---
# fn-77-typed-operations-parameterized-actions.2 Admit exact bounded structural operation values

## Description
Admit exact bounded structural operation values for the referenced parent requirements.

**Size:** M
**Files:** model/Umpire/Operation/**; model/Umpire/Value.lean; model/Umpire/Value/**; tools/umpire/cmd/umpire-gen-lean-api/**; model/Temporal/API/**
**Touches:** [model/Umpire/Operation/**, model/Umpire/Value.lean, model/Umpire/Value/**, tools/umpire/cmd/umpire-gen-lean-api/**, model/Temporal/API/**]

### Approach
- Implement a schema-driven checked concrete value carrier alongside descriptive summaries; cover nested messages, exact bytes, typed integer/enum values, presence, oneof discriminants, ordered repeated values and canonical decoded maps.
- Use schema identity from task1 and bounded recursive traversal with depth/work/collection/byte limits. Keep schema recursion metadata complete while rejecting exhausted concrete access; never fabricate empty subtrees.
- Normalize map inputs according to decoded protobuf semantics (last value for duplicate raw keys, canonical typed-key order after decoding). Pin exact signed/unsigned ranges and open-enum unknown numbers; reject unsupported closed-enum/special forms explicitly.
- Declare floating-point field evaluation/operators unsupported initially with responsible source diagnostics; do not substitute approximate/text comparisons. Preserve complete structural discovery and every required qualifying clause.
- Prove checked canonical encode/decode preserves exact values and presence; retain old descriptive representations and record supported forms in module docs.

### Investigation targets
**Required:**
- model/Temporal/API/Proto.lean:15 — Bytes/MessageRef summaries are not concrete values.
- tools/umpire/cmd/umpire-gen-lean-api/model.go:255 — descriptor presence/map metadata.
- tools/umpire/cmd/umpire-gen-lean-api/lean_plan.go:749 — recursive reference substitution.
- model/Testpilot/Authoring.lean:35 — existing portable concrete bytes/numeric constructors.
- common/testing/testpilot/internal/ir/runtime_value.go — actual codec semantics to match.

### Quick commands
`go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-gen-lean-api ./common/testing/testpilot/internal/ir`
`cd model && mise exec -- lake build Testpilot.Tests Umpire.TargetTests`

`cd model && mise exec -- lake build Umpire.Value.Tests`

### Execution constraints
Read the full parent spec. Preserve existing comments and unrelated dirty source. New paths listed here are proposed owners; reuse an established equivalent before creating one. Run Lean jobs serially; fn76 lint recovery is complete. Preserve fn75 semantic import isolation and fn78 obligation/evidence authority. No operation cancellation, new dependency/toolchain, or implicit fixture promotion. Do not stage, commit, or push; the user owns commits. Capture task-local before/after trust for changed load-bearing declarations; task11 additionally compares to the original task1 substrate.

Create and wire the proposed test module Umpire.Value.Tests into its existing aggregate test root; if an existing equivalent is reused, record its exact name and run it explicitly. The new-module Quick command applies after creation; baseline existing roots before editing instead of treating an absent module as an environmental failure.

## Acceptance
- [ ] Independent fixtures preserve required exact forms, including same-length different bytes, optional absent/default, unknown enums, integer boundaries, repeated order and keyed map normalization.
- [ ] Recursive depth, payload/collection bounds and unsupported forms fail explicitly without truncation or fabricated values.
- [ ] Canonical round-trip proof covers admitted exact values and schema identity; pre-existing descriptive consumers and bytes remain compatible.
- [ ] New owner tests are wired into normal Lean test roots; focused generator/codec/model checks pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
