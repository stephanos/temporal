---
satisfies: [R3, R4, R6, R11]
---
# fn-123-declare-faults-as-the-environments.4 Task-queue providers converted: derived crash, overriding crashes with fault-overridden lint, derived storage-loss assumption

## Description
Convert the shared task queue (R3, R4, R6, R11): `TaskQueueSystem` declares its durability and a derived crash, `crashDetail` goes, `ForgetfulQueue` and `VolatileQueue` rebind authored crashes over the derived one with lint kind `fault-overridden` and accepted reasons, and the storage-loss bindings derive their assumption in place of `storageLossAssumed`. It is one task because these files share one regeneration and one diff classification.

**Size:** M
**Files:** `model/temporal/foundations/taskqueue/system/System.scala`, `model/temporal/foundations/taskqueue/product/Product.scala`, `model/temporal/foundations/taskqueue/TaskQueue.scala`, `model/irgen/Declarations.scala` (derived assumption, refusal of a duplicate), `tools/umpire/lint/{lint,kinds}.go` (+ tests), `model/ir/*.lint.json` (acceptances), regenerated `model/ir/**`, `model/cases/**`
**Touches:** [model/temporal/foundations/taskqueue/**, model/irgen/**, tools/umpire/lint/**, model/ir/**, model/cases/**]
**Order:** After task 3. Re-read the targets first; fn-139.7 rewrote these rules into `when` blocks after the targets were verified.

### Approach
- After task .2 has classified both existing hole-bearing Disk fixtures, activate mandatory durability on all crash bindings atomically with this complete queue conversion. Enumerate every authored/derived crash binding independently; prove an unclassified authored binding now refuses, including a negative fixture not using `crashes`. Preserve inherited-classification removal for crash-free restrictions, and retain refusal for explicit crash-free classifications. No staging exception survives this task.
- `TaskQueueSystem`: classify `custody` as `inMemory(invoked -> history, reserved -> history)`, `polled` as `ephemeral(resetTo = false)`, `delivered` as `durable`, and replace the crash rule (`System.scala:153`) and `crashDetail` (`:115-120`) with `crashes(fault.crash, QueueFact.crashed)`. Run task 3's table test against the regenerated IR as well as the fixture.
- `ForgetfulQueue` (`:264`) and `VolatileQueue` (`:272`) keep `rebind(fault.crash ~> …)`. The IR records each as authored over a derived crash. Their counterexamples and refinement targets stay the same.
- Lint: add `fault-overridden` beside the kinds in `lint.go:29-87`, reported per binding that overrides a derived crash. Accept both with their reasons in the `.lint.json` the gate reads (`lint/accept.go`). Check whether the acceptance file needs a version bump and say so in the done summary.
- Storage loss: binding a `storageLoss` fault derives one assumption per fault, named after it and appended after authored assumptions. Delete `storageLossAssumed` (`TaskQueue.scala:115`) and the `.assuming(storageLossAssumed)` calls (`System.scala:239`, `Product.scala:89`). A hand-written assumption that duplicates a derived one is refused at its line (extend the twice-assumed check at `Declarations.scala:906-909`). `queueOpaque` keeps its IR order before the derived one.
- Check whether the assumption id feeds any Query Definition ID or Case fingerprint; if it does, stop for the owner (spec Open Questions).
- Regenerate in an isolated scratch checkout under the real `/tmp/umpire-heavy-gates.lock`, then classify every changed line against fn-149.5's final grouped baseline and composed fn-140/fn-155 mapping of `model/ir`, `model/cases` and the lift goldens against the recorded deltas (spec Edge Cases, "Behavior is frozen"). The pinned paths through `queue_crash` and `queue_ackLoss` must not change.

### Investigation targets
**Required** (read before coding):
- `model/temporal/foundations/taskqueue/system/System.scala:100-160, 225-276` - effects, rules, derived providers
- `model/temporal/foundations/taskqueue/product/Product.scala:65-92` - `queueOpaque` and the storage-loss interface
- `tools/umpire/lint/lint.go:29-87`, `tools/umpire/lint/kinds.go:95-155` - kinds and the fault-as-performable test
- `tools/umpire/lint/accept.go:22-120` - accepted findings

**Optional** (reference as needed):
- `model/irgen/Declarations.scala:900-912` - assumption checks
- `model/ir/activity-standalone-record.json` - where the queue machines lift

### Key context
- Memory `joined-composition-keys-must-be-checked-2026-09-28`: the derived row keyed under a composition member must keep the joined keys injective.
- Memory `behavior-neutral-refactors-must-not-2026-09-04`: the conversion must not tighten validation the Models relied on.

## Acceptance
- [ ] `crashDetail` and the authored crash rule are gone from `TaskQueueSystem`, and task 3's table test passes against the regenerated queue. Universal R2 classification is active for every crash binding, existing Disk hole behavior survives, and crash-free inherited restrictions stay valid.
- [ ] `ForgetfulQueue` and `VolatileQueue` give the same counterexamples. Lint reports `fault-overridden` for each, and the gate fails if either acceptance is removed.
- [ ] `storageLossAssumed` is gone. The system and the product share one derived assumption named `storageLoss`, after `queueOpaque` in IR order. A refusal fixture shows a duplicate hand-written assumption refused at its line.
- [ ] Against fn-149.5's final grouped baseline, the regenerated diff holds only the recorded fault deltas, and every changed line is classified in the done summary. Query answers and Case bytes are unchanged.
- [ ] `make umpire-check-model`, `make umpire-check-cases`, `make umpire-check-lint` and `go test -tags test_dep ./tools/umpire/lint/...` pass. Focused scratch proofs run here; production publication and complete gate evidence are linked from fn-123.8, without duplicate full-suite runs or early task closure.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
