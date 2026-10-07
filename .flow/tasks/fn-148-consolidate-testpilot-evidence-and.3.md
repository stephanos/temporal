---
satisfies: [R4, R8]
---
# fn-148-consolidate-testpilot-evidence-and.3 Introduce complete-state and result tables in lowering

## Description
Add the normalized schema and producer representation for R4, then measure compact artifact size before runtime admission switches. Use pre-change fixture projections for the behavior comparison, not a retained runtime representation.

**Size:** M
**Files:** `proto/internal/temporal/server/api/testpilot/v1/correlated.proto`, generated APIs, `tools/umpire/lower/internal/producer/correlated.go`, producer tests and corpus measurement
**Touches:** [proto/internal/temporal/server/api/testpilot/v1/correlated.proto, api/testpilot/v1/correlated*.go, tools/umpire/lower/internal/producer/correlated.go, tools/umpire/lower/internal/producer/**/*test.go, .flow/tmp/**]

### Approach
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
go test -tags test_dep ./tools/umpire/lower/... ./common/testing/testpilot/internal/verification/...
```

## Acceptance
- [ ] R4's normalized tables and deterministic IDs are produced.
- [ ] R8 has comparable pre-runtime size and producer-cost measurements.
- [ ] An expansion harness proves old and new transition semantics equal.
- [ ] Focused producer and state-identity tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
