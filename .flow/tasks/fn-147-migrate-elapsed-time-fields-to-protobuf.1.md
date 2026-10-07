---
satisfies: [R1, R2]
---
# fn-147-migrate-elapsed-time-fields-to-protobuf.1 Pin checked Duration conversion and presence semantics

## Description
Define and test the shared conversion contract for R1 and R2 before any schema field changes. Inventory all elapsed-time producers and consumers, then pin exact milliseconds, signs, overflow, absent values, explicit zero and monotonic Run coordinates.

**Size:** M
**Files:** Testpilot internal IR or protocol duration helper and tests, Umpire lowering conversion tests, schema inventory fixture
**Touches:** [common/testing/testpilot/internal/ir/*duration*, common/testing/testpilot/internal/ir/*_test.go, tools/umpire/lower/**/*duration*, tools/umpire/lower/**/*_test.go]

### Approach
- Centralize protobuf Duration validation and exact whole-millisecond conversion.
- Give each field family an explicit absent, zero, positive and monotonic policy.
- Test minimum and maximum supported values plus one step outside every bound before schema callers move.

### Investigation targets
**Required** (read before coding):
- `proto/internal/temporal/server/api/testpilot/v1/instruction.proto:35-45` - wait-hint bound
- `proto/internal/temporal/server/api/testpilot/v1/instruction.proto:216-245` - poll interval and timeout
- `proto/internal/temporal/server/api/testpilot/v1/program.proto:225-240` - Program ceilings
- `proto/internal/temporal/server/api/testpilot/v1/contract.proto:118-130` - elapsed deadline
- `proto/internal/temporal/server/api/testpilot/v1/run.proto:24-38` - monotonic Run coordinate
- `.plans/TESTPILOT_SCHEMA_RESEARCH.md:34-38` - accepted field inventory and boundaries


### Quick commands

```bash
go test -tags test_dep ./common/testing/testpilot/internal/ir/... ./tools/umpire/lower/...
```

## Acceptance
- [ ] R1 and R2 conversion policies are executable before schema migration.
- [ ] Boundary tests cover extrema, one-outside values and unsupported precision.
- [ ] Run elapsed-time monotonicity remains distinct from timestamps.
- [ ] Focused conversion tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
