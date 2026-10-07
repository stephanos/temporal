---
satisfies: [R1, R2, R7]
---
# fn-148-consolidate-testpilot-evidence-and.1 Generalize evidence declarations and extract evidence.proto

## Description
Define the one declaration shape for R1, R2 and R7 before either existing lift representation is removed. Move evidence identity, value and lifting declarations to a leaf schema while preserving the current inline and referenced forms during this task.

**Size:** M
**Files:** `proto/internal/temporal/server/api/testpilot/v1/program.proto`, `correlated.proto`, new `evidence.proto`, generated APIs and protocol tests
**Touches:** [proto/internal/temporal/server/api/testpilot/v1/program.proto, proto/internal/temporal/server/api/testpilot/v1/correlated.proto, proto/internal/temporal/server/api/testpilot/v1/evidence.proto, api/testpilot/v1/*evidence*.go, common/testing/testpilot/protocol_test.go]

### Approach
- Give declarations source identity, projected schema, operation key, scope and field expressions, guards and constants.
- Represent response lifting as an ordered declaration-ID list plus its Observation destination.
- Keep source-specific selection policy outside the common declaration shape.

### Investigation targets
**Required** (read before coding):
- `proto/internal/temporal/server/api/testpilot/v1/program.proto:1-120` - current evidence declarations
- `proto/internal/temporal/server/api/testpilot/v1/correlated.proto:1-120` - inline lift and evidence identities
- `common/testing/testpilot/internal/execution/evidence_lift_test.go:1-240` - supported inline lift surface
- `common/testing/testpilot/internal/execution/evidence_source_test.go:1-180` - source and ambiguity policies
- `.plans/TESTPILOT_SCHEMA_RESEARCH.md:40-48` - target declaration and retained policies


### Quick commands

```bash
go test -tags test_dep ./common/testing/testpilot/...
```

## Acceptance
- [ ] R1's complete declaration vocabulary is generated and descriptor-checked; one lift mixing inline data with declaration IDs remains invalid.
- [ ] R2's ordered and ambiguous selection policies stay representable.
- [ ] R7's evidence ownership produces an acyclic import graph.
- [ ] Protocol and evidence schema tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
