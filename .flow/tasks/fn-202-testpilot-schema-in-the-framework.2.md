---
satisfies: [R2]
---
# fn-202-testpilot-schema-in-the-framework.2 Owner decision and implementation of the driver seam for the seven Temporal-typed fields

## Description
**Owner decision 2026-10-18:** google.protobuf.Any with a type-URL registry; Temporal messages in model/temporal/proto; implement that shape, no further owner input needed.

---
satisfies: [R2]
---
# fn-202-testpilot-schema-in-the-framework.2 Owner decision and implementation of the driver seam for the seven Temporal-typed fields

### Description
Present the two seam shapes (google.protobuf.Any with a registry; generic oneof plus a Temporal wrapper), obtain the decision, record it, then implement: the Temporal messages move to model/temporal/proto/testpilot/v1/temporal.proto, the generic instruction carries the seam, Go and Scala registries decode it. Existing fixtures decode unchanged for the oneof shape or are flagged for fn-203's regeneration for Any.

**Files:** model/framework/proto/testpilot/v1/instruction.proto, model/temporal/proto/testpilot/v1/temporal.proto, common/testing/testpilot/** read sites of the seven fields, tools/umpire/lower (until fn-203), .flow/specs/fn-202-*.md.

**Quick commands:** `go test -tags test_dep ./common/testing/testpilot/...`

### Acceptance
- Framework proto imports only cel/expr and well-known types.
- Unregistered type URL fails decoding with the URL.

### Done summary
TBD

### Evidence
- Commits:
- Tests:
- PRs:
## Acceptance
- Framework proto imports only cel/expr and well-known types.
- Unregistered type URL fails decoding with the URL.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
