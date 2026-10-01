# Gomad D16 forward-clock poll deadline investigation

Assessment date: 2026-10-01.

Under `clock_tick: forward` the amount by which `time.Now` runs ahead of the timer clock grows
with every read and is never reduced. By the time
`TestStandaloneActivityTestSuite/TestStartDelay/UpdateWhilePaused_AfterWindow_ExtendsDispatch`
polls, about 650,000 reads have pushed that lead to about 0.33 s. Every context deadline computed
as `time.Now` plus a duration fires later than intended by the lead, and each gRPC hop re-derives
the deadline and adds the lead again. The frontend hop, the matching hop, and matching's
shortened child context add three leads to the server side. Once they sum to more than the 1 s
that matching reserves for returning an empty poll, the client's 3 s deadline fires first and the
test sees `context deadline exceeded`. The test, the 1 s budget, and gRPC deadline propagation
behave as they do natively. The correction belongs to the Gomad forward clock. The proposal below
is one clock for `time.Now` and timers, as fn-103 specified, with two partial fallbacks and an
accepted-limitation alternative. None is implemented or verified, the skip stays, and the fix
remains open work in fn-105.

This is the receipt for
[D16](../../../.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.16.md) (fn-105 R16).

## Scope and identities

| Item | Value |
| --- | --- |
| Repository revision | `7ef3800a2c` plus the shared uncommitted working tree |
| Toolchain | go1.27.1, build key `8d28bd4486f0b6300e8d25efd4caf8cb6ccbf000e96dbd26b1d8f53bf5f251bc` |
| Runner | `sha256:4dce3aed3599865194224c2b4219d0f13458585c8d849da10ffbc21a79db5222` |
| I/O profile | `gomad3-deterministic/v1`, implementation `sha256:9cd0cff9…ac7c0` |
| Suite target | `sha256:00afe9f408a65624163ee4026455419677bfc23bba72efbbe6dfe3cc4553149b` (all unmodified-tree runs) |
| Host | darwin/arm64, macOS 26.6.2 (25G83), 8 cores |
| Native Go | go1.27.0 darwin/arm64 |
| gRPC | `google.golang.org/grpc` v1.83.2 |
| linux/amd64 | Not available. Nothing below ran there. |

Evidence files are under `.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/` with the
prefix `fn105-d16-`.
[`fn105-d16-evidence.json`](../../../.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d16-evidence.json)
indexes them and records every command, identity, and per-seed digest. No tracked product, test,
runtime, or manifest file was edited. The skip was lifted by running `gomad explore` without a
`-test.skip` argument. The deadline probes ran in a scratch copy of the source tree that was
deleted afterwards (`fn105-d16-instrumentation.diff.txt`).

## Reproduction

Gomad suite runs use `gomad explore` with the manifest entry's build tags, closure mode, schema
mount, `-test.parallel=8`, and `-test.v`. The pattern is `^TestStandaloneActivityTestSuite$`, so
the skipped subtest runs with the rest of the suite. No run hit a wall watchdog or an
infrastructure error.

| Run | Tick | Seeds | Outcome of the subject subtest |
| --- | --- | --- | --- |
| Suite, unmodified tree | forward | 11, 17 | Both pass. Seed 11 passes with about 20 ms to spare (derived from the logged timeout). |
| Suite, unmodified tree | forward | 1 to 10, 12 to 16, 18 to 24 | 17 pass. Seeds 5, 12, 20, 21, 24 fail with `context deadline exceeded` at `activity_standalone_test.go:8495`, and nothing else in the suite fails. |
| Suite, unmodified tree, rerun with a 5 minute execution watchdog | forward | 5, 11, 12, 17, 20, 21, 24 | Seeds 5, 12, 20, 21, 24 fail, 11 and 17 pass. Stdout and stderr digests equal the first runs for all seven. |
| Suite, unmodified tree | strict | 5, 11, 12, 17, 20, 21, 24 | The subtest passes in exactly 5.00 s on all seven seeds. The suite fails on its timestamp-tie subtests, which is why it runs under `forward`. |
| Leaf alone, probed copy | forward | 11, 17 | Both pass. The child deadline is 989.5 ms before the client's. |
| Native leaf, `-count=3` | wall clock | none | 3 of 3 pass in 5.01 to 5.06 s. The probed copy receives the empty response with 998.5 to 998.8 ms left. |
| Native suite | wall clock | none | Passes, no failing subtest |

The recorded seed-11 timeout does not reproduce on this tree. The record predates the D14
lock-profile fix (toolchain `8d28bd44`) and later edits to the test tree, and the older toolchain
was not rebuilt for this investigation. Seed 11 now reaches the poll with a lead of about 326 ms,
just under the one-third second at which the outcome flips.
The same failure, with the same error at the same line, reproduces on five of the 24 seeds tried.
Which seeds fail is a property of the schedule, the toolchain, and the test selection. The
recorded classification "seed 11 fails, seed 17 passes" described one toolchain only.

The frontend logs the timeout it observes for every long poll below 10 s
(`common/util.go:648-656`). For the subject's 3 s poll it logs these values on the unmodified
tree.

| Tick | Seed | Frontend-observed timeout | Subtest |
| --- | --- | --- | --- |
| strict | all seven | 3 s | pass |
| native | none | 2.99985 to 2.99994 s | pass |
| forward | 17 | 3.5566 s | pass |
| forward | 11 | 3.6527 s | pass |
| forward | 20 | 3.6762 s | fail |
| forward | 24 | 3.6799 s | fail |
| forward | 21 | 3.6829 s | fail |
| forward | 5 | 3.6840 s | fail |
| forward | 12 | 3.6859 s | fail |

The excess over 3 s is two leads (the client's and the frontend's). The outcome flips between
0.6527 s and 0.6762 s of excess, which is a lead of one third of a second.

## Trace

The probes print, at seven points, the ticked `time.Now` reading, the lead (`time.Now` minus the
timer clock, measured as `-time.Since(time.Now())`), the timer clock, and the context deadline. A
deadline that carries a monotonic reading fires when the timer clock reaches the deadline's own
value. Times are virtual seconds since the target's start. The failing column is seed 5 under
`forward`. The control column is seed 11 under `strict`
(`fn105-d16-trace-forward-suite.txt`, `fn105-d16-trace-strict-suite.txt`).

| Step | Forward, seed 5 (fails) | Strict, seed 11 (passes) |
| --- | --- | --- |
| Test creates the 3 s poll context | timer clock 26.3839, lead 0.3341, deadline 29.7180 | timer clock 26.31, lead 0, deadline 29.31 |
| Frontend handler context (gRPC hop 1) | deadline 30.0521, 0.3341 after the client's | deadline 29.31 |
| Matching handler context (gRPC hop 2) | deadline 30.3863, 0.6683 after the client's | deadline 29.31 |
| Matching child context, 1 s budget | deadline 29.7206, 2.5 ms after the client's | deadline 28.31, 1 s before the client's |
| Timer clock advances while all goroutines wait | to 29.7180, the client deadline | to 28.31, the child deadline |
| First to fire | client context, `context deadline exceeded` | matching child context |
| Matching poll returns | context canceled, 2.5 ms before its child deadline | empty response at 28.31 |
| Frontend | matching call fails with `context canceled` | returns the empty response with 1 s left |
| Test | `require.NoError` fails | empty response with 1 s left, passes |

The other probed runs fit the same arithmetic.

| Run | Lead at the poll | Sum of the three server-side leads | Margin, child before client |
| --- | --- | --- | --- |
| Forward suite, seed 5 | 334.1 ms | 1002.5 ms | -2.5 ms, fails |
| Forward suite, seed 12 | 326.1 ms | 978.7 ms | 21.3 ms |
| Forward suite, seed 11 | 319.3 ms | 958.2 ms | 41.8 ms |
| Forward suite, seed 17 | 275.4 ms | 826.4 ms | 173.6 ms |
| Forward leaf alone, seeds 11 and 17 | 3.4 ms | 10.5 ms | 989.5 ms |
| Strict suite, seeds 11 and 17 | 0 | 0 | 1 s |
| Native leaf | none | 76 µs and 179 µs of transit | 999.8 ms (the response reaches the client with 998.5 to 998.8 ms left) |

The probes add `time.Now` reads, so a probed run is a different target. Seed 12 fails on the
unmodified tree and passes probed with 21.3 ms to spare.

In seven of the eight probed Gomad runs the poll is also forwarded from a child partition to the
root (forward suite seed 5 is the exception). The forwarded poll gets its own, earlier child deadline and returns first. The forwarding
partition still waits for its own child deadline, so the first partition's child deadline decides
the outcome in every run.

## Cause

Each step below is a source fact. Go references are under the go1.27.1 source, gRPC references
under `grpc@v1.83.2`.

1. `gomadTimeNow` returns `faketime + gomadClockTickOffset` and adds a draw of 1 to 1024 ns to
   the offset on every call (`tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go:163-170`).
   Nothing ever subtracts from the offset. The mean draw is 512.5 ns, so 334 ms is about 652,000
   reads.
2. Timers and `runtime.nanotime` read `faketime` alone (`go1.27.1.patch`, `time_nofake.go`
   hunk). `time.Until` and `time.Since` read `runtimeNano` for a reading that carries a
   monotonic value (`time/time.go:1227-1243`).
3. `context.WithTimeout` computes `time.Now().Add(timeout)` (`context/context.go:704-705`) and
   arms its timer for `time.Until(deadline)` (`context/context.go:645,653`). Under `forward` that
   duration is the timeout plus the current lead, so the timer fires late by the lead.
4. The gRPC client sends `grpc-timeout` as `time.Until(deadline)`
   (`internal/transport/http2_client.go:609-613`), rounded up to a microsecond for values of this
   size (`internal/grpcutil/encode_duration.go:27-50`). The server calls
   `context.WithTimeout(ctx, timeout)` (`internal/transport/http2_server.go:536`). The server
   deadline is therefore the client deadline plus the lead at the server's read. The same happens
   again on the frontend-to-matching hop.
5. Matching shortens the poll with `contextutil.WithDeadlineBuffer(ctx, longPollInterval,
   returnEmptyTaskTimeBudget)` (`service/matching/matching_engine.go:3110`,
   `service/matching/physical_task_queue_manager.go:42`, `common/contextutil/deadline.go:34-38`).
   It computes `time.Until(parentDeadline) - 1s` and passes it to `context.WithTimeout`, which
   adds the lead a third time.
6. The child deadline is the client deadline, minus 1 s, plus the three leads. When the leads sum
   to more than 1 s the idle scheduler reaches the client deadline first. The client cancels the
   stream, the frontend and matching contexts are canceled, and no empty response is sent.

Natively step 4 adds the transit time between the client's read and the server's read (76 µs and
179 µs measured), step 5 adds nothing measurable, and the 1 s budget absorbs both. Under `strict`
every read in a busy stretch returns the same instant and the three additions are zero.

The skip's recorded reason says the server deadline lands "a few microseconds later". That is
the size of the lead in a fresh process (the fixture measures 2.3 µs at the first hop with no
prior reads). In the suite the lead is five orders of magnitude larger.

## Minimal reproducer

`fn105-d16-fixture-main.go.txt` is a standard-library program of one file. It makes `BURN`
calls to `time.Now`, creates a 3 s client context, re-derives the deadline twice the way gRPC
does, creates the 1 s budget child, and waits for whichever context fires first. It exits 1
when the client deadline wins.

```sh
# in a directory holding main.go and the retained go.mod
go run . 700000
env -u GOMADSEED -u GOMAD3_CHILD_SEED gomad explore --seeds 11,17 --parallel 2 \
  --clock-tick=forward --keep-successes=all --success-limit=2 --success-bytes=1GiB \
  --on-failure=all --artifacts <scratch> go-run . -- 700000
```

| Clock | BURN | Seeds | Lead | Margin, child before client | Exit |
| --- | --- | --- | --- | --- | --- |
| native | 0, 600000, 700000 | none | none | 999.9998 ms | 0 |
| strict | 0, 600000, 700000 | 11, 17 | 0 | 1 s | 0 |
| forward | 0 | 11, 17 | 1.6 µs, 1.2 µs | 999.99 ms | 0 |
| forward | 600000 | 11, 17 | 307.1 ms, 307.6 ms | 78.6 ms, 77.2 ms | 0 |
| forward | 700000 | 11, 17 | 358.4 ms, 358.9 ms | -75.3 ms, -76.8 ms | 1 |

Logs are `fn105-d16-fixture-gomad.log` and `fn105-d16-fixture-native.log`.

## Classification

The gRPC and context behavior is ordinary. gRPC propagates a deadline as a relative timeout, so a
server deadline always lands after the client's by the time between the two clock reads, and
Temporal's 1 s budget exists to absorb that. The test is sound. Its 3 s timeout is above the 2 s
long-poll minimum (`common/constants.go:42`), and natively the child deadline is 999.8 ms before
the client's.

The `forward` contract in [`tools/gomad3/README.md`](../../../tools/gomad3/README.md) documents
the single-deadline effect. It states that a deadline computed from `time.Now` "expires later
than a timer armed for the same duration, by the offset accumulated when it was computed". The
failure is therefore inside the letter of the documented contract. The contract does not state
that the offset is unbounded, and the overlay comment at `gomad.go:136-137` describes the draw as
"far below any timer a test would set". That holds for one draw and fails for their sum. The same
README paragraph also states that `time.Since` against a monotonic reading reports less than a
second `time.Now` would. By that rule a 0.33 s lead makes `time.Since(start)` after a 1 s sleep
report 0.67 s (derived, not measured), and `go test` prints negative durations in the forward
suite runs (`--- FAIL: TestStandaloneActivityTestSuite (-0.38s)`).

The defect is the unbounded lead, and this test is one instance of it. By the arithmetic above,
an empty long poll that passes through the same three derivations in a `forward` workload fails
once about 650,000 `time.Now` reads have happened, whatever the client timeout, because the
budget is a fixed 1 s. The other six `forward` suites are expected `qualified` in the generated manifest. This investigation did not
run them or measure their leads.

## Owner and proposed correction

Owner: the Gomad runtime clock (`tools/gomad3/toolchain/runtime`, `gomadTimeNow`, and the clock
that timers and `time.Since`/`time.Until` read). No change is proposed to the test, to Temporal's
budget, or to gRPC. Widening the test's timeout would not help, because the margin depends on
the lead and not on the timeout.

The forward policy was specified in fn-103 with "bounds on per-draw and cumulative drift" as part
of its evidence, and with the note that "timers may come due while work is runnable"
(`.flow/specs/fn-103-gomad-seeded-virtual-clock-ticks.md`, Edge Cases). The first implementation
advanced `faketime` itself. It "broke the process-simulation time transport (runner failures in
functional suites)", and the follow-up moved the advance into the separate offset
(`.flow/tasks/fn-103-gomad-seeded-virtual-clock-ticks.1.md`, Done summary). The offset design has
a per-draw bound and no cumulative bound. D16 is the consequence.

Every option below keeps what `forward` exists for. Each `time.Now` still advances by a seeded
1 to 1024 ns, so consecutive readings never tie. Every option that changes the runtime changes
every `forward` execution identity. None has been built or run.

**Option 1, proposed. One clock.** Add each draw to the process virtual clock itself, as fn-103
specified, so `time.Now`, `time.Since`, `time.Until`, timers, and context deadlines read one
value. No lead exists. A deadline of `time.Now` plus 3 s arms a 3 s timer, a gRPC hop adds only
the draws made in transit, as native transit time does, and `time.Since` cannot go negative. In
the fixture the 700,000 reads become 0.36 s of elapsed virtual time before the client context is
created, and by the arithmetic above the child fires 1 s minus a few microseconds before the
client. Costs and unknowns:
virtual time advances while work is runnable, so a timer can come due without an idle transition
and is delivered at the runtime's next timer check; the README sentence that only `time.Now`
observes the advance is rewritten; and the simulation time transport failure that the first
implementation hit must be reproduced and resolved first. This investigation did not reproduce
or diagnose that failure, so the feasibility of option 1 is not established.

**Option 2, partial. Offset-aware `time.Since` and `time.Until`.** Keep the separate offset, let
`time.Since` and `time.Until` read the timer clock plus the offset without drawing, and keep
timer arming (`time/sleep.go:24-35`, `when`) on the timer clock. This removes the derivation
lag, because `time.Until(time.Now().Add(d))` is `d` again. It leaves a second disagreement. A
context arms its timer once, and every later draw moves the user-visible clock toward the
deadline without moving the timer. The deadline passes on the user-visible clock before
`ctx.Done()` closes, by the draws made during the context's lifetime. In the seed 5 trace the
lead grew from 334.1 ms to 344.1 ms during the 3.3 s wait, so that window would have been about
10 ms. Inside it `time.Until(deadline)` is negative while `ctx.Err()` is nil, and a gRPC client
call started there returns `DeadlineExceeded` without sending
(`internal/transport/http2_client.go:609-612`). Option 2 also needs a change in package `time`,
which needs a decision against the patch policy. It would let this subtest pass and is not a
complete correction.

**Option 3, partial. Absorb the offset on idle advances.** When the idle scheduler advances
`faketime` by a delta, subtract `min(offset, delta)` from the offset. `time.Now` stays monotonic
because the sum cannot decrease. This bounds the lead only by the draws made since virtual time
last advanced, and it does nothing for a deadline built inside the busy stretch that produced the
lead. The retained fixture burns its reads and builds its contexts in one busy stretch, so it
would still fail under option 3. Readings taken across an idle advance would also grow by less
than the advance. Whether the suite would pass under option 3 was not measured, and the defect
stays in place.

**Accepted-limitation alternative.** Keep the skip, and state in the README that the lead grows
by about 0.5 µs per `time.Now` read without bound, with the 650,000-read figure for a 1 s budget
across three derivations, and that fn-103's cumulative-drift bound does not hold.

Proposed next action: reproduce the fn-103 simulation time transport failure with `faketime`
ticks, decide between option 1 and the accepted limitation on that evidence, and treat options 2
and 3 as fallbacks with the residuals stated above.

## Regression and qualification criteria

A correction is verified when all of these hold on darwin/arm64, and on linux/amd64 when a host
is available.

1. The retained fixture becomes a runtime conformance fixture. Under `forward` with
   `BURN=700000` on seeds 11 and 17 the child deadline fires first, with a margin within 1 ms of
   1 s.
2. A second fixture burns reads after creating a context and asserts that once
   `time.Until(deadline) <= 0` the context becomes done without a further idle advance of the
   clock. Option 2 fails this by construction.
3. The existing `forward` conformance holds
   (`tools/gomad3/internal/gomadtool/conformance/runtime_clock_tick.go`). Consecutive reads
   advance by 1 to 1024 ns, repeat per seed, and differ across seeds. A new check asserts a
   stated bound on the difference between `time.Now` and the timer clock, which is zero for
   option 1.
4. `TestStandaloneActivityTestSuite` runs under `forward` with this skip removed and qualifies on
   seeds 11 and 17 at repeat 2, and also on seeds 5, 12, 20, 21, and 24, which fail today. The
   subtest's assertion that the activity does not dispatch within the extended delay is
   unchanged. The timestamp-tie subtests stay unskipped.
5. The other six `forward` suites and the simulation workloads that exercise the time transport
   requalify, because their identities change.
6. The README contract paragraph and the fn-103 identity evidence are updated with the change.

## What stays open

Nothing is fixed. The skip in `tools/gomad3integration/qualification/tests.generator.json` stays
with its current reason, which understates the lead and names seed 11. The choice between
option 1, the partial options, and the accepted limitation is a decision for fn-105, and
option 1's feasibility against the simulation time transport is unverified. The correction, its
conformance fixture, and the requalification are open work after that decision. linux/amd64 was
not run.
