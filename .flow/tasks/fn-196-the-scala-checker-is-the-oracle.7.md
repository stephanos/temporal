---
satisfies: [R8]
---
# fn-196-the-scala-checker-is-the-oracle.7 Release native discovery metadata with its declarations and remeasure retention

## Description
Implements the declaration-index half of R8 after the runtime-lifetime repair. Repair the coordinated discovery indexes and prove combined initialized/checked-owner reclamation. The exact mechanisms stay private; preserve the public checker APIs and all existing author semantics.

**Size:** M
**Files:** NativeCheck discovery indexes, a minimal private lifetime helper or declaration-local anchors if needed, focused discovery-lifetime tests, and the measurement after-receipt note.
**Touches:** `model/framework/checker/NativeCheck.scala`, `model/framework/checker/DeclarationLifetime.scala`, `model/framework/checker/DiscoveryLifetime.test.scala`, `model/framework/checker/ParametricActionNames.test.scala`, `model/framework/dsl/Action.scala`, `model/framework/dsl/Channel.scala`, `model/framework/dsl/Assume.scala`, `model/design/scala-checker-measurement*`

Only use the conditional private helper/declaration files required by the selected mechanism. Further production paths require conductor authorization before editing.

### Approach and behavior pin

- Audit spelled, channels, holes and claims together at `NativeCheck.scala:975` and their readers/writers. Use reference identity and stable first-discovered spellings. Retain static discovery and current own-versus-foreign actor/internal discovery rules. Avoid a generic configurable cache framework.
- Weak keys alone do not repair channels because a Channel owns its deliver/lose ActionDecl keys. Preserve executable channel behavior when only a legitimate live action/result retains it; do not trade collection for lost metadata. Claim callbacks can retain owners. Hole and claim names must follow declaration lifetime too. Preserve module-scan recovery and fallback naming; classloader byte/declaration caches remain out of scope unless an actual instance path contradicts the current boundary.
- Use `ParametricActionNames.test.scala` for exact own/foreign aliases, copied declarations and callback-free discovery. Add focused collection controls with identity-equal-but-distinct keys where applicable, retained results/actions/states across GC and another instance's check, channel delivery/loss and backreferences, names for holes/claims/monitors/progress, and static/derived/composed controls. Assert actual initial/effect/cut/unknown/refuted answers remain unchanged.
- Pin behavior with existing naming, native execution and six-sample answer/negative-control tests plus the immutable R4 receipt. After both repairs, rerun the original constructed/initialized/checked transient batches with fresh build/artifact/output paths. Show release after legitimate owners/results are dropped, with no harness cache clearing. Keep baseline raw logs immutable, record before/after source versions, and distinguish repeated-instance retention from single-graph live working set. Any remeasured times/heaps use the original public verifyAll timing/GC boundaries and raw evidence. No speedup or Go-deletion-go claim follows merely from reclaiming owners.
- Append a scoped after-repair section and runnable recipe to the measurement note, retaining its original measurement and decision. Oracle-direction docs remain the dependent .5 task.

### Investigation targets

Required: `model/framework/checker/NativeCheck.scala`, `model/framework/checker/ParametricActionNames.test.scala`, `model/framework/dsl/Action.scala`, `model/framework/dsl/Channel.scala`, `model/framework/dsl/Assume.scala`, `model/framework/checker/ClaimEvaluation.scala`, `model/design/scala-checker-measurement.scala`.

Research: `.flow/tmp/fn1963-prep/cache-lifetime-scope.md`, `cache-plan-repo.md`, `cache-plan-specs.md`, `cache-plan-gaps.md`. Preserve the instance-section cache bypass in ClaimEvaluation. Initialized-only and checked-owner reclamation are combined acceptance here; either repair alone leaves another root.

### Execution boundary

Run after the prerequisite closure is committed on umpire. Use own focused baseline, RED/GREEN lifetime controls and fresh workspaces. Remain framework/native-only, review none under owner phase-batch deferral, tracker inactive, parent OPEN. Preserve all native664, class7, rule31, boundary66 and source173 proofs plus fresh declaration/core/plugin/sample realization and public-client source/separate-library gates when classification is FULL. Reuse the proven fn1945 recipe with actual classify/check/receipt commands. Every handle reaches terminal exit. No Temporal, broad GuideSourcesTest, real-Go SamplesSuite, IR/export/lifting/schema/backend/generated/regeneration or held Make route. Root alone edits MILESTONES. Scoped stage/commits only; no publishing/history rewrite.

### Quick command

```sh
timeout 600 env -u BASH_ENV mise exec -- scala-cli test --server=false model/framework/checker --require-tests --test-only checkercontrol.CheckerTest
```
## Acceptance
- [ ] Discovery indexes release discarded declarations and owners while preserving reference identity, stable names and static/derived/composed/channel behavior for legitimate live results/actions/states.
- [ ] Channel value-to-key backreferences, claim/hole ownership, foreign aliases and equal-but-distinct identity cases have regression controls; classloader metadata remains outside the instance-lifetime guarantee.
- [ ] After both repairs, fresh constructed/initialized/checked retention controls demonstrate reclamation without clearing caches; raw before receipts remain immutable and the note distinguishes instance retention from single-graph memory.
- [ ] Fresh affected native/source/library/realization gates pass with unchanged claims, bounds, negative controls, cuts and unknown/refinement standings.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
