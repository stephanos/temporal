---
satisfies: [R1, R8]
---

# fn-28-portable-evaluation-contract-and.2 Implement deterministic contract packing and admission
## Description

Build the semantic-free Go codec that converts Lean-produced canonical ProtoJSON into deterministic protobuf bytes and strictly admits those bytes before runtime I/O.

**Size:** M
**Files:** `tools/umpire/evaluationcontract/**`
**Touches:** [`tools/umpire/evaluationcontract/**`]

### Approach
- Use generated protobuf types and deterministic protobuf marshaling; keep checksums outside their own digest input and require canonical re-encoding equality.
- Reject unknown fields, invalid enum values, unsupported major versions/operators, duplicates, noncanonical ordering, invalid Limits, crossed bindings, and checksum/fingerprint drift.
- Keep the packer structural: it may validate schema invariants but cannot select clauses, synthesize expected behavior, or consult model definitions.

### Investigation targets

**Required** (read before coding):
- Parent spec and task `.1` schema.
- `tools/umpire/artifact` and `tools/umpire/internal/artifactv2` canonical admission patterns.
- Existing protobuf deterministic marshal/descriptor helpers in `tools/umpire`.

## Acceptance
- [ ] Canonical ProtoJSON packs to stable deterministic protobuf bytes and re-admits byte-identically.
- [ ] Representative one-field, unknown-field/operator, ordering, checksum, crossed-binding, N/N+1, and fuzz mutations fail closed before execution.
- [ ] Focused unit, race, fuzz, and lint checks pass using `require` assertions.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
