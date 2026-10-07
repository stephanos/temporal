---
satisfies: [R3]
---
# fn-146-adopt-cel-for-runtime-predicates-and.3 Build the descriptor-aware CEL value adapter

## Description
Implement R3 independently of evaluator rollout. Convert between admitted Testpilot values, protobuf values and CEL values under the Case's descriptor catalog while retaining model atoms and `ValueType` as separate contracts.

**Size:** M
**Files:** `proto/internal/temporal/server/api/testpilot/v1/value.proto`, `common/testing/testpilot/internal/ir/runtime_value.go`, `common/testing/testpilot/internal/ir/type.go`, value tests
**Touches:** [proto/internal/temporal/server/api/testpilot/v1/value.proto, api/testpilot/v1/value*.go, common/testing/testpilot/internal/ir/runtime_value.go, common/testing/testpilot/internal/ir/type.go, common/testing/testpilot/internal/ir/*value*_test.go]

### Approach
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
go test -tags test_dep ./common/testing/testpilot/internal/ir/...
```
Format 2.0 replaces the custom value container family with standard CEL values. Distinguish invalid authored assignments from admitted unknown payload wire fields, which remain preserved.

## Acceptance
- [ ] R3's descriptor-aware round trips cover every supported runtime value family.
- [ ] Opaque and catalog-only messages do not use unsafe global resolution.
- [ ] Numeric extrema and one-outside-boundary tests pass.
- [ ] Focused runtime-value and type tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
