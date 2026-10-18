---
satisfies: [R1, R2, R3]
---
# fn-148-consolidate-testpilot-evidence-and.2 Unify evidence binding, lowering and policy checks

## Description
**Recut 2026-10-18 (one-language series):** The `tools/umpire/lower` half goes to fn-203's Temporal adapter.

Move execution binding and Umpire lowering to the generalized declaration, then retire inline lift fields. Preserve first-match response order, Run Event ambiguity, dense ordinals, causal parents and independent Contract policies for R1 through R3.

**Size:** M
**Files:** `common/testing/testpilot/internal/execution/dataflow.go`, `evidence.go`, response-read and scheduler code, `tools/umpire/lower/internal/producer/**`, evidence tests
**Touches:** [proto/internal/temporal/server/api/testpilot/v1/*.proto, api/testpilot/v1/*.go, common/testing/testpilot/internal/verification/correlated_prepare.go, common/testing/testpilot/internal/execution/dataflow.go, common/testing/testpilot/internal/execution/evidence.go, common/testing/testpilot/internal/execution/response_read.go, common/testing/testpilot/internal/execution/**/*evidence*_test.go, tools/umpire/lower/internal/producer/**]

### Approach
- Consolidate shape binding into one declaration admission path.
- Keep selection policy per source and preserve the single emitter's ordinal and ownership rules.
- Leave Contract retention, redaction, rejection, source and scope checks independent from Program declarations.

### Investigation targets
**Required** (read before coding):
- `common/testing/testpilot/internal/execution/dataflow.go:491-625` - inline lift binding
- `common/testing/testpilot/internal/execution/evidence.go:22-195` - declaration binding
- `common/testing/testpilot/internal/execution/evidence.go:274-350` - declared rules and reads
- `common/testing/testpilot/internal/execution/response_read.go` - ordered response projection
- `tools/umpire/lower/internal/producer/build.go` - producer declaration normal form
- `common/testing/testpilot/internal/verification/correlated_prepare.go:93-220` - Contract policy admission

### Key context
The prior extraction consolidation dropped a caller-specific nonempty rejection. Compare rejection sets and error identities before deleting either binder.


### Quick commands

```bash
go test -tags test_dep -run 'Test.*(Evidence|Correlated|Contract|Instruction|ReadSource|Protocol)' ./common/testing/testpilot/internal/execution/... ./common/testing/testpilot/internal/verification/... ./tools/umpire/lower/...
```
## Acceptance
- [ ] R1-R3 pass across execution, lowering and verification fixtures.
- [ ] Inline fields are retired and reserved only after equivalent declarations pass.
- [ ] Caller-specific absent, empty and ambiguity diagnostics remain distinct.
- [ ] Focused evidence lift, source and generic Contract tests pass.
## Done summary
Blocked:
OWNER HOLD (2026-10-16): Umpire framework DSL is the top priority. This integration task is not admitted while fn-188.3 completes final native public-client verification and formal batch review. Preserve its existing source checkpoints, evidence, dependencies and unfinished acceptance obligations; this scheduling hold claims no implementation or proof completion. Completion of fn-188.3 does not automatically release this task: select later IR/exporter/schema/Go/backend work explicitly. Any model/temporal work remains independently excluded entirely, including disabled source contents, until the owner restores that scope.
## Evidence
- Commits:
- Tests:
- PRs:
