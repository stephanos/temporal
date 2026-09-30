---
satisfies: [R1, R5, R9]
---
# fn-107-scala-umpire-prototype-for-standalone.5 Check Nexus close/reset outcome and ownership contracts

Touches: [model/scalav2/scala/temporal/nexuscaller/closepolicy/**, model/scalav2/ir/nexus-close*.json, model/scalav2/goir/nexus_close*_test.go, model/scalav2/run.sh]

## Description
Add a bounded linked-run Nexus design specimen using small close/reset/handler contracts. Keep current runtime cancellation work deferred.

**Size:** M
**Files:** proposed model/scalav2/scala/temporal/nexuscaller/closepolicy/Model.scala and Claims.scala, generated IR, design-check test/trace fixtures.

### Approach
- Reuse existing Nexus action vocabulary; keep logical operation/request IDs distinct from run ownership.
- Implement the reviewed cancel/handler/retention/ack cuts and two deliberately faulty policies as Scala declarations.
- Check safety monitors and conditional progress through the generic checker; vary terminal outcomes, rejection types, duplicates, and deadline/retention assumptions.
- Compare only behavior represented by the existing Go baseline; label new close/reset claims as authored design promises.

### Investigation targets
**Required:** model/scalav2/scala/temporal/nexuscaller/Model.scala; model/scalav2/scala/temporal/nexuscaller/Claims.scala; model/scalav2/scala/temporal/nexuscaller/kernel/Nexus.scala; model/go/nexuscaller/model.go; model/scalav2/scala/temporal/nexuscaller/Realization.scala:68.
**Optional:** .flow/specs/fn-79-deferred-nexus-operation-cancellation.md; tests/nexus_workflow_test.go:2078.

### Quick commands
`make umpire-check-scala`; `make lint-scala`; `mise exec -- go test -tags test_dep ./model/scalav2/... ./model/go/nexuscaller/...`.

## Acceptance
- [ ] Both pinned faulty designs produce replayable counterexamples and corrected ownership/retention excludes them within scope.
- [ ] Cancellation receipt/effect/knowledge and immutable history/detached work remain distinct.
- [ ] Terminal outcomes, duplicate reporting, reset cuts, and principal reconstruction satisfy the declared controls.
- [ ] Timeout resolution and finite open prefixes never become unsupported indefinite-hang claims.

- [ ] Legacy model/scala files remain unchanged; v2 generation, lifting, native tests and lint keep passing with that tree absent.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
