# Gomad D20: the heartbeat test sends its heartbeats exactly at the deadline

Assessment date: 2026-10-01.

`TestWorkflowTaskTestSuite/TestWorkflowTaskHeartbeatingWithEmptyResult` fails under Gomad because
of the test's own timing arithmetic. The server rejects a workflow task heartbeat only when the
chain of heartbeats has run for strictly longer than the 5 s heartbeat timeout. The test sends its
heartbeats one second apart, so one of them is due exactly 5 s after each chain began. Natively
that heartbeat arrives 10 to 92 ms late and is rejected. Under Gomad's default `strict` tick the
server work between two sleeps takes no virtual time, the heartbeat arrives at exactly 5.000000000
s, and the server accepts it. The first rejection then comes one heartbeat later, the second
chain ends the same way at the last loop iteration, and the test counts one rejection where it
expects two. Every seed tried fails identically: leaf seeds 1 to 17, and seeds 11 and 17 in the
whole suite.

The server behaves as its code and comment state in both modes, and this report claims no product
defect. Both expected rejections depend on incidental latency, not only the second. A native run
with 995 ms sleeps fails 2 of 3 times, which shows the native margin directly.

The recommended correction is a test change that states the deadline: the two heartbeats that must
be rejected are sent 100 ms after a deadline the test computes, and every heartbeat's outcome is
asserted. In a throwaway copy it passes natively, on Gomad seeds 1 to 64 under `strict`, on seeds
1 to 16 under `forward`, and in `qualify-set` on seeds 11 and 17 with the skip lifted under both
ticks. It keeps the expected history, the two timeouts, and both recoveries. The rejections then
need no latency. The accepted heartbeats still need latency to stay below a bound, about 0.93 s
per chain, as they do today, so the corrected test is not independent of time. `clock_tick:
forward` for the suite and an inclusive server comparison are recorded as alternatives with their
owners.
Nothing in this report is implemented or fixed. The skip stays, and the correction is open work in
fn-105 until a decision selects it and its verification passes.

This is the receipt for
[D20](../../../.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.20.md) (fn-105 R20).

## Scope and identities

| Item | Value |
| --- | --- |
| Repository revision | `6782b55f49` plus the shared uncommitted working tree |
| Test source | `tests/workflow_task_test.go`, SHA-256 `2e42b3f4...343f07` |
| Server source | `service/history/api/respondworkflowtaskcompleted/api.go`, SHA-256 `afb6e333...ae1d56` |
| Toolchain | go1.27.1, build key `8d28bd4486f0b6300e8d25efd4caf8cb6ccbf000e96dbd26b1d8f53bf5f251bc` |
| Runner | `tools/gomad3/.bin/gomad`, SHA-256 `0e00a1be...42a2f` |
| Native Go | go1.27.0 darwin/arm64 |
| Host | darwin/arm64, macOS 26.6.2 (25G83) |
| linux/amd64 | Not available. No linux run exists for this report. |

Source references are `file:line` in the repository at the revision above. Evidence files are
under `.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/` with the prefix `fn105-d20-`;
[`fn105-d20-evidence.json`](../../../.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d20-evidence.json)
indexes them and records the command templates and every run with its seeds and outcome. No tracked product, test, runtime, or manifest file was
edited. The skip was lifted only through manifest copies in a scratch directory. Variants ran under
Gomad in a throwaway copy of the source tree and natively through `go test -overlay`.

## Configured deadlines

| Setting | Value | Source |
| --- | --- | --- |
| `WorkflowTaskHeartbeatTimeout` | 5 s in functional tests, 30 min by default | `tests/testcore/dynamic_config_overrides.go:39`, `common/dynamicconfig/constants.go:2772` |
| Workflow task start-to-close timeout | 3 s | `tests/workflow_task_test.go:48` |
| Workflow run timeout | 20 s | `tests/workflow_task_test.go:47` |
| Heartbeats | 12 loop iterations, `time.Sleep(time.Second)` after each | `tests/workflow_task_test.go:80,107` |
| Expected rejections | 2 | `tests/workflow_task_test.go:110` |

## Timeout and reset semantics

A heartbeat is a `RespondWorkflowTaskCompleted` request with `ForceCreateNewWorkflowTask` set and
no commands or messages (`service/history/api/respondworkflowtaskcompleted/api.go:313`). The
server's rules, from the source and confirmed by every history below:

- **Deadline.** The server rejects the heartbeat when
  `timeSource.Now().After(OriginalScheduledTime.Add(timeout))` (`api.go:322-324`). The comparison is
  strict. The comment above it calls the timeout "a total duration for which workflow is allowed
  to send continuous heartbeats", and a heartbeat sent exactly when that duration ends is accepted.
  The handler's time source is the shard's (`api.go:89`), which the server provides as
  `RealTimeSource` (`common/resource/fx.go:152-153`), and its `Now` is `time.Now`
  (`common/clock/time_source.go:38`).
- **Start of the interval.** `OriginalScheduledTime` is read when a workflow task is scheduled
  outside a heartbeat (`service/history/workflow/workflow_task_state_machine.go:411`). A workflow
  task scheduled by an accepted heartbeat inherits the value (`api.go:617-622`), so a chain of
  heartbeats is measured from the scheduling of its first workflow task.
- **Rejection.** The server records `WorkflowTaskTimedOut` for the current workflow task, clears
  the sticky queue, and returns `NotFound` "workflow task heartbeat timeout" (`api.go:331-336,
  783-786`). The event carries timeout type StartToClose
  (`workflow_task_state_machine.go:1015-1021`), the same type the timer uses.
- **Reset.** The next workflow task is scheduled through the non-heartbeat path (`api.go:624`) and
  gets a new `OriginalScheduledTime`. The interval restarts with every workflow task that an
  accepted heartbeat did not schedule.
- **Recovery.** `ReturnNewWorkflowTask` is set and the request did not fail, so the server also
  starts the new workflow task inline (`api.go:577, 631-636`). The worker receives the error and
  no task token, so nobody holds that task. It times out after its 3 s start-to-close timeout
  (`service/history/timer_queue_active_task_executor.go:442-455`). The same transaction schedules
  another workflow task with a new `OriginalScheduledTime`
  (`timer_queue_active_task_executor.go:477, 1013-1027`), and the test's `PollWorkflowTaskQueue`
  receives it. Each rejection therefore produces two `WorkflowTaskTimedOut` events 3 s apart,
  which the test's expected history lists as events 19 and 22, and 37 and 40
  (`tests/workflow_task_test.go:133-180`).

## Reproduction

Gomad single-suite and single-test runs use `gomad explore` with the qualification manifest's
build tags, closure mode, and schema mount (wrapper `fn105-d20-run.sh`). The suite pattern is
`^TestWorkflowTaskTestSuite$` and the leaf pattern adds
`/^TestWorkflowTaskHeartbeatingWithEmptyResult$`. Both run with `-test.parallel=8 -test.v`.
"Instrumented" runs add logging only (`fn105-d20-variant-i-instrumentation.diff.txt`): the send
and return time of every heartbeat and the history with event times.

| Run | Tick | Seeds | Outcome of the heartbeat test |
| --- | --- | --- | --- |
| Leaf, unchanged tree | strict | 11, 17 | Both fail at 15.00 s virtual: expected 2, actual 1 (`workflow_task_test.go:110`) |
| Leaf, unchanged tree | strict | 1 to 16 | 16 fail, each at 15.00 s virtual |
| Suite, unchanged tree | strict | 11, 17 | The heartbeat test fails on both; the other 8 subtests pass |
| Leaf and suite, instrumented | strict | 11, 17 | Same failure. One rejection, two heartbeats accepted at exactly chain + 5.000000000 s. |
| `qualify-set`, skip lifted | strict | 11, 17 | Both `target_failure`, each replayed with a matching failure |
| Leaf, unchanged tree | forward | 1 to 32 | 32 pass |
| Suite, unchanged tree | forward | 11, 17 | 9 of 9 subtests pass on both |
| Leaf and suite, instrumented | forward | 11, 17 | Pass. Two rejections, 1.4 to 19.8 ms past the deadline. |
| `qualify-set`, skip lifted, traced | forward | 11, 17 | Both `qualified` at repeat 2 with exact choice-tape replay |
| Native leaf, instrumented, 13 runs | wall clock | none | 13 pass. Rejections 10.3 to 92.3 ms past the deadline. |
| Native suite, `-count=2` | wall clock | none | 18 of 18 subtest runs pass |

No run hit a wall watchdog, an infrastructure error, or a replay divergence
(`fn105-d20-gomad-runs.txt`, `fn105-d20-native-results.txt`).

## Histories and elapsed time

Times are measured from the `WorkflowTaskScheduled` event that began the chain, using the event
times in the history (`fn105-d20-gomad-unchanged-histories.txt`,
`fn105-d20-native-unchanged-histories.txt`). Heartbeats are numbered by loop iteration from 0. The
native column is the first of 13 runs, and the forward column is leaf seed 11. Seeds 11 and 17
give the same strict instants, alone and in the suite.

| Step | Native | Gomad strict | Gomad forward |
| --- | --- | --- | --- |
| First chain begins (event 2) | 0 | 0 | 0 |
| Heartbeats 0 to 4 | Accepted at 0.042 to 4.051 s | Accepted at 0, 1, 2, 3, 4 s | Accepted at 0.001 to 4.002 s |
| Heartbeat 5, due at 5 s | **Rejected** at 5.052352 s (event 19) | **Accepted** at 5.000000000 s (event 19) | **Rejected** at 5.002287 s (event 19) |
| Heartbeat 6 | Second chain | **Rejected** at 6.000 s (event 22) | Second chain |
| Unclaimed inline task times out (start-to-close timer) | 3.002 s after it started (event 22) | 3.002 s after it started (event 25) | 3.002 s after it started (event 22) |
| Second chain begins | Event 23 | Event 26, at 9.002 s virtual | Event 23 |
| Heartbeats in the second chain | 6 to 9 accepted at 1.003 to 4.009 s | 7 to 10 accepted at 1, 2, 3, 4 s | 6 to 9 accepted at 1.000 to 4.001 s |
| Heartbeat due at 5 s | 10: **rejected** at 5.010260 s (event 37) | 11: **accepted** at 5.000000000 s (event 40) | 10: **rejected** at 5.001428 s (event 37) |
| Loop ends after | 18.03 s, 2 rejections | 15.002 s virtual, 1 rejection | 18.00 s virtual, 2 rejections |
| History at the count assertion | 45 events, matches the expected prefix | 42 events, differs from event 19 on | 45 events, matches the expected prefix |

Under the strict tick every heartbeat's send time equals its return time to the nanosecond: the
round trip takes no virtual time. `StartWorkflowExecution`, both `GetHistory` calls, the first
poll, and heartbeat 0 all carry the timestamp 00:00:00.000000000. Within a chain the clock
advances only in the test's own `time.Sleep`, by exactly one second each. Between the chains it
advances by 3.002 s while the recovery poll waits for the start-to-close timer. The fifth second
of each chain therefore lands on the deadline to the nanosecond, and `After` is false.

The strict run rejects heartbeat 6 instead, one second later. The second chain then begins one
second later and gets heartbeats 7 to 11. Heartbeat 11 is its tie, and the loop ends before a
heartbeat 12 could be rejected. Twelve iterations hold two rejections only when each chain is cut
at its fifth second.

## Cause

The test spaces its heartbeats by 1 s and the heartbeat timeout is 5 s, so the test's own schedule
puts one heartbeat on each chain's deadline. The test expects that heartbeat to be rejected
(events 19 and 37 of its expected history). The server rejects only strictly after the deadline.
The expectation holds whenever the heartbeat is late by any amount and fails when it is on time.

