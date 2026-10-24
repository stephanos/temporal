---
satisfies: [R2]
---
# fn-202-testpilot-schema-in-the-framework.2 Owner decision and implementation of the driver seam for the seven Temporal-typed fields

## Description
**Owner decision 2026-10-18:** google.protobuf.Any with a type-URL registry; Temporal messages in model/temporal/proto; implement that shape, no further owner input needed.

Present the two seam shapes (google.protobuf.Any with a registry; generic oneof plus a Temporal wrapper), obtain the decision, record it, then implement: the Temporal messages move to model/temporal/proto/testpilot/v1/temporal.proto, the generic instruction carries the seam, Go and Scala registries decode it. Existing fixtures decode unchanged for the oneof shape or are flagged for fn-203's regeneration for Any.

**Files:** model/framework/proto/testpilot/v1/instruction.proto, model/temporal/proto/testpilot/v1/temporal.proto, common/testing/testpilot/** read sites of the seven fields, tools/umpire/lower (until fn-203), .flow/specs/fn-202-*.md.

**Quick commands:** `go test -tags test_dep ./common/testing/testpilot/...`
## Acceptance
- Framework proto imports only cel/expr and well-known types.
- Unregistered type URL fails decoding with the URL.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: