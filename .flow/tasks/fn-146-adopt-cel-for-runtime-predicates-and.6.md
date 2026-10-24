---
satisfies: [R5, R6]
---
# fn-146-adopt-cel-for-runtime-predicates-and.6 Lower Umpire runtime predicates into CEL

## Description
**Recut 2026-10-18 (one-language series):** Operand lowering into CEL is done by fn-203's Scala producer; conformance reads fn-204's descriptor (or the fn-209 oracle on go).

Implement R5 and the Umpire half of R6. Keep symbolic realization operands in the Umpire IR, then lower their supported runtime subset into Testpilot's CEL environment for execution, replay and conformance.

**Size:** M
**Files:** `tools/umpire/realization/**`, `tools/umpire/lower/**`, `tools/umpire/conformance/**`, `model/irgen/**`, realization fixtures
**Touches:** [tools/umpire/lower/producer.go, common/testing/testpilot/internal/execution/prepare.go, common/testing/testpilot/evaluation/admission.go, common/testing/testpilot/case.go, tools/umpire/realization/**, tools/umpire/lower/**, tools/umpire/conformance/**, model/irgen/**, model/irgen/testdata/**]

### Approach
- Complete CEL lowering and focused fixture/test-builder proofs without activating public emission or admission. The single public format 4.0 switch, including Case decoding and independent assessment admission, belongs to fn-148.6 after Duration and evidence/state consumers integrate. Retired and unknown formats reject before execution; no legacy evaluator is retained.
- Run Task 3's deferred adapter proofs and focused online/offline agreement checks when their consumers compile. Task 1's public current-format Case/Run assessment is proved with fn-148.6 activation and the full managed companion surface at fn-148.7.
- Centralize symbolic operand substitution and restricted-path lowering into canonical CEL.
- Reuse the same environment for offline Run Event guards and conformance reads.
- Preserve Umpire source positions, fixture-dependent names, empty-conjunction refusal and unsupported-realization diagnostics.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/realization/operand.go:45-150` - operand typing
- `tools/umpire/realization/payload.go:78-170` - guard restrictions
- `tools/umpire/lower/realization.go` - `adapter.operand` and `adapter.guardOf` lower symbolic operands and Run Event guards
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
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: