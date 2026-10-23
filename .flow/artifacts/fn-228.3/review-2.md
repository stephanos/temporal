# fn-228.3 re-review: e98ec44df6 and 5694e9732c

Verdict: **NEEDS_WORK**

Two small, verified problems remain. Both are in code that this round added or changed.

## What I ran
- Runner tests: `CGO_ENABLED=0 -count=3` ok; `-race -count=2` ok.
- `go vet` clean on the runner, on `-tags 'test_dep integration' ./tests/` and on
  tools/umpire/conformance. `proto-go.sh --check` rc 0.
- Throwaway probes in a scratch copy of the module (none in the repo):
  - the original tamper reproductions;
  - forged limits, empty Runs and question relabels;
  - interrupt during Start, and a late Start;
  - an anchored proto-go check on a scratch copy of model/ with stray and stale entries.

## Status of the nine findings

1. **Live gate broken by the deleted commands: resolved.** The four tests that built
   umpire-run, -replay and -assess are gone. `buildUmpireCommand` moved to its remaining
   umpire-ir-bridge user, and `go vet` passes on ./tests. Coverage mapping is in
   retired-coverage.md.
   Pre-existing claim: verified. baseline-four.log (base 0c7822bc43) and live-tests.log (after)
   fail with the same four identities and the same lines: the missing `model/ir/nexus-workflow.json`,
   the missing `model/temporal/features`, "malformed at contract: invalid or duplicate correlated
   transition", and "optional.none() dereference". Both missing paths are absent at the base and at
   HEAD, and this round does not touch those four tests' files.
   Nit: stale prose naming the deleted commands remains in tests/testpilot_signature_test.go:21,29,
   tests/testpilot_live_case_test.go:52 and tests/testpilot_nexus_control_case_test.go:38.
2. **Replay laundering: resolved for my reproductions.**
   - Truncated `events[:5]` labelled EVENT_BUDGET: Refused, "termination does not fit".
   - The good Run cut at 2, 5 or 7 and relabelled PROPOSAL_BUDGET or EVENT_BUDGET: Refused.
   - DriverFault inserted before the Unexplained judgment: Refused, "event 6 follows the event
     that ended the Run".
   - Appending a call after QUIESCENT: Refused.
   - Two residual cases are inherent and acceptable. A truncation whose `limit` is set to its
     own length is `proto.Equal` to a real EventBudget=5 Run (verified). A truncation ending at
     a Proposal and labelled QUESTION equals a real question Run over that prefix. The digest
     key, not replay, protects published Runs against forgery.
   - But see new finding B (empty Runs).
3. **Event budget overshoot: resolved.** Budget 2 now records 2 events and budget 4 records 4
   (probe); the tests cover budgets 1, 2, 4 and 5 and the limit.
4. **Reset refusal not replayable: resolved.** It replays to the identical Refused verdict.
   `TestRefusedResetReplays` also checks the world on the decoded request.
5. **Reduce: resolved.** The recorded Run is replayed first; altered protocol versions and seeds
   are rejected. Non-reproduction is Refused "not reproduced within n candidates", nothing is
   published, and the command exits 2 (tested).
6. **Start bound: resolved as specified, but it introduced regression A.** The Start doc now
   says Start returns once the request is sent, and Start is bounded by the read timeout
   (`TestStartMustReturnOnceSent`).
7. **Deferrals not recorded for .6: partly resolved.** The canary and testpilot/publish
   deferrals are written in .flow/tmp/fn-228.3/retired-coverage.md, which marks them "please
   file". The .6 task file still has neither. The coordinator should copy them into
   fn-228-runner-and-run-the-one-go-host-loop.6 with flowctl.
8. **proto-go exclude too broad: resolved.** It now diffs each generated package and refuses any
   other directory except `umpire/runner/`. Scratch checks: clean rc 0; stray directory rc 1;
   stale generated file rc 1; extra file inside a generated package rc 1.
   Nit: a stray *file* directly under `umpire/hosts/go/umpire/` now passes (rc 0). The old
   whole-tree diff caught it.
9. **Guest trap loses the partial Run: refusal accepted.** A trap leaves the instance unusable,
   the failing call has no response, and the event schema cannot express a trap for replay. An
   exit-3 error is the honest outcome, and it keeps published Runs to what the Model answered.

## Remaining findings

A. **major (regression): an exchange whose Start returns late, or is interrupted, is not joined before the Run returns.**
   session.go:362-390 (the late path in `start` closes the exchange from a background goroutine);
   modes_test.go:48.
   Scenario (verified, 48 of 50 iterations): the parent context is canceled while `Start` runs,
   for example on Ctrl-C under `umpire explore`. `startCtx.Done()` wins the select and the inner
   non-blocking receive misses Start's result. `exchange` returns `context.Canceled`, and the
   exchange is still open when `Runner.Run` returns. The command then calls `os.Exit`, so the
   Driver's owned work is never closed. In batch modes, a slow Start likewise leaves the previous
   exchange alive while the next Run starts.
   This breaks "Close cancels and joins owned work". The "interrupted" test was changed to
   cancel 10 ms after Start (`time.AfterFunc`), which hides the problem.
   Fix: in the late path, wait synchronously, at most one more read timeout, for the Start
   goroutine; Start must honor its now-canceled ctx. Then close the returned exchange through
   `closeExchange` before returning. Restore the synchronous-cancel interrupt test and assert
   that the exchange is joined.

B. **major (small fix): replay crashes on, or accepts, a Run with no events.**
   session.go:123-126 (`verdictOf`); replay.go:15 (`admit` accepts zero events).
   Scenario (verified): a canonical Run with the right hash, protocol and seed, no events, and
   verdict Explained/QUESTION. Both `umpire replay --run` and `Runner.Run(ModeReplay)` panic
   with "index out of range [-1]" at `events[len(events)-1]`. The same empty Run labelled
   Explained/EVENT_BUDGET with limit 0 replays as Explained (0 == 0).
   No live Run is empty: the Reset, or its refusal, is always recorded, and budgets default
   to at least 1.
   Fix: have `admit` refuse a Run with no events ("a Run begins with its Reset"), or guard
   `verdictOf`. Add both cases to `TestReplayRejectsAlteredRuns`.

## Other checks this round (no finding)
- The new "only faults follow the ending event" rule matches the live loop. After a terminal
  Judge, Answer, Annotate or Propose refusal, or a fault, the loop records only close faults.
- `accepted` correctly finds a final Explained judgment for the last proposal even when an
  Answer or an intermediate poll follows it.
- `Explained.limit` is set only for the two budget terminations.
- The ctx passed to `Start` is now canceled when Start returns, as documented ("ctx bounds Start
  alone"). The Temporal Driver in .4 must not run the in-flight RPC on that ctx. The fake already
  detaches it.
