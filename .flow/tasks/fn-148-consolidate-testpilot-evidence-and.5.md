---
satisfies: [R6, R7]
---
# fn-148-consolidate-testpilot-evidence-and.5 Remove derived Contract fields and preserve support presence

## Description
Derive Contract rule kind from deadline and remove the fixed correlated clock for R6. Convert independent support inclusion to an explicitly present Boolean. Keep absent versus present-empty dependencies and all source coordinates whose meaning is not derivable.

**Size:** M
**Files:** Testpilot protobuf schemas, generated APIs, execution and verification admission, lowering and protocol tests
**Touches:** [proto/internal/temporal/server/api/testpilot/v1/*.proto, api/testpilot/v1/*.go, common/testing/testpilot/internal/execution/**, common/testing/testpilot/internal/verification/**, tools/umpire/lower/**]

### Approach
- Derive rule kind from deadline and define the correlated bound as operation transitions.
- Preserve support inclusion as an independently authored, explicitly present Boolean.
- Keep instruction references and cardinality work in Task 6.
- Preserve meaningful Contract presence and update its lowering and verification callers.

### Investigation targets
**Required** (read before coding):
- `proto/internal/temporal/server/api/testpilot/v1/contract.proto:1-149` - rule kind and support fields
- `proto/internal/temporal/server/api/testpilot/v1/correlated.proto:120-235` - fixed clock and bounds
- `proto/internal/temporal/server/api/testpilot/v1/instruction.proto:1-264` - local references and presence wrappers
- `common/testing/testpilot/internal/verification/prepare.go` - derived Contract admission
- `common/testing/testpilot/internal/execution/prepare.go` - dependency and reference admission
- `.plans/TESTPILOT_SCHEMA_RESEARCH.md:62-75` - supported cleanup list and boundaries


### Quick commands

```bash
go test -tags test_dep -run 'Test.*(Evidence|Correlated|Contract|Instruction|ReadSource|Protocol)' ./common/testing/testpilot/... ./tools/umpire/lower/...
```

Scalar singleton-oneof cleanup is owned by fn-147. Protocol marker and instruction-reference cleanup belongs to Task 6.
## Acceptance
- [ ] Contract kind derives from deadline and the correlated clock is operation transitions.
- [ ] Support inclusion uses an explicitly present independent Boolean; omission still rejects where required.
- [ ] Superseded rule and support fields and compatibility admission paths are removed.
- [ ] Focused Contract, correlated and lowering tests pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
