# Gomad D19: activity fairness is measured before the backlog is complete

Assessment date: 2026-10-01.

`TestFairnessSuite/Test_Activity_Basic` and `TestFairnessAutoEnableSuite/Test_Activity_Basic` fail
under Gomad because of the test's setup. No run shows a fairness defect in matching. The test starts
polling activity tasks as soon as the last workflow task is completed and assumes that all 225
activity tasks are then in matching's backlog. Under Gomad they are not, on any of seeds 1 to 17,
when the whole suite runs. The history service reads each shard's transfer queue through a rate
limiter of 20 reads per second with a burst of 20. The sibling tests of the suite use up that burst
in the first virtual instant. The next read waits 50 ms on a timer, and Gomad delivers a timer only
when no goroutine is runnable. The test's poll loop stays runnable for as long as matching has a
task to hand out, so the test drains a partial backlog before the other tasks arrive. In the runs
here the partial backlog held the tasks of 0 to 14 of the 15 workflows. Fairness keys that occur
only in the late workflows are dispatched late, and the test's metric exceeds 1.0 on 7 of 17 seeds.

In every run where all 225 transfers had started before the first poll, the metric was 0.45, the
lowest value ten keys can produce: 191 of 191 such Gomad runs and every native run. In all 79
Gomad runs with a partial backlog, the first dispatches carried every key the partial backlog
held, one dispatch per key.

The auto-enable suite has a second, separate failure in its `triggerAutoEnable` helper. The
helper polls for one activity task while auto-enable reloads the activity queue. The reload can
end the waiting poll with an empty response, and the helper treats that as an error. Which
happens first is a scheduling choice under Gomad (3 of 17 seeds fail under the strict tick and 2
of 17 under forward), and it did not happen in 34 native runs.

The recommended correction is a test change with two parts: wait until the backlog holds all 225
tasks before measuring, and poll again when the trigger poll returns empty. Both parts use
observations and patterns the test file already has. In a throwaway copy with the change, both
suites pass on seeds 1 to 17, qualify on seeds 11 and 17 with the skip lifted, and pass natively.
Nothing in this report is implemented or fixed. Both skips stay, and the correction is open work in
fn-105 until a decision selects it and its verification passes.

This is the receipt for
[D19](../../../.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.19.md) (fn-105 R19).

## Scope and identities

| Item | Value |
| --- | --- |
| Repository revision | `7ef3800a2c` plus the shared uncommitted working tree for the first part of the investigation; the fresh clone `6782b55f4` for the rest (see below) |
| Test source | `tests/priority_fairness_test.go`, SHA-256 `ce9d9b9e...b293fc`, identical at both revisions |
| Toolchain | go1.27.1, build key `8d28bd4486f0b6300e8d25efd4caf8cb6ccbf000e96dbd26b1d8f53bf5f251bc` |
| Runner | `tools/gomad3/.bin/gomad`, SHA-256 `4dce3aed...db5222` before the re-clone and `0e00a1be...e42a2f` after |
| Native Go | go1.27.0 darwin/arm64 |
| Host | darwin/arm64, macOS 26.6.2 (25G83) |
| linux/amd64 | Not available. No linux run exists for this report. |

The checkout was replaced during the investigation by a fresh clone at `6782b55f4`, which contains
the earlier working tree as a commit. The test, the reader rate limiter source, `go.mod`, and both
qualification manifests have the same SHA-256 before and after. The seed 11 and 17 runs, the
qualification runs, the sweep of the proposed change, and one native run per suite were repeated
on the clone, and each repeated run gave the outcome of its predecessor. Runs named `r2-` in the
evidence are the repeats, and runs named `r3-` are the final form of the proposed change.

Source references are `file:line` at `6782b55f4`. Evidence files are under
`.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/` with the prefix `fn105-d19-`;
[`fn105-d19-evidence.json`](../../../.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d19-evidence.json)
indexes them and records every command. No tracked product, test, runtime, or manifest file was
edited. The skips were lifted only through a manifest copy in a scratch directory. The code
variants ran in throwaway copies of the source tree, and natively through `go test -overlay`.

## The test and its fairness contract

The test starts 15 workflows and completes one workflow task for each. Every completion schedules
15 activities, each with a fairness key drawn from a Zipf distribution with a fixed seed
(`tests/priority_fairness_test.go:529-580`). The 225 tasks carry ten keys. Key 0 has 95 tasks and
key 9 has 2. The test then polls and completes 225 activity tasks one at a time and records the
key of each (`:582-599`).

The metric is the sum, over the keys, of the index of each key's first dispatch, divided by the
square of the number of keys (`:606-618`). The assertion is `unfairness < 1.0` (`:603`). With ten
keys the lowest possible value is 0.45, reached when the first ten dispatches carry ten different
keys. The assertion fails when the first-dispatch indices sum to 100 or more. If seven keys are
dispatched first, the remaining three may not average more than about index 26.

The contract the test checks is therefore: a key with few tasks gets its first turn early, however
many tasks other keys have queued. That contract is about tasks that are eligible. A key cannot
take a turn before its first task is in the backlog. The test has no step that establishes this.
`PrioritySuite.TestActivity_Basic` in the same file has one, with the comment "wait for activity
tasks to appear in the matching backlog (from transfer queue)" (`:112-133`).

## Reproduction

Gomad runs use `gomad explore` with the qualification manifest's build tags, closure mode, and
schema mount (wrapper `fn105-d19-run.sh`; `fn105-d19-sweep.sh` runs seed batches). The suite
patterns are `^TestFairnessSuite$` and `^TestFairnessAutoEnableSuite$`. The leaf pattern adds
`/^Test_Activity_Basic$`. All run with `-test.parallel=8 -test.v`. "Backlog incomplete" means that
fewer than 225 of the test's `TransferActivityTask`s had started executing in history when the
first measured activity poll reached the frontend.

| Run, unchanged test | Tick | Seeds | Passed | Failed the fairness assertion (`:603`) | Failed in `triggerAutoEnable` (`:508`) | Backlog incomplete |
| --- | --- | --- | --- | --- | --- | --- |
| `TestFairnessSuite`, leaf alone | strict | 11, 17 | 2 | 0 | 0 | 0 of 2 |
| `TestFairnessSuite`, suite | strict | 11, 17 | 1 (seed 17, 0.85) | 1 (seed 11, 1.3) | 0 | 2 of 2 |
| `TestFairnessSuite`, suite | forward | 11, 17 | 0 | 2 (1.15 and 1.4) | 0 | 2 of 2 |
| `TestFairnessSuite`, suite | strict | 1 to 17 | 10 | 7 | 0 | 17 of 17 |
| `TestFairnessSuite`, suite | forward | 1 to 17 | 10 | 7 | 0 | 17 of 17 |
| `TestFairnessAutoEnableSuite`, leaf alone | strict | 11, 17 | 1 (seed 17) | 0 | 1 (seed 11) | 0 of 1 |
| `TestFairnessAutoEnableSuite`, suite | strict | 11, 17 | 1 (seed 17) | 0 | 1 (seed 11) | 1 of 1 |
| `TestFairnessAutoEnableSuite`, suite | forward | 11, 17 | 2 | 0 | 0 | 2 of 2 |
| `TestFairnessAutoEnableSuite`, suite | strict | 1 to 17 | 13 | 1 (seed 9, 1.04) | 3 (seeds 6, 11, 12) | 14 of 14 |
| `TestFairnessAutoEnableSuite`, suite | forward | 1 to 17 | 15 | 0 | 2 (seeds 10, 16) | 15 of 15 |
| `qualify-set`, skips lifted, both suites | strict | 11, 17 | `TestFairnessSuite` seed 11 | `TestFairnessSuite` seed 17 (1.65) | `TestFairnessAutoEnableSuite` seeds 11 and 17 | not measured |
| Native, both suites and leaves, 12 single runs | wall clock | none | 12 | 0 | 0 | 8 of 12 |
| Native, `TestFairnessAutoEnableSuite` leaf, `-count=30` | wall clock | none | 29 | 0 | 0 (one run failed at `:468`, see below) | not measured |

`fn105-d19-gomad-runs.txt` lists every Gomad run, 288 in total with the variants below, one line per
seed. `fn105-d19-native-results.txt` lists the native runs. No run hit a wall watchdog, an
infrastructure error, or a replay divergence. `qualify-set` replayed its three failures with
matching outcomes. It runs without `-test.v`, so its schedules differ from the `explore` runs of the
same seeds.

Three observations follow from the table and the run list.

- A run alone passes and a run inside the suite does not, so the failure depends on what else
  runs on the shared cluster. A seed does not own it.
- Across all 288 runs, variants included, the backlog was complete at the first poll in 191, and
  all 191 scored 0.45. It was incomplete in 79, of which 57 passed and 22 failed. The other 18
  failed in `triggerAutoEnable` before the measurement. A partial backlog passes when the rare
  keys happen to be in the early workflows.
- The recorded skip reason gives 1.03 to 1.04 for `TestFairnessSuite` on seeds 11 and 17 and 1.27
  for the auto-enable suite on seed 11. Today's values differ, and the auto-enable suite's seed 11
  no longer reaches the assertion. The values belong to a schedule, and the schedule changed with
  the toolchain.

## Timeline

The server logs every queue task at debug level. `fn105-d19-timeline.py` reduces a run to four
event kinds in log order: transfer task created, transfer task execution started, activity poll
received, and activity task completed. Under the strict tick every event of one busy stretch
carries the same virtual timestamp, so the order of the log lines is the timeline. The table is the
`TestFairnessSuite` suite run under the strict tick
(`fn105-d19-gomad-suite-strict-timeline.txt`).

| Virtual time | Step | Seed 11 (fails, 1.3) | Seed 17 (passes, 0.85) |
| --- | --- | --- | --- |
| 0.000 | 15 workflow tasks completed | 225 `TransferActivityTask`s created | 225 created |
| 0.000 | First activity poll | 30 transfers started, from workflows 2 and 11 | 45 started, from workflows 0, 2, and 11 |
| 0.000 | Dispatches 1 to 30 (seed 17: 1 to 45) | All from those workflows. No further transfer starts. 7 of 10 keys appear, in the first 7 dispatches. | All from those workflows. 9 of 10 keys appear, in the first 9 dispatches. |
| 0.000 | The backlog is empty and the next poll waits | No goroutine is runnable | No goroutine is runnable |
| 0.050 | The clock advances to the earliest timer | The other 195 transfers start | The other 180 transfers start |
| 0.050 | Remaining dispatches | Keys 7, 8, and 9 first appear at indices 34, 36, and 39 | Key 9 first appears at index 49 |
| 0.050 | Metric | (0+1+3+4+6+2+5+34+36+39)/100 = 1.3 | (0+2+1+8+3+7+4+5+6+49)/100 = 0.85 |

Run alone on the same seeds, all 225 transfers start before the first poll, the first ten
dispatches carry the ten keys, and the metric is 0.45 (`fn105-d19-gomad-controls-timeline.txt`).

The per-shard view shows where the transfers stop (`fn105-d19-shards.py`, same file). The test
cluster has four history shards, and the sibling tests start their workflows in the same instant.

| Seed 11, before the first activity poll | Transfer tasks created | Started | This test's workflows |
| --- | --- | --- | --- |
| Shard 1 | 51 | 49 | 2 and 11, all 30 tasks started |
| Shard 2 | 116 | 19 | 1, 3, 4, 5, 6, 14, none started |
| Shard 3 | 83 | 19 | 8, 9, 12, 13, none started |
| Shard 4 | 73 | 19 | 0, 7, 10, none started |

Three shards stop at exactly 19 started tasks on seed 11, and two on seed 17, where shard 4
reaches 33. In the leaf run every shard starts all of its 48 to 64 tasks.

## Cause

Four facts combine.

1. **Every read of a shard's transfer queue takes a rate-limiter token.** `loadAndSubmitTasks` calls
   `r.ratelimiter.Wait` before it reads (`service/history/queues/reader.go:427-436`). The limiter is
   built per shard from `MaxPollRPS` (`service/history/queues/queue_base.go:137-141`,
   `reader_quotas.go:39-51`), which the transfer queue sets from `TransferProcessorMaxPollRPS`
   (`service/history/transfer_queue_factory.go:195`). `history.transferProcessorMaxPollRPS` defaults
   to 20 (`common/dynamicconfig/constants.go:2275-2279`), and the burst equals the rate
   (`common/quotas/dynamic_rate_limiter_impl.go:11,57-64`, `common/quotas/rate_burst.go:102-130`). A
   shard can read 20 times at one instant and then once per 50 ms.
2. **A waiting read sleeps on a timer.** `Wait` reads `time.Now`, reserves a token, and blocks on
   `time.NewTimer(delay)` (`common/quotas/multi_request_rate_limiter_impl.go:70-105`).
3. **Under Gomad the limiter cannot refill while work is runnable.** The
   [contract](../../../tools/gomad3/README.md#contract) states that under the strict tick every
   `time.Now` within one busy stretch returns the same instant, and that runnable work is never
   skipped to deliver a future timer. In the suite, the sibling tests start their workflows in the
   same first instant on the same four shards. The 19 started tasks on the stalled shards are one
   short of the 20-token burst, which fits one task per read. The 50 ms timer then waits for the
   process to go idle.
4. **The test keeps the process busy until it has drained the partial backlog.** Each poll returns
   at once while matching holds a task, so the test goroutine is runnable through dispatches 1 to
   30. The process goes idle only when the backlog is empty and the poll parks. The clock then
   jumps 50 ms, the readers resume, and the remaining tasks arrive after the measurement has
   already recorded 30 dispatches.

The readers resume at the first clock advance of the run, and the advance is 50 ms, the limiter's
period. The intervention below removes the stall by changing the limiter alone.

The forward tick does not help. It advances `time.Now` by 1 to 1024 ns per reading, which is far
from the 50 ms a token needs, and the timer still waits for an idle process. 7 of 17 seeds fail
under either tick.

The recorded skip reason names the symptom correctly: polling begins before the backlog is
complete, and the first dispatches come from a few workflows. Its explanation, that server work
takes no time under virtual time, is not the mechanism. The test passes alone on both seeds, where
server work takes equally little time. The mechanism is a rate limiter that needs elapsed time
which the busy test never grants.

## Native comparison

Natively the same limiter refills in real time, and a reader waits at most 50 ms of wall time
while the test is still completing workflow tasks. The backlog is nevertheless incomplete at the
first poll in 8 of 12 native runs: 210 to 224 of 225 transfers had started, and the missing ones
belonged to a single workflow: `wf14`, the last one the test starts, in 7 of the 8 runs and `wf5`
in the other (`fn105-d19-native-results.txt`, last section). In all 12 runs every transfer had
started before the first dispatch completed, and all 41 passing native runs scored 0.45.

The precondition the test assumes is therefore unmet natively as well. Wall-clock latency closes
the gap within one poll round trip, so the test does not notice. Under Gomad the gap stays open
for 30 or more dispatches.

## Setup bias or product defect

The evidence points to setup bias and gives no sign of a matcher fairness defect.

- Given a complete backlog, the dispatch order is ideal. Every one of the 191 Gomad runs and all
  native runs with a complete backlog put ten different keys into the first ten dispatches.
- Given a partial backlog, the matcher is fair to what it has. In all 79 runs with a partial
  backlog, the first K dispatches carried K different keys, where K is the number of keys among
  the workflows transferred at the first poll (`partial=` in `fn105-d19-gomad-runs.txt`; one of
  the 79 had no workflow transferred, so K is 0 there). On seed 11 the first 7 dispatches carry
  the 7 keys present in workflows 2 and 11. Keys 7, 8, and 9 are dispatched at indices 34,
  36, and 39, within nine dispatches of their arrival after dispatch 30, while more than 100 tasks
  of keys 0 and 1 were still queued.
- The failing quantity is the arrival time of tasks, which history's transfer queue decides.
  Matching cannot dispatch a task it has not received.

The rate limiter is working as designed. This report claims no product defect in history either.

## The auto-enable trigger

`TestFairnessAutoEnableSuite` first calls `triggerAutoEnable` (`:456-517`), which runs one workflow
with one activity so that matching switches the task queue to fairness. The helper has two
assumptions about ordering. `fn105-d19-trigger-phase.txt` holds the reduced logs.

**The trigger activity poll (`:501-508`).** The helper polls once and requires a task. The
sequence is the same natively and under Gomad up to the last step:

1. The activity task reaches matching. `AddTask` calls `autoEnableIfNeeded`, which sends
   `UpdateFairnessState` and waits for it
   (`service/matching/task_queue_partition_manager.go:524-553,570`). The update takes 100 ms in both
   environments.
2. The test's poll arrives and waits on the activity queue, which is still a classic queue.
3. The user data changes. The activity partition sees the new fairness state and unloads itself
   (`task_queue_partition_manager.go:2479-2481`), and the log shows the classic queue stopped with
   cause `ConfigChange`.
4. Either the waiting poll receives the trigger task, or it returns empty. The poller helper turns
   an empty response into `NoActivityTaskAvailable`
   (`common/testing/taskpoller/taskpoller.go:576-577`), and `s.NoError` fails at `:508`.

Under Gomad step 4 falls in the same virtual instant as step 3, and the seed decides. Seed 17
receives the task, and seed 11 gets the empty response. The failure occurs on 3 of 17 seeds under
the strict tick, 2 of 17 under forward, and 5 of 17 with the rate limiter raised, so it does not
depend on the limiter. Natively the task arrived 1 ms after the unload in the run inspected, and
all 34 native runs that reached this poll received the task.

An empty poll response is normal for a long poll, and a worker polls again. On the seeds where
the proposed change polls again, the second poll loads the queue as a fairness queue and receives
the task 0.9 s of virtual time later.

**The trigger workflow task wait (`:468-477`).** The helper waits until `GetTaskQueueTasks` with
`MinPass: 1` returns one workflow task, which is true once the task is in the fairness backlog.
One native run of 30 failed here after 10 s. Its log shows the classic workflow queue unloaded and
never reloaded, and none of the lines that the passing runs show at that point: a failed write, a
failed `AddWorkflowTask` with `task queue shutting down`, and the retry that writes to the
fairness queue. No Gomad run failed at this line. This report records the native failure and does
not explain it.

## Interventions

Each intervention ran on the suites under the strict tick. Variant R and the P-only run exist to
separate the causes. Only P+T is proposed.

| Intervention | What changes | Where it ran | `TestFairnessSuite`, seeds 1 to 17 | `TestFairnessAutoEnableSuite`, seeds 1 to 17 |
| --- | --- | --- | --- | --- |
| None | | Working tree | 10 pass, 7 fail at `:603`. Backlog incomplete on 17. | 13 pass, 1 fails at `:603`, 3 fail at `:508`. Backlog incomplete on 14 of 14. |
| Variant R: the shard reader rate is multiplied by 1000 (`fn105-d19-variant-r-server.diff.txt`). Diagnostic only. | Fact 1: the burst is never exhausted | Throwaway copy, test unchanged | 17 pass at 0.45. Backlog complete on 17. | 12 pass at 0.45, 5 fail at `:508`. Backlog complete on 12 of 12. |
| Variant P: wait for 225 tasks in the backlog before the first poll | The test's precondition | Throwaway copy | 17 pass at 0.45. Backlog complete on 17. | 13 pass at 0.45, 4 fail at `:508`. Backlog complete on 13 of 13. |
| Variant P+T: P, and the trigger activity poll is repeated until it returns a task (`fn105-d19-variant-pt-test.diff.txt`) | The test's precondition and the helper's poll | Throwaway copy, three sweeps | 17 pass at 0.45, each time | 17 pass at 0.45, each time |

Variant R shows that the limiter is the reason the backlog is incomplete: with it out of the way,
all 225 transfers start before the first poll on every seed, with the test unchanged. Variant P
shows that the fairness assertion holds on every seed once the backlog is complete. The `:508`
failures move to other seeds with each rebuilt binary and disappear only with T.

The first two P+T sweeps used `assert.NoError` inside the retry, as `processWft` does. The
proposed diff uses `require.NoError`, which the repository's test guidance prefers inside
`Eventually` callbacks, and the third sweep and the `r3-` runs below used that form.

| Run of P+T | Form | Result |
| --- | --- | --- |
| Both suites, strict tick, seeds 1 to 17 | assert, twice | 68 pass at 0.45 |
| Both suites, strict tick, seeds 1 to 17 | as proposed | 34 pass at 0.45 |
| Both suites, forward tick, seeds 11 and 17 | each form once | 8 pass at 0.45 |
| Both leaves alone, strict tick, seeds 11 and 17 | each form once | 8 pass at 0.45 |
| `qualify-set`, skips lifted, strict tick, seeds 11 and 17, repeat 2 | assert twice, as proposed once | Both suites `qualified` on both seeds, each time. The unchanged test gives `target_failure` for both suites from the same manifest. |
| Native, both suites, through `go test -overlay` | assert 4 runs per suite, as proposed 2 per suite | 12 pass at 0.45. All 225 transfers had started before the first poll in all 12. |

P+T leaves the workload and the assertion untouched: 15 workflows, 15 activities each, the Zipf
keys with their seed, one poll per dispatch, the metric, and the `< 1.0` threshold.

## Corrections and owners

| Option | Change | Owner | Evidence here | Cost and risk |
| --- | --- | --- | --- | --- |
| A | Variant P+T in `tests/priority_fairness_test.go`. P: before the first activity poll, wait until `countTasksByDrainingActive` (`:795-825`, `DescribeTaskQueuePartition`'s `ApproximateBacklogCount`) reports 225 activity tasks, as `PrioritySuite.TestActivity_Basic` already does at `:112-133`. T: wrap the trigger activity poll in `EventuallyWithT`, as `processWft` does for workflow task polls at `:686-718`, with `require.NoError` in the callback. Remove both `skip_subtests` entries only after the verification below passes. | Owner of `tests/priority_fairness_test.go` (matching) | The table above | A change to a Temporal test. It makes a precondition explicit that the test already assumes and that a sibling test already checks, it is valid natively, and it needs no Gomad-specific code. The milestones' constraint against test rewriting applies unless the test owner accepts it on those grounds, as for D22 to D24. It adds one wait of at most 10 s. |
| B | Raise the transfer reader poll rate for functional test clusters in `tests/testcore/dynamic_config_overrides.go` | Owner of `tests/testcore` | Variant R only, which changes the product default in a copy. No run used a test-cluster override. | Changes the pacing of every functional test, natively too. It leaves the precondition implicit and does not address the trigger poll: 5 of 17 seeds still fail at `:508`. |
| C | `clock_tick: forward` for the two suites | Gomad qualification configuration | Ruled out: 7 of 17 and 2 of 17 seeds fail | None, it does not work |
| D | Keep both skips and replace the recorded reason with the cause stated here | Gomad qualification configuration (`tools/gomad3integration`) | This report | Two tests stay unqualified |

Recommendation: option A. The cause is a precondition the test does not establish, the observable
to wait on exists in the test's own suite, and the change holds natively and under both tick
policies. Option D's reason replacement applies whichever option the decision selects. The
recorded reason is inaccurate about the mechanism and does not mention the trigger failure.

Proposed next action, as open work in fn-105 for a subsequent decision. Both skips stay in the
tracked manifest through step 5.

1. Apply `fn105-d19-variant-pt-test.diff.txt` to `tests/priority_fairness_test.go` with the test
   owner's agreement. Run the repository's lint on it.
2. Run both suites natively and require them green. Run the auto-enable leaf with `-count=30` and
   record whether the `:468` wait fails; that failure exists without the change and is not a
   criterion for it.
3. Copy the two suite entries of `tests.json` into a scratch manifest, remove their skips there,
   and run `qualify-set`. Require `qualified` on seeds 11 and 17 at the manifest's repeat count,
   and a traced run with exact choice-tape replay.
4. Run both suites on at least seeds 1 to 17 under the strict tick. Require 34 passes, a metric
   below 1.0 on each, and 225 of 225 transfers started at the first measured poll
   (`fn105-d19-sweep.sh` reports all three).
5. Keep the unchanged-test sweep as the regression reproducer. It fails at `:603` on 7 of 17 seeds
   for `TestFairnessSuite` today, with an incomplete backlog on all 17.
6. When steps 2 to 5 pass, remove the two `skip_subtests` entries from the generator manifest,
   regenerate `tests.json`, and qualify both suites from the tracked manifest.
7. Run both suites on linux/amd64 when a host is available. The decision must state whether that
   run gates step 6.

## Limits

- No linux/amd64 run.
- "Transfer started" is the history log line for a transfer task beginning to execute. Matching
  does not log the arrival of a task in its backlog. For the unchanged test the backlog state is
  inferred from that line. Variant P observes the backlog count directly before it polls.
- The limiter is identified from the source, the per-shard counts, the 50 ms resume, and variant
  R. No run logged the limiter's wait itself, and the read that takes the 20th token was not
  traced.
- Seeds 1 to 17 only. The proportion of failing seeds is a sample.
- In the trigger poll, the code path that hands the task to a waiting poll at the moment of the
  unload was not traced, and neither was the 0.9 s before the second poll receives the task.
- The native `:468` failure was seen once in 30 runs and is not explained. Option A does not
  address it.
- Option A was verified through a scratch manifest, throwaway copies, and a build overlay. The
  generator, the generated manifest, and the full `./tests` set were not run with it, and no
  traced run with choice-tape replay was made.
- Variant R ran only as a diagnostic. Option B itself was not implemented or run.
