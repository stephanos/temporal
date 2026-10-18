---
satisfies: [R4, R8]
---
# fn-148-consolidate-testpilot-evidence-and.3 Introduce complete-state and result tables in lowering

## Description
**Recut 2026-10-18 (one-language series):** The `tools/umpire/lower` half goes to fn-203's Temporal adapter.

Add the normalized schema and producer representation for R4, then measure compact artifact size before runtime admission switches. Use pre-change fixture projections for the behavior comparison, not a retained runtime representation.

**Size:** M
**Files:** `proto/internal/temporal/server/api/testpilot/v1/correlated.proto`, generated APIs, `tools/umpire/lower/internal/producer/correlated.go`, producer tests and corpus measurement
**Touches:** [tools/umpire/lower/internal/producer/localize.go, tools/umpire/lower/lower.go, proto/internal/temporal/server/api/testpilot/v1/correlated.proto, api/testpilot/v1/correlated*.go, tools/umpire/lower/internal/producer/correlated.go, tools/umpire/lower/internal/producer/**/*test.go, .flow/tmp/**]

### Approach
- Update `tools/umpire/lower/internal/producer/localize.go` and `tools/umpire/lower/lower.go` field inventory/derived accounting for complete-state and result tables.
- Tasks 3 and 4 are one breaking integration batch: removed embedded fields may leave the intermediate consumer tree red. Do not introduce compatibility representations. Restore normalized admission in Task 4 before running preparation-dependent lowerer tests; public format 4.0 emission/admission switches only in Task 6.
- At this stage, run schema generation and generated-API checks plus pure table-builder measurements. Defer `TestLoweredCasesPrepareUnderTheirDerivedProfile` to Task 4 and managed Case/fixture checks to Task 7.
- Intern each complete atom-plus-ordered-fields state locally with deterministic IDs.
- Intern each action, destination, outcome and facts result locally; reference it from transitions and ordered projection outputs.
- Compare expanded old and new structures and record size and construction-cost measurements before deleting old fields.

### Investigation targets
**Required** (read before coding):
- `proto/internal/temporal/server/api/testpilot/v1/correlated.proto:120-235` - transitions, outputs and limits
- `tools/umpire/lower/internal/producer/correlated.go:200-310` - rows, results and state fields
- `common/testing/testpilot/internal/verification/correlated_prepare.go:23-48` - current result equality
- `common/testing/testpilot/internal/verification/correlated_state_test.go:1-100` - atom-plus-fields identity
- `.plans/TESTPILOT_SCHEMA_RESEARCH.md:50-60` - normalization boundary and corpus baseline


### Quick commands

```bash
make proto
go test -tags test_dep ./api/testpilot/v1
```
## Acceptance
- [ ] R4's normalized tables and deterministic IDs are produced.
- [ ] R8 has comparable pre-runtime size and producer-cost measurements.
- [ ] An expansion harness proves old and new transition semantics equal.
- [ ] Schema/API checks and the pure fixture expansion proof pass; consumer-dependent lowering tests wait for Task 4 and managed-artifact checks wait for Task 7.
## Done summary
Blocked:
OWNER HOLD (2026-10-16): Umpire framework DSL is the top priority. This integration task is not admitted while fn-188.3 completes final native public-client verification and formal batch review. Preserve its existing source checkpoints, evidence, dependencies and unfinished acceptance obligations; this scheduling hold claims no implementation or proof completion. Completion of fn-188.3 does not automatically release this task: select later IR/exporter/schema/Go/backend work explicitly. Any model/temporal work remains independently excluded entirely, including disabled source contents, until the owner restores that scope.
## Evidence
- Commits:
- Tests:
- PRs:
