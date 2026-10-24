---
satisfies: [R1, R2]
---
# fn-147-migrate-elapsed-time-fields-to-protobuf.1 Pin checked Duration conversion and presence semantics

## Description
**Note 2026-10-18 (one-language series):** Depends on fn-202 (Testpilot protos move under the framework) and then fn-203's single Case regeneration.

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
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: