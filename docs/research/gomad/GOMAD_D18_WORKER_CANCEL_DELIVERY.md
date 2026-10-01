# Gomad D18: the worker cancel command after a workflow timeout

Assessment date: 2026-09-30.

`TestWorkerCommandsTaskSuite/TestDispatchCancelOnWorkflowTimeout` fails under Gomad because the
server never creates the cancel command, so nothing arrives late. Under the default `strict` tick
the workflow start and the activity schedule share one virtual instant. The server caps the
activity's timeouts at the 3 s workflow run timeout, so the activity deadline equals the run
expiration and two timer tasks become due at the same timestamp. When the activity's timer task
runs first, the server records the activity as timed out, the workflow then closes with no
in-flight activity, and no cancel command exists to deliver. The test then polls an empty control
queue until its context ends at 120 s. The seed selects which timer task runs first. Seed 11
fails and seed 17 passes when the whole suite runs. The roles swap when the test runs alone. 9 of
16 seeds fail.

No test timeout budget and no step of the delivery path causes the failure. A longer timeout
cannot help. Natively the activity is scheduled 5 to 49 ms after the workflow starts, its deadline
is later than the run expiration by that margin, and the server creates no activity timer at all.

Three changes each remove the failure on every seed tried. The recommended one is to run the suite
under `clock_tick: forward`, owned by the Gomad qualification configuration. A test change and a
server change are recorded as alternatives with their owners. Nothing in this report is
implemented or fixed. The skip stays, and the correction is open work in fn-105 until a decision
selects it and its verification passes.

This is the receipt for
[D18](../../../.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.18.md) (fn-105 R18).

## Scope and identities

| Item | Value |
| --- | --- |
| Repository revision | `7ef3800a2c` plus the shared uncommitted working tree |
| Test source | `tests/worker_commands_task_test.go`, SHA-256 `13a3b0d4...7ce195` |
| Toolchain | go1.27.1, build key `8d28bd4486f0b6300e8d25efd4caf8cb6ccbf000e96dbd26b1d8f53bf5f251bc` |
| Runner | `tools/gomad3/.bin/gomad`, SHA-256 `4dce3aed...db5222` |
| Native Go | go1.27.0 darwin/arm64 |
| Host | darwin/arm64, macOS 26.6.2 (25G83) |
| linux/amd64 | Not available. No linux run exists for this report. |

Source references are `file:line` in the repository at the revision above. Evidence files are
under `.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/` with the prefix `fn105-d18-`;
[`fn105-d18-evidence.json`](../../../.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d18-evidence.json)
indexes them and records every command. No tracked product, test, runtime, or manifest file was
edited. The skip was lifted only through manifest copies in a scratch directory, and the two code
variants ran in a throwaway copy of the source tree that was deleted afterwards.

## Reproduction

Gomad single-suite and single-test runs use `gomad explore` with the qualification manifest's
build tags, closure mode, and schema mount (wrapper `fn105-d18-run.sh`). The suite pattern is
`^TestWorkerCommandsTaskSuite$` and the leaf pattern adds
`/^TestDispatchCancelOnWorkflowTimeout$`. Both run with `-test.parallel=8 -test.v`.

| Run | Tick | Seeds | Outcome of the timeout test |
| --- | --- | --- | --- |
| Suite, two runs of both seeds and a third of seed 11 | strict | 11, 17 | Seed 11 fails at 120.00 s virtual in all three runs. Seed 17 passes at 3.00 s in both. |
| Leaf alone, two runs | strict | 11, 17 | Seed 11 passes. Seed 17 fails at 120.00 s. Both runs agree. |
| Leaf alone | strict | 1 to 16 | 7 pass, 9 fail |
| Suite | forward | 11, 17 | Both pass |
| Leaf alone | forward | 1 to 64 | 64 pass |
| `qualify-set`, skip lifted | strict | 11, 17 | Seed 11 `target_failure`, replayed with a matching failure. Seed 17 `qualified`. |
| `qualify-set`, skip lifted | forward | 11, 17 | Both `qualified` at repeat 2 |
| `qualify-set`, skip lifted, traced | forward | 11, 17 | Both `qualified` with exact choice-tape replay |
| Native leaf, `-count=5` | wall clock | none | 5 of 5 pass in 3.02 to 3.12 s |
| Native suite, `-count=3` | wall clock | none | 21 of 21 subtest runs pass |

Every Gomad run in the first five rows that retained its timelines carries the same signature
(`fn105-d18-gomad-runs.txt`; the first suite run and the seed-11-only run are recorded by outcome in
the evidence index). Each failing seed processed the `ActivityTimeoutTask` before the
`WorkflowRunTimeoutTask` and created no `WorkerCommands` task. Each passing seed processed the
`WorkflowRunTimeoutTask` first and created one. No run hit a wall watchdog, an infrastructure
error, or a replay divergence.

The failing seed changes with the test selection, so the failure belongs to the schedule a seed
produces and not to seed 11. The recorded classification "seed 11 fails, seed 17 passes" holds only
for the whole-suite selection.

## Trace

The server logs every queue task at debug level, and the test log carries those lines. The two
timelines below are filtered from the suite runs
(`fn105-d18-gomad-suite-strict-seed11-timeline.txt` and `...-seed17-timeline.txt`). Times are
virtual seconds since the target's start.

| Virtual time | Step | Seed 11 (fails) | Seed 17 (passes) |
| --- | --- | --- | --- |
| 0.000 | Workflow starts with a 3 s execution timeout | `WorkflowRunTimeout` timer for 3.001 | `WorkflowRunTimeout` timer for 3.001 |
| 0.000 | First workflow task schedules the activity | `ActivityTimeout` timer for 3.001, `TransferActivityTask` | `ActivityTimeout` timer for 3.001, `TransferActivityTask` |
| 0.000 | Test polls the activity task, then starts polling the control queue | First `PollNexusTaskQueue` | First `PollNexusTaskQueue` |
| 3.002 | First timer task processed | `ActivityTimeoutTask` (ScheduleToClose, event 5) | `WorkflowRunTimeoutTask` |
| 3.002 | Its effect | Activity closed as timed out, workflow task scheduled as event 8 | Workflow closed, `WorkerCommands` task 1048597 created |
| 3.002 | Second timer task processed | `WorkflowRunTimeoutTask` closes the workflow. Close tasks are `DeleteHistoryEvent`, `TransferCloseExecution`, `VisibilityCloseExecution`. No `WorkerCommands` task. | `ActivityTimeoutTask` finds the workflow closed |
| 3.002 | Outbound queue | Nothing to dispatch | `WorkerCommandsTask` processed, the waiting poll receives the command, the test passes |
| 4.1 to 114.8 | Control-queue polls | 28 further polls, one every 4.1 s, each returns empty | |
| 118.9 to 119.9 | Control-queue polls | 11 polls refused with `Context timeout is too short` | |
| 120.0 | End | Await and test context expire together | |

The steps the task names resolve as follows.

- **Workflow timeout.** Fires on time on both seeds, at the first virtual instant after its
  deadline.
- **Cancel command generation.** This is where the seeds differ. `GenerateWorkflowCloseTasks` ends
  with `GenerateActivityCancelCommandsForClose` (`service/history/workflow/task_generator.go:274`),
  which builds one command per pending activity that has a control queue and a started clock
  (`service/history/workflow/mutable_state_impl.go:4789-4860`). On seed 11 the activity is no longer
  pending, the loop adds nothing, and no task is generated.
- **Persistence and transfer.** On seed 17 the `WorkerCommandsTask` is an outbound-queue task with
  an immediate key (`service/history/tasks/worker_commands_task.go:29-51`). The server assigns its
  key and processes it in the same virtual instant, 3.002.
- **Control-queue delivery and polling.** The test's poll has waited in matching since 0.000 and
  receives the dispatch at 3.002. On seed 11 the polls are present for the whole run and find
  nothing, because no dispatch is ever attempted for this queue.

The seed-11 log also holds six lines `No worker polling control queue, dropping command` at 4.000
(`fn105-d18-gomad-suite-strict-seed11-sibling-no-poller.txt`, from a third seed-11 suite run that
ended in the same `target_failure`). Each names the control queue of one of the six sibling tests,
which take their Nexus task and never answer it. None names this test's queue.

## Cause

Three facts combine.

1. **The server caps the activity's timeouts at the run timeout.** The test asks for 5 minute
   ScheduleToClose and StartToClose timeouts so that "the activity is still in-flight when the
   workflow times out" (`tests/worker_commands_task_test.go:433-435`). The command handler passes
   the workflow run timeout into validation
   (`service/history/api/respondworkflowtaskcompleted/workflow_task_completed_handler.go:495-500`),
   and validation reduces every activity timeout above it to the run timeout
   (`chasm/lib/activity/validator.go:197-212`). The run timeout here is 3 s, from
   `WorkflowExecutionTimeout: 3s` (`tests/worker_commands_task_test.go:416`). The activity's
   ScheduleToClose deadline is therefore its schedule time plus 3 s
   (`service/history/workflow/timer_sequence.go:331-352`), and the run expiration is the workflow
   start time plus 3 s (`service/history/workflow/mutable_state_impl.go:3183-3194`). The activity
   deadline exceeds the run expiration by exactly the time between workflow start and activity
   schedule.
2. **An activity timer is skipped only when it is strictly later than the run expiration.**
   `CreateNextActivityTimer` returns without creating a task when
   `firstTimerTask.Timestamp.After(workflowRunExpirationTime)`
   (`service/history/workflow/timer_sequence.go:127-132`). An equal deadline passes the guard and
   produces an `ActivityTimeoutTask` with the same visibility timestamp as the
   `WorkflowRunTimeoutTask`.
