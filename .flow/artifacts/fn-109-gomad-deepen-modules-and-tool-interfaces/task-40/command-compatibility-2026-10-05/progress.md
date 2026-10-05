# Task 40 reviewed command compatibility source progress

The private Compatibility operation now preserves actual command causes through shared bounded capture and process-group cleanup. Default hostexec, Structured and Diagnostic behavior remain unchanged. This is reviewed source progress. Task 40 remains open.

## Corrected behavior

- Startup preserves lookup/context/cwd precedence and actual exec.Error/os.PathError objects.
- After acknowledged startup, caller cancellation retains the actual SIGKILL exec.ExitError. A finite 15-minute default watchdog has its own error type.
- Infrastructure failures precede stdout overflow, stderr overflow, watchdog and raw command outcomes. Failed or truncated stdout is unavailable to decoders; complete bounded stderr survives.
- Corrective C1 controls reproduce and fix stdin copying errors being demoted below real output overflow, including a reader returning context.Canceled with a live caller context. The executor retains a true context outcome only when its original per-command Cancel succeeded and Wait returns the exact caller error. Wait/channel synchronization protects that local provenance value.
- Corrective C2 reproduces and fixes a late watchdog relabeling an already reaped exit-7 leader while stdin copying remains blocked. ErrProcessDone clears false watchdog attribution. Actual watchdog/descendant termination controls remain passing.

## Current evidence

BASE is 29e12b5e473de0aa4e7b3df4db8626d421ed8891. [Frozen source/tool bindings](corrected/source-freeze.sha256), [worker command receipts](corrected/evidence.json) and [conductor receipts](conductor/receipts.json) identify the final five-file candidate and commands.

| Gate | Result |
| --- | --- |
| Conductor focused tests | Exit 0; 56 passed test results, no failures or skips |
| Two corrective regressions repeated 25 times | Exit 0 |
| Architecture, errortype and formatting | Exit 0; formatting has no output |
| Worker corrective race, generator validation and six applicable consumer groups | Exit 0 |
| Actual unfiltered lint | Exit 1; four inherited unchecked pipe Close defers and one inherited test-helper sleep |

The original and corrective RED logs remain in [initial](initial/summary.md) and [corrected](corrected/summary.md). Initial lint locations move with admitted inserted lines; final unfiltered output is byte-identical to the corrective baseline. No lint suppression, filtering or rule/pin change ran. Conductor checks prove the complete original test-function suffixes are byte-identical to BASE, protected module/config/helper/pin paths are unchanged and git diff --check passes.

### Correctness axis

[Fresh corrected-source report](corrected/correctness-review.md). SOURCE_PROGRESS_COMMIT_ONLY. Introduced findings 0; worst severity none.

### Standards axis

[Fresh corrected-source report](corrected/standards-review.md). SOURCE_PROGRESS_COMMIT_ONLY. Introduced findings 0; worst severity none.

Both complete reports are retained verbatim. Requested writer and reviewers use the same gpt-6.1-sol/high family. The harness supplied no actual-model metadata.

## Outstanding acceptance

Actual unfiltered lint is red. The predecessor/matched-first-baseline, complete patched-toolchain/full/default/functional/smoke/affected-native gates, exact-input full adapter regeneration, formal implementation review and native Darwin qualification remain unproved. This host is developmental linux/arm64, with no patched toolchain. The rare successful-cancellation/exit-0 interleaving is supported by pinned standard-library inspection and synchronization reasoning, not a claimed deterministic reproduction. Projection seam controls supply no real OS cleanup-fault injection.

Task 9 retains helper integration, its task 8 dependency, admitted listing-capacity measurements, nonempty source selection and pin proof. fn-113/task 21 retain publication and aggregate R10/R18/R19 qualification. The unchanged helper's historical 29 ordinary controls do not establish those requirements. Linux execution remains deferred under fn-128 and does not block source acceptance.

stage: source-review - ran
stage: impl-review - skipped(policy: original source-owned gates red or unproved)
stage: plan-sync - skipped(empty: no completed task)
stage: completion-review - skipped(policy: task remains open)
stage: QA - skipped(policy: no live application surface; real process controls retained)
Tracker sync: n/a (bridge inactive)
