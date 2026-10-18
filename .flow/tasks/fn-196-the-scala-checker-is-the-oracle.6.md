---
satisfies: [R8]
---
# fn-196-the-scala-checker-is-the-oracle.6 Bind native var-runtime records to their Model instance lifetime

## Description
Implements the runtime half of R8 after the committed measurement checkpoint. Remove the process-global ownership of each Vars runtime record while preserving native evaluation behavior. The discovery-index repair is a separate dependent task; this task does not claim checked-owner reclamation while discovery caches still retain declarations.

**Size:** M
**Files:** VarRuntime, Vars and focused runtime-lifetime tests.
**Touches:** `model/framework/checker/VarRuntime.scala`, `model/framework/dsl/Vars.scala`, `model/framework/checker/RuntimeLifetime.test.scala`, `model/framework/dsl/VarMachines.test.scala`, `model/framework/dsl/ParametricMachines.test.scala`

### Approach and behavior pin

- Read `model/framework/checker/VarRuntime.scala` and `model/framework/dsl/Vars.scala` together. Attach the runtime record to its existing owner lifetime or an equivalently complete private seam. Keep implementation in checker and declarations loadable without checker imports or eager callbacks. Do not add author-visible close/reset/dispose or cache-management APIs.
- Preserve exactly one initial snapshot/depth/accessor record for a live owner, nested and exceptional restoration, exact foreign-state refusal, independent instances, composition member scopes and synchronization. A global weak-key map with a strong Live value still leaks through captured State.owner; a weak Live that forgets the initial snapshot while the owner remains alive is also invalid.
- Extend the existing restoration/identity controls in VarMachines and ParametricMachines only where needed. Keep the new focused suite in checker; test concurrent same-owner access and independent owners without expanding the runtime's concurrency contract.
- Pin behavior against current focused runtime/parametric/composition assertions and R4's retained baseline. Reproduce initialized-only retention with constructed-only release controls, then show initialized-only owners collectible after all legitimate States/results/owners are released. Keep a State alive across collection pressure and verify its original owner/snapshot remains usable before releasing it. Use a bounded observed-GC protocol, not a single unqualified System.gc assertion.

### Investigation targets

Required: `model/framework/checker/VarRuntime.scala`, `model/framework/dsl/Vars.scala`, `model/framework/dsl/VarMachines.test.scala`, `model/framework/dsl/ParametricMachines.test.scala`, `model/framework/dsl/PlainComposition.test.scala`.

Research: `.flow/tmp/fn1963-prep/cache-lifetime-scope.md`, `cache-plan-repo.md`, `cache-plan-memory.md`, `cache-plan-gaps.md`. Current immutable before receipt is `.flow/tmp/fn1963/public-observations-v2/retention-64.log`; the finished .3 handover is authoritative for its final source and counts. The broad memory results contain only a compiler-freshness caveat and historical caller-coverage lesson, no prior native leak fix.

### Execution boundary

Run after the prerequisite closure is committed on umpire. Use own focused baseline, RED/GREEN lifetime controls and fresh workspaces. Remain framework/native-only, review none under owner phase-batch deferral, tracker inactive, parent OPEN. Preserve all native664, class7, rule31, boundary66 and source173 proofs plus fresh declaration/core/plugin/sample realization and public-client source/separate-library gates when classification is FULL. Reuse the proven fn1945 recipe with actual classify/check/receipt commands. Every handle reaches terminal exit. No Temporal, broad GuideSourcesTest, real-Go SamplesSuite, IR/export/lifting/schema/backend/generated/regeneration or held Make route. Root alone edits MILESTONES. Scoped stage/commits only; no publishing/history rewrite.

### Quick command

```sh
timeout 600 env -u BASH_ENV mise exec -- scala-cli test --server=false model/framework/checker --require-tests --test-only framework.checker.CheckerTest
```

## Acceptance
- [ ] Initialized-only transient owners are collectible under an observed bounded collection protocol; constructed controls release, and intentionally retained States preserve their owner and initial snapshot.
- [ ] Nested/exceptional restoration, same-owner synchronization, independent parametric owners and foreign-state refusal retain their current exact behavior.
- [ ] Focused RED/GREEN and fresh affected gates pass with terminal raw evidence; no public disposal API or claim of checked-owner reclamation before discovery repair.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
