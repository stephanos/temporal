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
The native batch now passes all 452 tests across 57 suites after two fixture corrections. CoreBindings conversion preserves lazy initialization and once-only metadata acquisition; the ordinary-assumption fixture uses its actual declaration spelling while the added characterization preserves strict retained-identity refusal.

Task: fn-173-reachable-exploration-drop-full-tables.17
Tier: session (jev-unavailable(no_key))
Base commit: 73dec37ec7f1b40c2ac30c86efb56bff98f1117c
Source commit: 2bf90a8ed717cf83a16ecd96f5dcd7aa9c36b389

Only CoreBindings.test.scala and ResidualSignature.test.scala changed. The conversion case pins zero init reads through construction and repeated collection, two explicit false init results with one source read, and subsequent collection without another read. Callback counts stay zero and refinement/operand acquisition stays one. The ordinary fairness test preserves exact fixture.fair/fair metadata, ordered duplicates and raw empty name, and adds explicit empty-segment pins. The new case checks inferred spelling through empty/ordinary copies, absent native identity, incompatible retention refusal and exact qualified identity after matching retention.

baseline: red (unfiltered native tests failed pre-edit at 8bf89c02b60df29499afbb2bf503eeab0f6f13e9; 451 executed across 57 suites, 449 passed, two failed). The conductor explicitly authorized this fixture reconciliation from the recorded red baseline. Only metadata changed between that run and admission. Baseline logs remain .flow/tmp/native-dsl-batch-8bf89c02b6-compile.log (rc0, terminal session 34778) and .flow/tmp/native-dsl-batch-8bf89c02b6-tests.log (rc1, terminal session 90641). Baseline identity is persisted at .flow/tmp/fn17317-native-baseline and .flow/tmp/base_commit.

Frozen-source verification completed with terminal exit codes observed once per command. Strict compilation passed rc0 (session 3871); unfiltered native tests passed rc0 with 452 tests across 57 suites (session 88270); scoped lint RuleSuites passed rc0 with 29 tests across three suites (session 76328); changed-file formatting passed rc0; git diff --check passed rc0. Logs, commands, counts, terminal handles and baseline hashes are in .flow/tmp/fn17317-native-run-evidence.json. Compile and unittest green receipts were written for source commit 2bf90a8e. gate classify returned FULL rc1 for the Scala diff, and all task-defined full native commands ran without exclusions.

The added identity case characterizes intended existing behavior and earns no failing-production regression claim. The two original failures supply the fixture red-to-green evidence. Production behavior, retained source identity rules and missing-identity refusals remain unchanged. Existing in-process synthetic carrier tests remain discovered and earn no compiler capture/export proof credit.

Defect route:
- prior fixes: reviewed recent affected-file history and conductor's exclusive lane assignment; no competing fix identified. Memory search found no directly relevant fixture entry. Open PR and issue checks were unchecked because gh returned HTTP 401 Bad credentials.
- diagnosis: the recorded CoreBindings failure obtains starts=0 against obsolete expected 1; Built takes starts by name and memoizes initialStates. The assumption failure refuses absent identity because inferred spelling value disagreed with retained fair. Corrected native cases pass without production edits.
- introduced by: 7d6e1443453bf0be39764c6ec4d6f523cb5145c7 made bare assume infer declaration spelling; lazy-init behavior predates this checkpoint. Bisect skipped because the task reconciles deliberately changed contracts and names no known-good revision of those contracts.
- base: 449 passed and two fixture failures at 8bf89c02b60df29499afbb2bf503eeab0f6f13e9; head: all 452 passed at source commit 2bf90a8ed717cf83a16ecd96f5dcd7aa9c36b389. A separate failing-test commit was unnecessary because both reproductions were already committed and recorded red before this task.
- live: no live surface; native library tests are the scoped evidence.

Only .17 closes. The conductor owns the remaining production boundary scan, full native formatting, fresh public-client batch and formal review. Parent R6/full IR, graph/Query, universal replacement, lifting, numeric, engine/Go/backend and Temporal obligations remain open. MILESTONES.md and retained sample fixtures/maps were not edited. No excluded gate, catalog walk, exporter/schema/Go/backend/regeneration or model/temporal read ran.

stage: impl-review - skipped(config: REVIEW_MODE=none; conductor owns formal batch review after native and public gates)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 2bf90a8ed717cf83a16ecd96f5dcd7aa9c36b389
- Tests: mise exec -- scala-cli compile --server=false --test model/framework/dsl model/framework/realize model/framework/samples, mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests, mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only 'framework.check.lint.*RuleSuite', mise exec -- scala-cli fmt --check --scalafmt-conf model/.scalafmt.conf model/framework/dsl/CoreBindings.test.scala model/framework/dsl/ResidualSignature.test.scala, git diff --check
- PRs: