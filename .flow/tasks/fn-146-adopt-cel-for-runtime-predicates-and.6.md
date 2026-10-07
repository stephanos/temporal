---
satisfies: [R5, R6]
---
# fn-146-adopt-cel-for-runtime-predicates-and.6 Lower Umpire runtime predicates into CEL

## Description
Implement R5 and the Umpire half of R6. Keep symbolic realization operands in the Umpire IR, then lower their supported runtime subset into Testpilot's CEL environment for execution, replay and conformance.

**Size:** M
**Files:** `tools/umpire/realization/**`, `tools/umpire/lower/**`, `tools/umpire/conformance/**`, `model/irgen/**`, realization fixtures
**Touches:** [tools/umpire/realization/**, tools/umpire/lower/**, tools/umpire/conformance/**, model/irgen/**, model/irgen/testdata/**]

### Approach
- Centralize symbolic operand substitution and restricted-path lowering into canonical CEL.
- Reuse the same environment for offline Run Event guards and conformance reads.
- Preserve Umpire source positions, fixture-dependent names, empty-conjunction refusal and unsupported-realization diagnostics.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/realization/operand.go:45-150` - operand typing
- `tools/umpire/realization/payload.go:78-170` - guard restrictions
- `tools/umpire/lower/realization.go:884-1000` - operand lowering
- `tools/umpire/conformance/guard.go:100-175` - offline guard evaluation
- `model/irgen/testdata/lifts/Realizations.scala:280-330` - lifted realization fixtures


### Quick commands

```bash
go test -tags test_dep ./tools/umpire/realization/... ./tools/umpire/lower/... ./tools/umpire/conformance/...
```

## Acceptance
- [ ] R5's lowering and diagnostic contract covers supported and rejected operands.
- [ ] R6's live, replay and conformance paths agree on real correlated fixtures.
- [ ] No Umpire IR import of Testpilot schemas is introduced.
- [ ] Focused lifter, realization, lowering and conformance tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
