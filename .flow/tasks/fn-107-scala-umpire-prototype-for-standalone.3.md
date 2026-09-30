---
satisfies: [R3, R4, R5, R9]
---
# fn-107-scala-umpire-prototype-for-standalone.3 Adapt generic Go checking to the finite semantic IR

Touches: [model/scalav2/goir/**, model/go/umpire/search.go, model/go/umpire/refine.go, model/go/umpire/claims.go]

## Description
Adapt the existing finite Go checker for IR properties, passive monitors, composition, scoped refinement, and support accounting.

**Size:** M
**Files:** goir machine.go and proposed checking.go/checking_test.go; existing generic search/refine seams if adaptation requires them.

### Approach
- Reuse umpire table/search/composition owners instead of another feature checker. Include monitor state in visited identity.
- Check initial correspondence plus visible event/result projection; reject public-output stutter. Enumerate opaque-provider behaviors under selected bounds.
- Add reusable deadline/deadlock/fair-cycle progress checks with explicit assumptions. Report unestablished progress as unresolved rather than infer it from a prefix.
- Keep admission errors, exhausted resource limits, semantic holes, and failed witnesses distinct. Replay every generated witness before receipt finalization.
- Add a tenfold finite-input probe with an explicit small work ceiling. Assert complete coverage under sufficient ceilings or an explicit resource-limit result under insufficient ceilings, preserving actual scope/work counts and never claiming truncated exploration is exhaustive.

### Investigation targets
**Required:** model/go/umpire/search.go:10; model/go/umpire/compose.go:18; model/go/umpire/claims.go:140; model/scalav2/goir/machine.go:311; model/scalav2/goir/load.go.
**Optional:** model/go/umpire/refine.go; model/scalav2/goir/parity_test.go.

### Quick commands
`mise exec -- go test -tags test_dep ./model/scalav2/... ./model/go/umpire/...`.

## Acceptance
- [ ] Finite search preserves distinct passive-monitor histories and never lets a monitor suppress a behavior.
- [ ] Initial-state and visible-output refinement mutation controls fail, while invisible provider steps pass.
- [ ] Opaque assumptions, reachable holes, deadline/deadlock/cycle witnesses, and bound exhaustion have distinct receipts; the tenfold probe completes within ceilings or reports an explicit resource limit without silent truncation.
- [ ] Rejected witnesses remain errors and valid witnesses replay through the same interpreter.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
