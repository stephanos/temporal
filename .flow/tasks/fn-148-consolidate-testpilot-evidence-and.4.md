---
satisfies: [R4, R5, R8]
---
# fn-148-consolidate-testpilot-evidence-and.4 Admit and verify normalized correlated tables

## Description
Switch correlated admission and monitoring to the normalized tables for R4 and R5. Preserve continuity, authorization, causal buffering and verdicts while charging expanded work before enumeration.

**Size:** M
**Files:** `common/testing/testpilot/internal/verification/correlated_prepare.go`, `correlated.go`, limits and correlated tests, conformance fixtures
**Touches:** [common/testing/testpilot/internal/verification/correlated_prepare.go, common/testing/testpilot/internal/verification/correlated.go, common/testing/testpilot/internal/verification/**/*correlated*_test.go, tools/umpire/conformance/**]

### Approach
- Resolve IDs into checked complete states and results during bounded admission.
- Authorize each transition by prior state and result, then retain ordered projection outputs.
- Charge referenced and expanded structures with overflow-safe ceilings before allocating catalogs or products.

### Investigation targets
**Required** (read before coding):
- `common/testing/testpilot/internal/verification/correlated_prepare.go:93-220` - current whole-contract admission
- `common/testing/testpilot/internal/verification/correlated.go:44-173` - state continuity and causal graph
- `common/testing/testpilot/internal/verification/correlated.go:204-480` - projection, matching and retention
- `common/testing/testpilot/internal/verification/correlated.go:547-590` - work accounting
- `.flow/memory/bug/runtime-errors/finite-ir-admission-must-count-work-2026-09-30.md` - bounds-before-enumeration rule


### Quick commands

```bash
go test -tags test_dep ./common/testing/testpilot/internal/verification/... ./tools/umpire/conformance/...
```

## Acceptance
- [ ] R4 and R5 are implemented in admission and monitoring.
- [ ] R8 runtime and admission-cost measurements use the same fixture corpus as Task 3.
- [ ] Compression cannot bypass any existing ceiling.
- [ ] Focused correlated and conformance tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
