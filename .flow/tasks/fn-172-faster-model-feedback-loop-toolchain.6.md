---
satisfies: [R6]
---
# fn-172-faster-model-feedback-loop-toolchain.6 check.Check interprets each Model once: share or remove the replay binding and cut the GC share, with byte-identical receipts

## Description
Share the checker binding when witness replay uses the identical Model, retaining separate binding for deliberately different replay Models. Profile and remove redundant interpretation/allocation within the existing interpreter and replay surface, without changing search semantics, IR, IDs or fingerprints.

**Touches:** tools/umpire/check/**, tools/umpire/interp/**, tools/umpire/internal/engine/**, .flow/tasks/fn-172-faster-model-feedback-loop-toolchain.6.*

### Investigation targets
Required: tools/umpire/check/checking.go, tools/umpire/check/claims.go, tools/umpire/check/replay_test.go, tools/umpire/interp/machine.go, tools/umpire/internal/engine/table.go.

### Quick commands
- go test -tags test_dep ./tools/umpire/check -run '^(TestReplaySharesTheInterpretationOfTheSameModel|TestPropertyRowWalkKeepsItsIndependentReplayBinding|TestComposedProgressWitnessReconstructsTypedMemberStates|TestComposedProgressCountsASynchronizedStepOnceAndReplays|TestTheCounterModelIsAdmitted)$' -count=1
- make lint-code-fast GOLANGCI_LINT_BASE_REV=9792def9fcf7d6110a54e93adebf5c3a09af32df

Full model gate and complete check/interp package suite remain conductor-owned at final spec closure. The attempted broad pre-edit suite has inherited missing model/irgen fixture paths and was terminated under resource pressure; it is red/inconclusive, not waived as passing. Focused existing synthetic tests passed before implementation. Use preserved six sample IR artifacts to measure checker wall time and allocation before/after, compare complete Report bytes on representative samples, and require all six expected Query/progress verdicts after the fix. Inherited old Bank OOM and terminated capped runs are red/inconclusive observations, not green evidence.
## Acceptance
- [ ] R6: Check shares its interpreted Model with replay; a distinct replay Model still binds independently and rejects stale witnesses.
- [ ] Reduce measured checker allocation/GC without changing reachable exploration or IR shape.
- [ ] Complete Report bytes match the pre-edit reports on representative sample IR; all six samples produce unchanged expected Query/progress verdicts.
- [ ] Record before/after checker wall time and memory/allocation measured on the same environment and artifacts; preserve all IR bytes, fingerprints, Definition IDs and Case identities.
- [ ] Focused Go tests and changed-package lint pass; independent configured implementation review reaches SHIP before done.
## Done summary
Model authors get faster checker feedback with unchanged receipts. `Check` shares its interpretation with replay of the same Model, honors an explicitly supplied replay binding, and binds a distinct replay Model independently.

Tier: session (jev-unavailable(no_key)); project implementer gpt-6.1-sol medium

stage: impl-review - ran [2026-10-13T18:02:47Z..2026-10-13T18:12:32Z] (model: gpt-6.1-sol high; same family as writer)

R6 is implemented. The final rebuilt checker passed the existing six-sample verdict runner on the retained current IR artifacts, covering 25 Queries, six negative controls and three progress claims. Complete Kettle and TwoPhaseCommit Report JSON bytes match their pre-edit reports. No IR, generated Case, Definition ID or fingerprint encoding was edited.

Isolated measurements use the same retained IR and profiling harness, one process at a time with the same environment. Allocation and HeapAlloc include the same profiling instrumentation; HeapAlloc is a snapshot, not peak RSS.

| Sample | Check wall time before / after | Allocated bytes before / after | Mallocs before / after | GC cycles before / after |
| --- | --- | --- | --- | --- |
| Kettle | 22.616208 ms / 14.589416 ms | 21,364,672 / 15,153,768 | 103,913 / 68,607 | 5 / 4 |
| TwoPhaseCommit | 15.728571799 s / 9.254639588 s | 18,733,283,096 / 10,730,000,408 | 90,282,558 / 49,666,190 | 42 / 35 |

TwoPhaseCommit allocated 42.7% fewer bytes. Its sampled `gcDrain` work fell from 28.56 to 14.28 CPU seconds, and its share fell from 60.62% to 59.40%. HeapAlloc snapshots were 4,502,111,984 and 2,999,833,760 bytes. Bank completed after sharing in 44.572743355 seconds with four receipts; its old OOM and capped stand-down runs supply no comparable successful before timing. Bank still has substantial GC work.

Evidence is under `.flow/tmp/fn172-r6/`: `Kettle-isolated-{before,after}.{log,json}`, `TwoPhaseCommit-{before,after}.{log,json,cpu,heap}`, `Bank-after.{log,json,cpu,heap}`, `all-samples-final.log`, `verify-tests.log`, `verify-lint.log`, and the measurement/checker binaries. The canonical review receipt is `/tmp/impl-review-receipt-8f37faba39e2-fn-172-faster-model-feedback-loop-toolchain.6.json`; fanout sidecars are `.flow/review-fanout/75527949acc548babc14d81d2bfef687/`.

Baseline and gates:
- Focused existing synthetic replay/progress/counter tests passed before implementation. The final five-test selection passed after review; changed-package lint reported zero issues.
- `go test -tags test_dep ./tools/umpire/check ./tools/umpire/interp` was red/inconclusive before the fix. The interpreter suite failed on inherited missing `model/irgen/testdata/lifts/expected` paths. The checker also reported an inherited declaration-fixture failure and was deliberately terminated under resource pressure after 461.783 seconds. `baseline-tests.log` records both. A trial helper-root correction exposed further inherited Step-type drift and was reverted completely. No existing assertion or fixture was weakened.
- The earlier concurrent Voting diagnostic timed out at 120 seconds without a report and remains inconclusive. It supplies no performance baseline.
- `gate classify` selected FULL. The scoped unittest receipt was refused with `NO_RECEIPT: worktree dirty outside the ignore set (MILESTONES.md) - receipt not warrantable`; test and lint exit codes are green. No green receipt or skip is claimed.
- The complete package suite and spec-end model gate remain outstanding for the conductor. This task does not claim either passed.

The initial same-Model regression failed before the sharing fix (`replay-red.log`). Review then found that the shortcut bypassed `newPropertyRowWalk`'s supplied binding. `TestPropertyRowWalkKeepsItsIndependentReplayBinding` failed on that shortcut (`explicit-binding-red.log`) and passed after the ordering fix. The helper now asserts the binding replay actually uses; existing synthetic composed-progress tests still reject stale witnesses against a distinct Model. The review lesson was captured as `.flow/memory/bug/integration/preserve-explicit-replay-bindings-when-2026-10-13.md`.

Defect route:
- prior fixes: related `w-perf` / `w-perf-dev` work changes row copying and admission, while retaining fresh witness interpretation; no competing same-Model replay-sharing fix found. Recent checker history and memory were inspected. GitHub PR/issue checks were unchecked after HTTP 401 authentication failure.
- diagnosis: the Kettle allocation profile includes `again -> bind`; the same-Model regression confirms a second binding. Retained-IR measurements isolate the Go stage from Scala toolchain overhead. Sharing preserves complete representative reports and removes measured allocation and GC work.
- introduced by: skipped; no known-good revision of the requested sharing behavior was supplied.
- base: same-Model replay regression fails at the pre-fix implementation; explicit-binding regression fails at the first sharing implementation. head: both regressions, scoped replay/progress tests, lint and six-sample verdict checks pass.
- live: no live application surface; the checker library and preserved IR runner provide the proof.

Owner `MILESTONES.md`, `model/design/`, the three prepared R1 Scala test edits and sibling task planning files remain outside task staging. All task-started commands exited. The pre-existing Bloop daemon was preserved.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 9ee68a467bdfc6ba5ba56911de99efe9c03609c8, ccbc7442f5fe6da11df3641b991ebe372d9ed0b4, 22dda540a4743901604a3775dc6cd4318b8a4dca
- Tests: baseline green: go test -tags test_dep ./tools/umpire/check -run '^(TestComposedProgressWitnessReconstructsTypedMemberStates|TestComposedProgressCountsASynchronizedStepOnceAndReplays|TestTheCounterModelIsAdmitted)$' -count=1, baseline red/inconclusive: go test -tags test_dep ./tools/umpire/check ./tools/umpire/interp (inherited missing model/irgen fixtures; checker terminated at resource-pressure stand-down after 461.783s; baseline-tests.log), regression red: go test -tags test_dep ./tools/umpire/check -run '^TestReplaySharesTheInterpretationOfTheSameModel$' -count=1 (replay-red.log), review regression red: go test -tags test_dep ./tools/umpire/check -run '^TestPropertyRowWalkKeepsItsIndependentReplayBinding$' -count=1 (explicit-binding-red.log), go test -tags test_dep ./tools/umpire/check -run '^(TestReplaySharesTheInterpretationOfTheSameModel|TestPropertyRowWalkKeepsItsIndependentReplayBinding|TestComposedProgressWitnessReconstructsTypedMemberStates|TestComposedProgressCountsASynchronizedStepOnceAndReplays|TestTheCounterModelIsAdmitted)$' -count=1, make lint-code-fast GOLANGCI_LINT_BASE_REV=9792def9fcf7d6110a54e93adebf5c3a09af32df, .flow/tmp/fn172-r6/check-samples-final .flow/tmp/fn172-r1/before-bounded/Kettle.check.json .flow/tmp/fn172-r1/before-bounded/Bank.check.json .flow/tmp/fn172-r1/before-bounded/TwoPhaseCommit.check.json .flow/tmp/fn172-r1/before-bounded/TwoLayeredCache.check.json .flow/tmp/fn172-r1/before-bounded/ReliableBroadcast.check.json .flow/tmp/fn172-r1/before-bounded/Voting.check.json, cmp .flow/tmp/fn172-r6/Kettle-isolated-before.json .flow/tmp/fn172-r6/Kettle-isolated-after.json, cmp .flow/tmp/fn172-r6/TwoPhaseCommit-before.json .flow/tmp/fn172-r6/TwoPhaseCommit-after.json, isolated Kettle profiling: Check 22.616208ms -> 14.589416ms; allocation 21364672 -> 15153768 bytes; GC 5 -> 4, isolated TwoPhaseCommit profiling: Check 15.728571799s -> 9.254639588s; allocation 18733283096 -> 10730000408 bytes; gcDrain 28.56 -> 14.28 CPU seconds, NO_RECEIPT: worktree dirty outside the ignore set (MILESTONES.md) - receipt not warrantable, full package suite and spec-end model gate outstanding; no green or skipped full-gate claim
- PRs: