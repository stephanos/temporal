---
satisfies: [R3]
---
# fn-146-adopt-cel-for-runtime-predicates-and.3 Build the descriptor-aware CEL value adapter

## Description
Implement R3 independently of evaluator rollout. Convert between admitted Testpilot values, protobuf values and CEL values under the Case's descriptor catalog while retaining model atoms and `ValueType` as separate contracts.

**Size:** M
**Files:** all dependent runtime-value protobuf fields and generated APIs, `proto/internal/temporal/server/api/testpilot/v1/value.proto`, `common/testing/testpilot/internal/ir/runtime_value.go`, `common/testing/testpilot/internal/ir/type.go`, value tests
**Touches:** [proto/internal/temporal/server/api/testpilot/v1/run.proto, proto/internal/temporal/server/api/testpilot/v1/instruction.proto, proto/internal/temporal/server/api/testpilot/v1/program.proto, proto/internal/temporal/server/api/testpilot/v1/contract.proto, api/testpilot/v1/*.go, proto/internal/temporal/server/api/testpilot/v1/value.proto, api/testpilot/v1/value*.go, common/testing/testpilot/internal/ir/runtime_value.go, common/testing/testpilot/internal/ir/type.go, common/testing/testpilot/internal/ir/*value*_test.go]

### Approach
- Replace every dependent protobuf `Value` field, including Run, instruction, Program and Contract schemas; regenerate their APIs. Do not leave references to the removed custom family.
- The schema/adapter and execution task form a breaking integration batch. Consumer-dependent compilation and tests may remain red until Task 4 updates public Drivers and live activation consumers. Run generated-API checks here; preserve adapter test specifications and run them once consumers compile.
- Use authoritative descriptor lookup for messages, enums, maps and narrowing.
- Preserve opaque `Any` values when their concrete descriptor is intentionally unavailable.
- Pin signed and unsigned extrema, bytes and nested values without decimal-string or floating-point loss.

### Investigation targets
**Required** (read before coding):
- `proto/internal/temporal/server/api/testpilot/v1/value.proto:1-124` - current runtime and model value families
- `common/testing/testpilot/internal/ir/runtime_value.go:28-180` - snapshot and budget boundary
- `common/testing/testpilot/internal/ir/type.go` - descriptor-exact admission
- `.plans/TESTPILOT_SCHEMA_RESEARCH.md:28-32` - tested adapter gaps and retained contracts
- `.flow/memory/bug/integration/check-unbounded-lean-numbers-before-2026-09-07.md` - narrowing failure history


### Quick commands

```bash
make proto
go test -tags test_dep ./api/testpilot/v1
```
Format 2.0 replaces the custom value container family with standard CEL values. Distinguish invalid authored assignments from admitted unknown payload wire fields, which remain preserved.
## Acceptance
- [ ] R3's descriptor-aware round trips cover every supported runtime value family.
- [ ] Opaque and catalog-only messages do not use unsafe global resolution.
- [ ] Numeric extrema and one-outside-boundary tests pass.
- [ ] Generated-API checks pass; adapter/type tests and consumer-dependent gates run once Task 4 restores compilation.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
