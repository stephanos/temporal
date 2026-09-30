---
satisfies: [R9]
---
# fn-107-scala-umpire-prototype-for-standalone.8 Demonstrate Quint transition agreement and one P monitor

Touches: [model/scalav2/backends/**, model/scalav2/README.md]

## Description
Build narrow adapters from the admitted finite IR to Quint and one P event monitor. Gate each supported backend on semantic agreement.

**Size:** M
**Files:** proposed Quint adapter/runner, P monitor adapter, agreement/replay fixtures, backend README.

### Approach
- Verify primary tool documentation and available pinned toolchains during implementation. Do not silently substitute sampling for exhaustive transition agreement.
- Encode components as finite state, preserving queue order/fault choices, initial states, public results, monitor state, and progress assumptions in the selected slice.
- Enumerate reachable transition agreement with Go and replay external witnesses through ordinary IR admission/evaluation.
- Export one passive authored monitor to P and compare bounded accepted/rejected event traces. State coverage and exclude P module-refinement claims.

### Investigation targets
**Required:** model/scalav2/SEMANTICS.md; proto/internal/temporal/server/api/modelir/v1/ir.proto; model/scalav2/goir/parity_test.go; model/go/umpire/table.go; model/go/umpire/search.go.
**Optional:** .plans/lean/UMPIRE_OUTSIDE_THE_BOX.md:237.

### Quick commands
`mise exec -- go test -tags test_dep ./model/scalav2/...`; document backend-specific agreement commands after verifying the tools.

## Acceptance
- [ ] Quint and Go agree on the entire reachable transition relation and selected properties of the declared finite specimens.
- [ ] External counterexamples replay and rejected/unsupported witnesses remain diagnostic errors.
- [ ] The P monitor and authored Go monitor agree on supported bounded traces, including negative controls.
- [ ] Receipts distinguish transition agreement, checker coverage, and unsupported module-refinement functionality.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
