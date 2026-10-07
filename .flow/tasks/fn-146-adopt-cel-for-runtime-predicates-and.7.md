---
satisfies: [R1, R4, R5, R6, R7]
---
# fn-146-adopt-cel-for-runtime-predicates-and.7 Retire duplicate predicate machinery and close the CEL migration

## Description
Remove superseded expression machinery after its callers migrate, regenerate checked-in companions and run current artifact gates, and update the ownership and semantics records for R1 through R7.

**Size:** M
**Files:** retired Testpilot expression implementation, `model/SEMANTICS.md`, `model/README.md`, `.plans/UMPIRE_MODULES.md`, `.plans/UMPIRE4_SPEC.md`, `MILESTONES.md`, generated artifacts and fixtures
**Touches:** [common/testing/testpilot/casefile/**, common/testing/testpilot/evaluation/**, common/testing/testpilot/temporal/**, common/testing/testpilot/internal/ir/**, model/SEMANTICS.md, model/README.md, .plans/UMPIRE_MODULES.md, .plans/UMPIRE4_SPEC.md, MILESTONES.md, model/cases/**, common/testing/testpilot/**/testdata/**]

### Approach
- Regenerate managed Case/Run companions under exactly 2.0 after all consumers compile; only now run generated-artifact parity and full-protocol gates.
- Use live-caller and ownership checks to identify removable evaluator and walker code; retain no legacy replay path.
- Regenerate checked-in Case and Run companions and replay the current format.
- Record the pinned environment, semantic decisions, conversion boundary and remaining domain admission owners.

### Investigation targets
**Required** (read before coding):
- `.plans/UMPIRE_CEL_RUNTIME_RESEARCH.md:134-154` - migration decision and retirement checklist
- `.plans/UMPIRE_MODULES.md:20-80` - schema and runtime ownership rows
- `model/SEMANTICS.md:560-640` - current runtime guard semantics
- `common/testing/testpilot/protocol_test.go:80-175` - protocol contract
- `MILESTONES.md:19-49` - full-gate evidence discipline


### Quick commands

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
- [ ] R1's current-format migration and R4-R6 agreement pass on the full fixture surface.
- [ ] R7's superseded machinery and compatibility dispatch are gone; ownership docs are current.
- [ ] No uncategorized Case, Run, verdict, identity or conformance delta remains.
- [ ] Required full gates pass with evidence recorded for review.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
