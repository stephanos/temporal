---
satisfies: [R1, R4, R5, R6, R7]
---
# fn-146-adopt-cel-for-runtime-predicates-and.7 Retire duplicate predicate machinery and close the CEL migration

## Description
Remove superseded expression machinery after its callers migrate, categorize CEL identity deltas, and update the ownership and semantics records for R1 through R7. Managed companion regeneration, full gates, review and live-run evidence belong to the shared fn-148.7 close; fn-146 does not close before that evidence is recorded.

**Size:** M
**Files:** retired Testpilot expression implementation, `model/SEMANTICS.md`, `model/README.md`, `.plans/UMPIRE_MODULES.md`, `.plans/UMPIRE4_SPEC.md`, generated artifacts and fixtures
**Touches:** [common/testing/testpilot/casefile/**, common/testing/testpilot/evaluation/**, common/testing/testpilot/temporal/**, common/testing/testpilot/internal/ir/**, model/SEMANTICS.md, model/README.md, .plans/UMPIRE_MODULES.md, .plans/UMPIRE4_SPEC.md, model/cases/**, common/testing/testpilot/**/testdata/**]

### Approach
- Hand the milestone summary and shared-close evidence to the conductor. Only the conductor edits `MILESTONES.md` and completes its required update at the shared fn-148.7 close; workers do not edit that file.
- Prepare the CEL identity-delta ledger against the retained baseline; consume fn-148.7's single format 4.0 regeneration, generated-artifact parity and full-protocol evidence.
- Use live-caller and ownership checks to identify removable evaluator and walker code; retain no legacy replay path.
- Account for every checked-in Case and Run companion; their regeneration and current-format replay occur once at fn-148.7.
- Record the pinned environment, semantic decisions, conversion boundary and remaining domain admission owners.

### Investigation targets
**Required** (read before coding):
- `.plans/UMPIRE_CEL_RUNTIME_RESEARCH.md:134-154` - migration decision and retirement checklist
- `.plans/UMPIRE_MODULES.md:20-80` - schema and runtime ownership rows
- `model/SEMANTICS.md` - Realizations, especially Run Event payload guards and shared `GuardProblem` admission
- `common/testing/testpilot/protocol_test.go:80-175` - protocol contract
- `MILESTONES.md:19-49` - full-gate evidence discipline


### Shared-close commands (run once at fn-148.7)

```bash
make umpire-gen-model
make umpire-gen-fixtures
make canary-gen-case
make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks
go test -tags test_dep -p 2 -timeout 30m ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/...
make umpire-check-cases
make umpire-check-fixtures
make canary-check-case
make lint-model
make lint-code-fast
make umpire-check-live-tests
```
## Acceptance
- [ ] R1's current-format migration and R4-R6 agreement pass on the full fixture surface, with shared fn-148.7 evidence linked before spec closure.
- [ ] R7's superseded machinery and compatibility dispatch are gone; ownership docs are current.
- [ ] No uncategorized Case, Run, verdict, identity or conformance delta remains.
- [ ] Required full gates, review and live run pass at fn-148.7 with evidence linked here before spec closure; this task does not independently regenerate or close the spec.
## Done summary
Blocked:
OWNER HOLD (2026-10-16): Umpire framework DSL is the top priority. This integration task is not admitted while fn-188.3 completes final native public-client verification and formal batch review. Preserve its existing source checkpoints, evidence, dependencies and unfinished acceptance obligations; this scheduling hold claims no implementation or proof completion. Completion of fn-188.3 does not automatically release this task: select later IR/exporter/schema/Go/backend work explicitly. Any model/temporal work remains independently excluded entirely, including disabled source contents, until the owner restores that scope.
## Evidence
- Commits:
- Tests:
- PRs:
