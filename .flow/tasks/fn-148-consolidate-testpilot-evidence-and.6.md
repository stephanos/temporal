---
satisfies: [R6, R7]
---
# fn-148-consolidate-testpilot-evidence-and.6 Simplify instruction references, cardinality and surviving markers

## Description
Complete R6's instruction-side cleanup after the Contract cleanup. Derive response cardinality, shorten references only where entrypoint scope is fixed, and replace surviving empty markers.

**Size:** M
**Files:** instruction, program, event and run schemas, generated APIs, execution preparation and response-read binding, lowering
**Touches:** [tools/umpire/lower/producer.go, common/testing/testpilot/internal/execution/prepare.go, common/testing/testpilot/protocol_test.go, proto/internal/temporal/server/api/testpilot/v1/*.proto, api/testpilot/v1/*.go, common/testing/testpilot/internal/execution/**, tools/umpire/lower/**]

### Approach
- After all new shapes and consumers are integrated, switch Case emission and admission together to exactly format 4.0 (`tools/umpire/lower/producer.go`, execution `prepare.go`). Update hand-authored fixtures and test builders, and prove 4.0 acceptance plus rejection of formats 1.0, 2.0, 3.0 and unknown formats. Managed artifact regeneration is Task 7.
- Derive `ReadSource.single` from its descriptor-checked response path.
- Use local instruction IDs for `After` and `AwaitInstruction` where enclosing entrypoint admission already applies.
- Retain cross-entrypoint references at authorized sites and preserve absent `After` versus present-empty `After`.
- Replace surviving empty marker payloads with WKT Empty, reassess obsolete event reference enums after CEL, and keep event identity.

### Investigation targets
**Required** (read before coding):
- `proto/internal/temporal/server/api/testpilot/v1/instruction.proto:1-264` - references and markers
- `proto/internal/temporal/server/api/testpilot/v1/program.proto` - read source cardinality
- `proto/internal/temporal/server/api/testpilot/v1/event.proto` - obsolete reference consumers
- `common/testing/testpilot/internal/execution/prepare.go` - graph reference admission
- `common/testing/testpilot/internal/execution/evidence.go` - path-bound cardinality

### Quick commands

```bash
go test -tags test_dep -run 'Test.*(Evidence|Correlated|Contract|Instruction|ReadSource|Protocol)' ./common/testing/testpilot/... ./tools/umpire/lower/...
```
## Acceptance
- [ ] Local references preserve authorized entrypoint boundaries and After absence semantics.
- [ ] Bound response paths determine cardinality for singular and repeated reads.
- [ ] Surviving markers use WKT Empty and retired reference enums have no live consumers.
- [ ] Protocol ownership, execution and lowering checks pass.
- [ ] Producer/admission agree on exactly format 4.0; retired and unknown formats reject before execution.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
