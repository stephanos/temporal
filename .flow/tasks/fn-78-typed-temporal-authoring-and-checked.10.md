---
satisfies: [R6, R7, R8, R9]
---
# fn-78-typed-temporal-authoring-and-checked.10 Qualify generic scoped authoring without cancellation

## Description
Deliver reusable bounded temporal syntax and non-cancellation qualification formerly bundled in task 8. Resolve typed trigger/response, correlation key, clock, natural bound, endpoint, and source-local diagnostics through the same checked clauses. Use generic multi-operation fixtures for scoped Contract parity, and preserve the existing Nexus success Prepare/Run integration. Update model/README.md, model/ARCHITECTURE.md, model/Umpire/ARCHITECTURE.md and Nexus3/Integration.md to describe the generic delivery and explicitly deferred cancellation. Do not implement a cancellation Target, evidence adapter, capability, or Case.

**Size:** L
**Touches:** [model/Umpire/Property/**, model/Temporal/Feature/Nexus3/**, model/Temporal/Tool/Testpilot.lean, tests/testcore/testpilot/**, tests/testpilot_async_nexus_case_test.go, model/README.md, model/ARCHITECTURE.md, model/Umpire/ARCHITECTURE.md]

## Acceptance
- [ ] Readable bounded temporal notation and typed constructors have identical checked canonical meaning and fingerprints.
- [ ] Compile-failure tests reject wrong contexts, references, keys, clocks, scopes, bounds, raw evidence, command effects, and unsupported formulas at the author expression.
- [ ] Authors supply semantic choices without serialization/proof/monitor plumbing.
- [ ] A deterministic admitted non-cancellation Case exercises scoped obligations through the public Prepare/Run facade; existing Nexus success/rejection integration remains green.
- [ ] Generic violating, incomplete, wrong-correlation, stopped/lost execution, and cleanup-failure fixtures preserve verdict/disposition and prior proof.
- [ ] Repeated/concurrent Runs and tenfold evidence/obligation loads demonstrate isolation and bounded failure.
- [ ] Required model builds/lints, regression/staleness gates, focused Go tests with test_dep, and existing success integration with test_dep integration pass or record verified inherited failures.
- [ ] Architecture and authoring documentation matches the generic delivery and retains cancellation as explicitly deferred to fn-79.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
