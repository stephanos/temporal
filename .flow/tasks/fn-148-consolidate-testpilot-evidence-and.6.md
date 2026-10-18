---
satisfies: [R6, R7]
---
# fn-148-consolidate-testpilot-evidence-and.6 Simplify instruction references, cardinality and surviving markers

## Description
**Recut 2026-10-18 (one-language series):** The `tools/umpire/lower` half goes to fn-203's Temporal adapter.

Complete R6's instruction-side cleanup after the Contract cleanup. Derive response cardinality, shorten references only where entrypoint scope is fixed, and replace surviving empty markers.

**Size:** M
**Files:** instruction, program, event and run schemas, generated APIs, execution preparation and response-read binding, lowering
**Touches:** [tools/umpire/lower/producer.go, common/testing/testpilot/internal/execution/prepare.go, common/testing/testpilot/protocol_test.go, common/testing/testpilot/case.go, common/testing/testpilot/casefile/**, common/testing/testpilot/evaluation/admission.go, common/testing/testpilot/evaluation/*_test.go, common/testing/testpilot/recordedrun/**, common/testing/testpilot/**/testdata/**, proto/internal/temporal/server/api/testpilot/v1/*.proto, api/testpilot/v1/*.go, common/testing/testpilot/internal/execution/**, tools/umpire/lower/**]

### Approach
- After all new shapes and consumers are integrated, switch Case emission and admission together to exactly format 4.0 (`tools/umpire/lower/producer.go`, execution `prepare.go`, Case decoding in `case.go`/`casefile`, and independent offline assessment in `evaluation/admission.go`). Update hand-authored fixtures and test builders, and prove 4.0 acceptance plus rejection of formats 1.0, 2.0, 3.0 and unknown formats. Managed artifact regeneration is Task 7.
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
- [ ] Producer, Case decoding, execution and independent offline assessment admission agree on exactly format 4.0; retired and unknown formats reject before payload interpretation or execution. fn-146.1's successful current-format Case/Run assessment and identity-negative proofs pass.
## Done summary
Blocked:
OWNER HOLD (2026-10-16): Umpire framework DSL is the top priority. This integration task is not admitted while fn-188.3 completes final native public-client verification and formal batch review. Preserve its existing source checkpoints, evidence, dependencies and unfinished acceptance obligations; this scheduling hold claims no implementation or proof completion. Completion of fn-188.3 does not automatically release this task: select later IR/exporter/schema/Go/backend work explicitly. Any model/temporal work remains independently excluded entirely, including disabled source contents, until the owner restores that scope.
## Evidence
- Commits:
- Tests:
- PRs:
