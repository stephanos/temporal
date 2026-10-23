# fn-228.4 re-review: 510b0a2f12 and 22c20a7eae

## Verdict: SHIP

All twelve findings from `review.md` are fixed, and I verified each one, by probe or by reading the code. The re-review found two new minors and no major:
- a flaky new unit test;
- a teardown fault that is classified by timing rather than by which task the Model answered.

Neither changes a verdict about the system. Both are small; fix them before closing if cheap, otherwise file them.

## What I ran

All probes went in through `go test -overlay` from the session scratchpad (`probe/v2/`); the repo is untouched.

| Check | Result |
| --- | --- |
| Gate probe v2 (finding 1) | 3 runs: three jittered pollers with instant empty polls, one task, 6000 events per run. 0 reads while a poll or task was held, no hang, the callback delivered once, Close in 0.15–0.30 ms. Before the fix, 3 of 3 runs hung. |
| Runner redaction probe (finding 3) | A refused Answer is decoded, redacted, re-encoded and marked redacted, with no token left. Undecodable bytes are removed and marked. Non-matching paths leave the bytes unchanged. |
| `act` and `cmd/umpire` under `-race -count=3`, with the gate probe (CGO_ENABLED=1) | ok |
| `act` `-count=10` | ok |
| Boundary tests | ok |
| Runner tests | ok in 50 of 51 runs (see note 3) |
| gofmt | clean |
| `make umpire-check-model-slice` | rc 0, 36 s, 26 passes |

Live timing from the slice log, umpire Runs only:

| | Before | After |
| --- | --- | --- |
| Terminate to ShutdownWorker | 1.00x s every time | 1–3 ms, Terminate first |

In HostRefusal, Terminate (27.750) precedes the released task's RespondActivityTaskFailed (27.754), and the subtest asserts TERMINATED.

## Findings from review.md

| # | Finding | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Gate stale outcomes | Fixed | Each `window` has its own outcome channel and admits at most once (`admit` clears `open`). `closeIdle` decides from `w.admitted` under the lock. At most two outcomes ever reach a 2-slot channel, so sends never block or drop. Probe v2 is clean; `TestAnOutcomeCountsOnlyInTheWindowThatAdmittedItsPoll` and `TestNoReadWhilePollsEndAtOnce` cover it. |
| 2 | Close never released held polls | Fixed | `gate.stop()` refuses held polls with `errStopped` and cancels the admitted one (`context.AfterFunc(g.ctx, cancel)`, stopped after the invoke, so no leak). Live spacing went from 1 s to 1–3 ms. Close runs inline; the unjoined goroutine is gone. |
| 3 | Redaction missed refused calls | Fixed | `redactEvent` and `refusedCall` cover all five methods (`session.go:72-117`). `Run.world` and the Reset request are separate clones, and the world is redacted explicitly. Probe and `TestAHostRefusedRunRedactsItsEvidenceAndIsTerminatedByClose` pass. |
| 4 | Missing wait tests | Fixed | Every wait now has a test, each ending at its timeout and at Close; empty-poll resume, Close's Terminate and the out-of-range status are covered too. One new test is flaky (N1). |
| 5 | Invented worker responses | Fixed, per the conductor's decision | Refused, unclaimed and abandoned tasks `hold` until `released`, which Close closes only after Terminate. The live log shows the late response after Terminate, and HostRefusal asserts TERMINATED. See N2 for the remaining classification gap. |
| 6 | Weak live assertions | Fixed | The replay must print the same refusal line as explore. Cancellation asserts request 2 is Cancel and Accepted, and a `Rejected(NotFound)` after completion. |
| 7 | Boundary too broad | Fixed | An explicit allow-list; the guest is allowed only in `_test.go`. New negatives cover `umpire.local/model` and `umpire.local/temporal`. |
| 8 | Temporal prose in model/umpire | Fixed | The README now reads "registers the Drivers its build links". |
| 9 | Configuration | Fixed | `resolvable` walks the registry and the field descriptors. `MinWorkerPollTimeout` is 2 s. |
| 10 | Unanswered Start leaked | Fixed | A sent Start (stats `OutHeader`) with no readable answer is terminated by activity ID. An unreachable endpoint sends nothing, so it triggers no termination. |
| 11 | Zero instruction fields | Fixed | `validStart` refuses a zero poll interval, backoff or read timeout. |
| 12 | Unsendable reply fields | Fixed | `sendable` refuses any field besides the returned one and `task_token`, `namespace` and `identity`. |

## Regression hunt

I checked the new Close ordering, the per-window channels and the redaction re-encode for deadlocks, held tasks that are never released, and goroutine leaks.

- **Close ordering** (`gate.stop`, exchanges, Terminate, release, `worker.Stop`, teardowns):
  - No deadlock. `released` closes before `Stop`, so held activity bodies return before the SDK waits on them.
  - `workerMu` is taken only after the exchanges are joined.
  - An exchange left open at Close may issue one more Describe after its admitted poll is canceled. That is harmless: its context is canceled right after.
- **Windows:**
  - `finish` and `took` act only when `held == w`, so the double `finish` from `callback` and `deliver` is idempotent and cannot clear another window.
  - A task taken in a window whose `work` already timed out stays held until `deliver` returns after release. That is the intended hold.
- **Redaction:**
  - With the Temporal Driver, `opts.Redact` is always set, even with no paths. Then nothing matches and the bytes are unchanged (probe case `nomatch`).
  - Undecodable bytes cannot occur in practice: the Runner produced them from a typed request.

## New findings

### N1. minor: `TestCloseRefusesHeldPollsAndCancelsTheAdmittedOne` is flaky

**Where:** `common/testing/testpilot/temporal/act/worker_test.go:246-264`.

**Cause.** Two pollers race for the one window: `poll(…, nil)`, which answers empty at once, and `pollWith(…, hang)`. The test assumes the hanging poll is the one admitted. When the instant poll wins, it returns nil (code OK), and `require.Contains([]codes.Code{Canceled}, …)` fails with `[]codes.Code{0x1} does not contain 0x0`.

**Reproduced:**
- 4 failures in 200 runs under `-race`;
- 1 failure in a `-race -count=2` run under load;
- 0 failures in 300 runs without race.

This contradicts the handoff's clean `-race -count=5`; that run was simply lucky.

**Fix:** start the hanging poll first and wait until `d.gate.held != nil` before starting the second poll. Or make both pollers `hang`.

### N2. minor: a late Model-answered response can be recorded as a teardown fault

**Where:** `act/worker.go:387-399`.

**Cause.** `observeResponses` faults any successful RespondActivityTask* that completes after `released`. It never checks whether the Model answered that task.

**Failure scenario.**
1. A Run ends right after an Answer: the event budget is spent, or the next step refuses.
2. Close terminates and releases while the SDK's RPC carrying the Model's own reply is still in flight.
3. The server applied that RPC before the Terminate, so the reply returns OK.
4. It is recorded as "the SDK completed a task after the Run", and an Explained Run becomes Refused.

The window is narrow: once a final read has been explained, the server has already applied the response.

**Fix:** keep the task tokens of held (unanswered) tasks, and fault only responses whose request `task_token` is among them. Add a test where an answered task's response completes after release.

### Notes

1. **Close now runs unbounded in the Runner.** It joins rather than times out; that was the fix asked for. Its parts are bounded: `CloseTimeout` for Terminate, 1 s for `WorkerStopTimeout`, and exchange contexts. The one wait without its own bound is an exchange blocked in the SDK's `worker.Start()`, which relies on the SDK's own dial and RPC timeouts. No action needed.
2. **The handoff is accurate otherwise.** It reports Terminate and ShutdownWorker about 1 ms apart; I measured 1–3 ms.
3. **An unattributed Runner failure.** `model/umpire/hosts/go/umpire/runner` failed once, and I did not capture the output. It passed in 50 later runs, including under load. I cannot attribute it to this range.