3. **Under the strict tick the two times are equal.** The
   [contract](../../../tools/gomad3/README.md#contract) states that every `time.Now` within one
   busy stretch returns the same instant and that runnable work is never skipped to deliver a
   future timer. Start, first workflow task, and activity schedule run back to back with no idle
   period between them, so they share one instant. Both timer tasks are logged with
   `VisibilityTimestamp` 3.001.

Two timer tasks of one workflow are then due together, and the order in which they run is a
scheduling choice that the seed decides. The log shows the order is not the task-ID order. On
seed 11 the activity task (ID 1048592) ran before the run-timeout task (ID 1048581).

- **Run timeout first.** `executeWorkflowRunTimeoutTask` closes the workflow
  (`service/history/timer_queue_active_task_executor.go:658-733`) while the activity is pending, the
  close generates the cancel command, and the later activity task returns at the
  workflow-not-running check (`timer_queue_active_task_executor.go:225`).
- **Activity timeout first.** `executeActivityTimeoutTask` calls
  `processSingleActivityTimeoutTask`, which adds an `ActivityTaskTimedOut` event
  (`timer_queue_active_task_executor.go:368`), and then schedules a workflow task. The run timeout then
  closes a workflow that has no pending activity, and no command is generated.

Both orders are valid server behavior for two deadlines that expire at the same instant. In the
second order the server has already recorded the activity as timed out, so no in-flight activity is
left for the best-effort cancel command to cancel. This report claims no product defect.

## Native comparison

Natively the first workflow task takes real time. In the five native runs the activity was
scheduled 5, 5, 5, 40, and 49 ms after the workflow started
(`fn105-d18-native-leaf-count5-timeline.txt`), so its capped deadline lay that far after the run
expiration. The guard in fact 2 skipped the activity timer every time. The native logs contain no
`ActivityTimeout` task at all, where every strict Gomad run contains one. Only the run timeout can
close the activity natively, and the command is generated at the run expiration plus 1 to 2 ms.

Wall-clock latency therefore hides a timestamp tie. It hides no exhausted budget, because the
native test finishes in about 3.05 s and never approaches one.

## Interventions

Each intervention changes one of the three facts and was run on leaf seeds 1 to 16, where the
unchanged test fails on 9.

| Intervention | Fact changed | Where it ran | Result |
| --- | --- | --- | --- |
| `--clock-tick=forward` | 3: consecutive `time.Now` readings advance | Working tree, no source change | 16 of 16 pass, extended to 64 of 64. No `ActivityTimeout` task is created on any seed. |
| Variant B: the first workflow task starts a 1 s timer and the second schedules the activity (`fn105-d18-variant-b-test.diff.txt`) | 1: the activity is scheduled 1 s into the run | Throwaway copy, strict tick | 16 of 16 pass. No `ActivityTimeout` task is created. Native leaf passes 5 of 5. |
| Variant C: the guard skips the activity timer when its deadline is not before the run expiration (`fn105-d18-variant-c-server.diff.txt`) | 2: an equal deadline is skipped | Throwaway copy, strict tick | 16 of 16 pass |

All three leave the test's assertions untouched: one `ExecuteCommands` request on the control
queue, exactly one command, a `CancelActivity` command, and a task token equal to the activity's.

## Budgets

The failing run exhausts its budgets in a fixed order, and none of them is the cause.

| Budget | Source | What happens on seed 11 |
| --- | --- | --- |
| 90 s default test context | `common/testing/testcontext/context.go:17` | Never expires. `Awaitf` asks for the await timeout plus a 10 s reserve (`common/testing/await/require_ctx.go:44,87`), and `EnsureRemaining` extends a default context up to 2 minutes after its creation (`context.go:26`). The context ends at 120.0 s. The log line is `test exceeded timeout of 2m0s`. |
| 120 s `Awaitf` | `tests/worker_commands_task_test.go:459-471` | Capped at the context deadline, 120.0 s after the test started. It reports `not satisfied after 2m0s` with 41 attempts. |
| 5 s child poll context | `tests/worker_commands_task_test.go:460` | Accepted, with a warning on every poll because it is below the 10 s critical value (`common/constants.go:45`, `common/util.go:655`). Matching returns an empty response 1 s before the deadline (`service/matching/physical_task_queue_manager.go:42`, `matching_engine.go:3110`), so each poll lasts 4 s and, with the 100 ms await interval, one starts every 4.1 s: at 0.0, 4.1, and so on to 114.8. |
| 2 s server long-poll minimum | `common/constants.go:42`, `common/util.go:649`, called at `service/frontend/workflow_handler.go:6394` | The first budget to bite. The poll at 118.9 s has 1.1 s left and is refused, as are the ten that follow at 100 ms spacing. |
| 10 s await attempt timeout | `common/testing/await/config.go:21` | Never reached. Every attempt ends within 4 s. |

The budgets are consistent enough for the test to pass whenever a command exists. They are
inconsistent in two respects that do not affect the outcome.

- A 120 s `Awaitf` cannot fit under a 2 minute context ceiling together with its 10 s reserve. The
  await receives whatever remains of the 2 minutes, and when it runs out the test context has
  expired too. A test that calls `Awaitf` with 120 s on the default context has no time left after
  a timed-out await.
- The last 2 s of any context cannot carry a long poll, so the last 1.1 s of polling time is spent
  on refusals.

The skip's recorded reason attributes the failure to the 90 s context running below the 2 s
minimum while the await allows 120 s. The run shows the context extended to 120 s, 29 full polls
before the first refusal, and no command at any time. The refusals are the tail of a wait that
could not succeed. Extending any of these budgets changes how long the test waits and nothing else.

Under virtual time the exhaustion order is the 2 s minimum at 118.9 s, then the await and the
context together at 120.0 s. The first is reached only because the command was never created at
3.002 s.

## Corrections and owners

| Option | Change | Owner | Evidence here | Cost and risk |
| --- | --- | --- | --- | --- |
| A | Set `"clock_tick": "forward"` on `TestWorkerCommandsTaskSuite` in `tools/gomad3integration/qualification/tests.generator.json` and regenerate `tests.json`. Remove the `skip_subtests` entry only after the verification below passes. | Gomad qualification configuration (`tools/gomad3integration`, owner `gomad3`) | Suite qualifies on seeds 11 and 17 with the skip lifted, untraced at repeat 2 and traced with exact replay. Leaf passes on seeds 1 to 64. | No server or test change. Seven suites already run under `forward`. The forward tick shifts deadlines derived from `time.Now`, which the recorded D16 skip reason names as the cause of a client timeout in another suite. No run of this suite showed that effect. |
| B | Schedule the activity one second into the run, as in variant B | Owner of `tests/worker_commands_task_test.go` | Leaf passes on seeds 1 to 16 under the strict tick and natively 5 of 5 | Changes a Temporal test for a condition a wall clock does not produce. The milestones' constraint against test rewriting applies unless the test owner accepts it as making the test's stated precondition structural. The existing comment about the 5 minute timeout is inaccurate either way. |
| C | Skip the activity timer when its deadline equals the run expiration, as in variant C | History service owner, with the dedicated production review the milestones require | Leaf passes on seeds 1 to 16 under the strict tick | Changes production behavior at a boundary a wall clock does not reach. No unit test or wider suite ran against it. It follows the precedent of `fb63ac814f`, which ordered an activity's own equal-deadline timers in the same file. |

Recommendation: option A. The cause is a timestamp tie that exists only when no time passes between
two server transactions. `forward` is the tick policy this project already uses for timestamp ties,
it needs no change to Temporal code, and the evidence above covers both seeds with the skip lifted.
Options B and C stay recorded for the decision. Each is sufficient alone, and neither is needed
if A is accepted.

Proposed next action, as open work in fn-105 for a subsequent decision. The skip stays in the
tracked manifest through step 5.

1. Add `"clock_tick": "forward"` to the suite in the generator manifest, keep the `skip_subtests`
   entry, regenerate `tests.json`, and validate it with `qualify-set --check`.
2. Copy the regenerated suite entry into a scratch manifest, remove its skip there, and run
   `qualify-set`. Require `qualified` on seeds 11 and 17 at the manifest's repeat count, and a
   traced run with exact choice-tape replay.
3. Run the leaf on at least seeds 1 to 64 under `forward`. Require 64 passes, one `WorkerCommands`
   task per seed, and no `ActivityTimeout` task on any seed (`fn105-d18-run.sh` with
   `fn105-d18-summarize.py` reports all three).
4. Keep the strict-tick leaf sweep as the regression reproducer. It fails on 9 of 16 seeds today,
   and every failure must still show the activity timer task running first.
5. Run the native suite and require it green.
6. When steps 2 to 5 pass, remove the `skip_subtests` entry from the generator manifest, regenerate
   `tests.json`, and qualify the suite from the tracked manifest.
7. Run the suite on linux/amd64 when a host is available. The decision must state whether that run
   gates step 6. The corrections D22 to D25 removed their skips on darwin/arm64 evidence and
   recorded that linux/amd64 was not run.

Whichever option the decision selects, including keeping the skip, the skip's recorded reason
should be replaced with the cause stated here.

## Limits

- No linux/amd64 run.
- The cause is established from server debug logs, source, and three interventions. No history
  events were fetched, so the `ActivityTaskTimedOut` event on failing seeds is inferred from the
  processed `ActivityTimeoutTask`, the workflow task scheduled as event 8, and the source.
- The claim that the seed decides the order of two equal-deadline timer tasks rests on the
  observed orders. The timer queue's scheduling code was not traced.
- Log timestamps have millisecond resolution. Under `forward` the activity deadline and the run
  expiration can print the same millisecond. That the guard saw them as different rests on the
  absence of an `ActivityTimeout` task on all 64 seeds, not on a printed difference.
- Option A was verified through scratch manifests. The generator, the generated manifest, and the
  full `./tests` set were not run with it.
- Variants B and C ran only for the leaf under the strict tick on 16 seeds. Neither ran as a suite,
  under `qualify-set`, or against unit tests, and variant C did not run natively.
- No run exercised a dispatch that arrives in the 100 ms between two control-queue polls.
