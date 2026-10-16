---
satisfies: [R6]
---
# fn-173-reachable-exploration-drop-full-tables.17 Reconcile native batch fixtures for lazy init and inferred assumptions

## Description
Reconcile the two fixture mismatches exposed by the first broad native batch run after .16. The run compiled successfully and executed 451 tests across 57 suites: 449 passed, two failed. Preserve this failing evidence and the approved callback-free native contracts; do not change production behavior to recover obsolete fixture expectations.

**Size:** S, one bounded fixture/verification checkpoint.
**Files:** model/framework/dsl/CoreBindings.test.scala, model/framework/dsl/ResidualSignature.test.scala.
**Touches:** [model/framework/dsl/CoreBindings.test.scala, model/framework/dsl/ResidualSignature.test.scala]

### Approach
- In the CoreBindings conversion test, pin zero init reads during construction and repeated metadata conversion. Keep refinement metadata and contextual operand acquisition once-only and callbacks at zero. Explicit subsequent derived.init reads must return the literal result and memoize source initialization once, without later conversion rereading it.
- In the ordinary-assumption fixture, make the actual local declaration name agree with its retained name while preserving every expected exported ID/name, ordered duplicate/empty fairness segment and raw empty-name assertion.
- Add a narrow regression distinguishing inferred native spelling from compiler-owned source identity through fairness copies: incompatible retained spelling must not invent identity; matching metadata supplies the qualified ID. Do not weaken DeclarationName retention, derive IDs from names or relax missing-identity refusals.
- Production behavior is already intended. If evidence instead requires a production change, return to the conductor for re-scoping rather than silently expanding Touches.

### Investigation targets
- CoreBindings.test.scala conversion/init/refinement/contextual operand counter test; Machine.scala built/Built and CoreBindings.collect.
- ResidualSignature.test.scala ordinary assumption test; Assume.scala inferredAssumption/fair; DeclarationName.scala retain/copyTo; NativeLocation.declaredName in Action.scala.
- Parent native construction/query contracts and fn-123 .10 inferred assumption contract.
- Actual failed log .flow/tmp/native-dsl-batch-8bf89c02b6-tests.log, with terminal rc1; no setup error caused either failure.

### Verification
Use the actual red batch as the source baseline: no framework source changed since it, only task/milestone metadata. Keep source/test writes frozen during every compiler epoch. Run strict native compilation and the full unfiltered native-root suite, requiring nonzero counts and no suite exclusions; run scoped lint RuleSuites, changed-Scala formatting and diff checks. Record exact terminal codes/counts/logs. Formal review and fresh public-client batch remain conductor-owned after this correction; no held IR/Go/Temporal proof credit follows.

```bash
mise exec -- scala-cli compile --server=false --test model/framework/dsl model/framework/realize model/framework/samples
mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests
mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only 'framework.check.lint.*RuleSuite'
```

No exporter/plugin/schema/Go/backend/regeneration, full Model gate, catalog walks or Temporal reads. Do not edit the six retained sample texts/maps or the conductor's ignored external-client preparation.

## Acceptance
- [ ] Metadata construction/conversion reads init zero times; explicit derived init returns the correct result once and is memoized. Existing callback/refinement/operand assertions remain or strengthen.
- [ ] Ordinary-assumption fixture retains all original ID/name/fairness assertions with matching declaration spelling. A regression distinguishes inferred spelling, incompatible retained metadata and compiler-owned identity without changing production retention.
- [ ] Strict compile, full unfiltered native-root tests with nonzero counts, scoped lint and changed-file formatting pass; initial 449-pass/two-fail batch evidence remains recorded. No exclusion, stale-jar or missing-identity workaround is introduced.
- [ ] Only this native compatibility checkpoint closes. Parent/full R6, public-client/formal review, IR/engine and Temporal obligations remain open.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