- Natively the heartbeat is always late, by the latency accounted for below.
- Under the strict tick it is on time. The
  [contract](../../../tools/gomad3/README.md#contract) states that every `time.Now` within one busy
  stretch returns the same instant, and the server work between two sleeps is one busy stretch.
- Under the forward tick it is late by the tick offset. Every `time.Now` advances the offset by 1
  to 1024 ns and the offset never decreases, so the reading the server compares is later than the
  reading it stored when the chain began.

The difference follows from the test's timing assumption. The server applies the same rule in all
three modes, and no run shows a runtime or server defect.

The skip's recorded reason names the mechanism correctly and is imprecise in two respects. Both
expected rejections depend on round-trip latency, not only the second. The loop spans 15.002 s of
virtual time, twelve sleeps plus one 3.002 s recovery, not 12 s.

## Native comparison

The 13 instrumented native runs give the time by which the heartbeat due at 5 s missed its
deadline, and what that time consists of (`fn105-d20-native-unchanged-histories.txt`).

| Quantity | Minimum | Median | Maximum |
| --- | --- | --- | --- |
| First chain: rejection past the deadline | 19.8 ms | 33.2 ms | 92.3 ms |
| Second chain: rejection past the deadline | 10.3 ms | 21.4 ms | 64.2 ms |
| One `RespondWorkflowTaskCompleted` round trip (156 calls) | 0.82 ms | 2.43 ms | 25.0 ms |
| Overshoot of one `time.Sleep(time.Second)` (143 sleeps) | 0.03 ms | 1.03 ms | 19.8 ms |
| First workflow task, scheduled event to started event | 2.5 ms | 7.1 ms | 43.9 ms |

- **First chain.** The interval starts inside `StartWorkflowExecution`. Before heartbeat 5 the
  test makes two `GetHistory` calls, one `PollWorkflowTaskQueue`, and five heartbeat round trips,
  and sleeps five times.
- **Second chain.** The interval starts when the timer task schedules the retry, just before the
  recovery poll returns. Before heartbeat 10 the test makes four heartbeat round trips and sleeps
  five times. In the run with the 10.3 ms margin the four round trips took 6.2 ms and the five
  sleeps overshot by 1.5 ms in total. The rest is the poll's return and the rejected request's
  way to the server.

The test's arithmetic assumes this latency at `tests/workflow_task_test.go:107`: five sleeps of
one second must add up to more than five seconds of server time.

A native control confirms the dependence (`fn105-d20-native-sleep-995ms-control.txt`). With the
sleep reduced to 995 ms and nothing else changed, 2 of 3 runs fail. In both, heartbeat 10 arrives
4.992 and 4.995 s into the second chain and is accepted, the rejection moves to heartbeat 11, and
the history assertion at line 133 fails. The run that passed was rejected 2.2 ms past the
deadline.

The opposite bound was not approached in these runs. The last accepted heartbeat of a chain
arrives 4.01 to 4.07 s into it natively, about 0.93 s before the deadline. That is a second
timing assumption of the test: accumulated latency must stay below that margin.

## Clock policies

| Policy | Result | Why |
| --- | --- | --- |
| `strict` | Fails on every seed tried, identically | The heartbeat due at 5 s arrives at exactly 5 s |
| `forward` | Passes on every seed tried | The tick offset makes it late: 2.29 and 1.43 ms in the leaf, 18.1 to 19.8 ms and 1.5 ms in the suite |

`forward` removes the tie and keeps the dependence. The rejection then rests on the offset that
accumulated between two `time.Now` readings, which is as incidental as the native latency. The
other eight subtests of the suite also pass under `forward`, and the suite qualifies with exact
replay. The offset would have to grow by a full second within four virtual seconds to reject a
heartbeat early. The largest growth seen was 19.8 ms in five virtual seconds.
[D16](GOMAD_D16_FORWARD_CLOCK_POLL_DEADLINE.md) records that this offset is cumulative and
unbounded and breaks a propagated deadline in another suite. No run of this suite showed that
effect.

## Interventions

Each intervention ran on the unchanged failure, where 16 of 16 leaf seeds fail under `strict`.

| Intervention | What changes | Where it ran | Result |
| --- | --- | --- | --- |
| `--clock-tick=forward` | The heartbeat is late by the tick offset | Working tree, no source change | Leaf seeds 1 to 32 pass. Suite seeds 11 and 17 pass. `qualify-set` with the skip lifted is `qualified` with exact replay. |
| Variant T: the test waits until 100 ms past a deadline it computes before heartbeats 5 and 10 and asserts each heartbeat's outcome (`fn105-d20-variant-t-test.diff.txt`) | The test no longer sends a heartbeat on the deadline | Throwaway copy and native overlay | Strict leaf seeds 1 to 64 pass. Forward leaf seeds 1 to 16 pass. `qualify-set` with the skip lifted is `qualified` with exact replay under `strict` and under `forward`. Native leaf 8 of 8, native suite 18 of 18 subtest runs. |
| Variant P: the server rejects with `!Now.Before(deadline)` (`fn105-d20-variant-p-server.diff.txt`) | A heartbeat exactly on the deadline is rejected | Throwaway copy and native overlay | Strict and forward leaf seeds 1 to 16 pass. `qualify-set` with the skip lifted is `qualified` with exact replay under `strict`. Native leaf 3 of 3, native suite 9 of 9, the three heartbeat unit tests in `service/history` pass. |

With variant T the rejections land 100.0 ms past the server's deadline under `strict`, 105.0 to
106.5 ms under `forward`, and 103.6 to 109.6 ms natively
(`fn105-d20-gomad-variant-t-histories.txt`, `fn105-d20-native-variant-t-histories.txt`). With
variant P under `strict` both rejections are recorded at chain + 5.000000000 s
(`fn105-d20-gomad-variant-p-histories.txt`).

Variants T and P keep the count assertion, the 47-event expected history, both recoveries, and
the final completion unchanged. Variant T adds one assertion per heartbeat.

## The proposed condition

Variant T states the semantic condition instead of leaving it to latency: a heartbeat sent after
the chain's deadline is rejected, and one sent before it is accepted.

- The test records `chainDeadline = time.Now() + 5 s` after the call that returned the chain's
  first workflow task: the first poll, and each recovery poll. The server scheduled that task
  before the call returned, so `chainDeadline` is not earlier than the server's deadline.
- Before heartbeats 5 and 10 the test sleeps until `chainDeadline` plus 100 ms. Those two
  heartbeats must be rejected with `NotFound`, and all others must be accepted. The test asserts
  this for each heartbeat, which is stronger than counting two rejections. The suite's `s.Equal`
  is testify's `require` (`common/testing/parallelsuite/suite.go:59, 95`), so a wrong outcome
  stops the test before the branch for that outcome runs.
- The loop, the sleeps, the recovery polls, the count, and the expected history are otherwise
  unchanged. The loop takes 18.2 s natively and under Gomad, against the 20 s run timeout. It took
  18.03 to 18.18 s natively before.

The change is not specific to Gomad and has no build tag.

Variant T makes the rejections independent of latency. It does not make the test independent of
time. Three timing assumptions remain, and each exists in the unchanged test:

- **Accepted heartbeats.** The last accepted heartbeat of a chain is sent four sleeps into it and
  must reach the server before the deadline. The margin is about 0.93 s natively on this host and
  exactly 1 s under `strict`. `chainDeadline` is read after the poll returns and the server's
  interval starts before it, so poll latency counts against this margin (2.5 to 43.9 ms measured
  for the first chain). Latency above the margin makes the server reject a heartbeat the test
  expects to be accepted, and the test fails at the per-heartbeat assertion.
- **Run timeout.** The workflow must complete within 20 s. The loop takes 18.2 s, 0.2 s more than
  before.
- **Recovery.** Each recovery waits for the 3 s start-to-close timer.

Removing the first assumption needs either a server clock the test controls or a loop that
heartbeats until the server rejects and derives the expected history from the observed counts.
The functional test environment provides only the real time source for the history service
(`common/resource/fx.go:152-153`). Neither design was built or evaluated.

## Corrections and owners

| Option | Change | Owner | Evidence here | Cost and risk |
| --- | --- | --- | --- | --- |
| T | Apply `fn105-d20-variant-t-test.diff.txt` to `tests/workflow_task_test.go`. Remove the `skip_subtests` entry only after the verification below passes. | Owner of `tests/workflow_task_test.go` | Strict leaf seeds 1 to 64, forward leaf seeds 1 to 16, `qualified` with exact replay under both ticks, native leaf 8 of 8, native suite 18 of 18 | Changes a Temporal test. The milestones' constraint against test rewriting applies unless the change is approved as a test correction, as R23 and R24 were. It raises the native margin from 10 ms to 100 ms and adds 0.2 s to the test. |
| A | Set `"clock_tick": "forward"` on `TestWorkflowTaskTestSuite` in `tools/gomad3integration/qualification/tests.generator.json` and regenerate `tests.json` | Gomad qualification configuration (`tools/gomad3integration`, owner `gomad3`) | Leaf seeds 1 to 32, suite seeds 11 and 17, `qualified` with exact replay | No Temporal change. The test still passes by an incidental margin, 1.4 ms at the smallest. The suite takes on the open forward-clock defect D16 describes. |
| P | Apply `fn105-d20-variant-p-server.diff.txt`: reject when the deadline is reached, not only when it is passed | History service owner, with the dedicated production review the milestones require | Strict and forward leaf seeds 1 to 16, `qualified` with exact replay under `strict`, native leaf, suite, and three unit tests | Changes production behavior at the exact deadline instant, which no native run here reached. No unit test covers that instant, so one would have to be added with a controlled time source. It also removes the native dependence on latency, because five 1 s sleeps always reach the deadline. |

Recommendation: option T. The cause is in the test's schedule, the server's rule is consistent
with its comment, and T is the only option under which neither rejection needs incidental elapsed
time, natively or under either tick. It keeps the latency ceiling for accepted heartbeats stated
above. Option A qualified the suite on darwin/arm64 in the runs here and needs no Temporal
change. It is the fallback if a test change is declined. Option P stays recorded for the
decision. In the darwin/arm64 runs here each option alone made the test pass; P was not qualified
under `forward`, and no option ran on linux/amd64.

Proposed next action, as open work in fn-105 for a subsequent decision. The skip stays in the
tracked manifest through step 5.

1. Decide on option T as a test correction, as R23 and R24 were decided. Apply the diff to
   `tests/workflow_task_test.go`.
2. Native: run the leaf with `-count=20` and the suite, and require every run green.
3. Gomad: generate a scratch manifest with the suite's skip removed (`fn105-d20-gen-manifest.py`)
   and run `qualify-set`. Require `qualified` on seeds 11 and 17 at the manifest's repeat count
   under `strict`, and a traced run with exact choice-tape replay.
4. Run the leaf on seeds 1 to 64 under `strict` and under `forward`, and require 64 passes each.
   The per-heartbeat assertion in the corrected test fails any run in which a rejection moves.
5. Keep the unchanged test under `strict` as the regression reproducer. It fails on every seed
   today with one rejection, and the instrumented run shows both chains accepted at exactly
   chain + 5.000000000 s.
6. When steps 2 to 5 pass, remove the `skip_subtests` entry from the generator manifest,
   regenerate `tests.json`, validate it with `qualify-set --check`, and qualify the suite from the
   tracked manifest.
7. Run the suite on linux/amd64 when a host is available. The decision must state whether that
   run gates step 6. The corrections D22 to D25 removed their skips on darwin/arm64 evidence and
   recorded that linux/amd64 was not run.

Whichever option the decision selects, including keeping the skip, the skip's recorded reason
should be replaced with the cause stated here.

## Limits

- No linux/amd64 run.
- Elapsed times are measured from the event time of the chain's first `WorkflowTaskScheduled`
  event. The server compares against `OriginalScheduledTime`, which it reads from the clock in a
  separate call immediately before the event time
  (`workflow_task_state_machine.go:411`, `353`). The two are equal under `strict`. Natively and
  under `forward` the stated margins can be too small by the time between the two readings.
- The statement that the server follows its intended semantics rests on the source, its comment,
  and the histories. The owner of the heartbeat timeout was not asked whether a heartbeat exactly
  on the deadline should be rejected. Option P is the correction if the answer is yes.
- The server starts a workflow task inline for a worker to which it returns `NotFound`, and
  recovery then waits for that task's start-to-close timeout. The test's expected history pins
  this behavior. Whether it is intended was not assessed.
- Variant T ran under `forward` on 16 leaf seeds and under `strict` on 64. Variant P ran on 16
  leaf seeds per tick and was not qualified under `forward`. Neither ran in the full `./tests`
  set, and variant P ran against three unit tests only.
- Variant T's native evidence is 8 leaf runs and 2 suite runs on an otherwise lightly loaded host.
  No native run under host load exists for any variant.
- Option A was verified through scratch manifests. The generator, the generated manifest, and the
  full `./tests` set were not run with it.
- The native control with 995 ms sleeps ran three times.
