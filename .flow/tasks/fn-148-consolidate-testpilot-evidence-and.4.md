---
satisfies: [R4, R5, R8]
---
# fn-148-consolidate-testpilot-evidence-and.4 Admit and verify normalized correlated tables

## Description
**Note 2026-10-18 (one-language series):** Depends on fn-202 (Testpilot protos move under the framework) and then fn-203's single Case regeneration.

Switch correlated admission and monitoring to the normalized tables for R4 and R5. Preserve continuity, authorization, causal buffering and verdicts while charging expanded work before enumeration.

**Size:** M
**Files:** normalized producer emission and preparation-dependent lowering tests, `common/testing/testpilot/internal/verification/correlated_prepare.go`, `correlated.go`, limits and correlated tests, conformance fixtures
**Touches:** [tools/umpire/lower/internal/producer/**, tools/umpire/lower/**/*_test.go, common/testing/testpilot/internal/execution/**, common/testing/testpilot/internal/verification/correlated_prepare.go, common/testing/testpilot/internal/verification/correlated.go, common/testing/testpilot/internal/verification/**/*correlated*_test.go, tools/umpire/conformance/**]

### Approach
- Restore all normalized consumers and hand-authored test builders, then switch public normalized emission after admission understands the new tables.
- Run the preparation-dependent `TestLoweredCasesPrepareUnderTheirDerivedProfile` after this integration point; managed artifact parity remains Task 7's gate.
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
go test -tags test_dep -run TestLoweredCasesPrepareUnderTheirDerivedProfile ./tools/umpire/lower/...
go test -tags test_dep -run 'Test.*(Evidence|Correlated|Contract|Instruction|ReadSource|Protocol)' ./common/testing/testpilot/internal/verification/... ./tools/umpire/conformance/...
```
## Acceptance
- [ ] R4 and R5 are implemented in admission and monitoring.
- [ ] R8 runtime and admission-cost measurements use the same fixture corpus as Task 3.
- [ ] Compression cannot bypass any existing ceiling.
- [ ] Focused correlated and conformance tests pass.
## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: