# fn-228.3 final re-review: 47e2d38822 (plus 12a9c38c71 for the .6 deferrals)

Verdict: **SHIP**

## What I ran
- Runner tests: `CGO_ENABLED=0 -count=3` ok; `-race -count=2` ok.
- `go vet` clean on the runner and on `-tags 'test_dep integration' ./tests/`.
  `proto-go.sh --check` rc 0.
- Every earlier throwaway probe, re-run in a fresh scratch copy of the module, plus a new probe
  for Close counts and goroutine leaks. Nothing was added to the repo.

## Finding A (an exchange from a late or interrupted Start was not joined): resolved
- **The 50-try interrupted-Start probe** (cancel inside `Start`) now leaves 0 of 50 exchanges
  unjoined when `Run` returns; it was 48 of 50 before. The test was restored as "interrupted
  while starting": 50 tries, synchronous cancel, and it asserts the join.
- **Close counts.** A counting exchange was run 20 times in each of these scenarios:
  - Start on time, then a read timeout;
  - late within the grace period (80 ms against a 50 ms read timeout);
  - late past the grace period (150 ms);
  - interrupted;
  - interrupted with an instant return.

  `Close` ran exactly once every time, so there is no double Close. In every case but "past
  grace" it ran before `Run` returned. In the past-grace case it ran once, after `Start`
  returned, from the background goroutine. That is the documented end state for a Start that
  ignores its canceled ctx, and it is recorded as the second fault.
- **Goroutines.** `runtime.NumGoroutine` was 2 before and 2 after all 100 Runs: no leak once
  `Start` returns. A `Start` that never returns still pins its goroutine; that is inherent and
  out of the Runner's control.
- **No double-close path.** In the late path `start` returns `(nil, true, nil)` after closing,
  so `exchange` neither closes again nor defers a close. In the in-time-but-parent-canceled path,
  `exchange` closes once before the deferred close is registered.
- **Fault ordering.** In the late path the first, terminal fault is "start did not return within
  the read timeout". Any close fault, or "start did not return after its context ended", follows
  it. That matches the replay rule that only faults may follow the ending event. All of these
  probe Runs replayed to the identical verdict, and `TestStartMustReturnOnceSent` replays the
  two-fault shape. On interrupt, no timeout fault is recorded, and the Run is discarded with
  `context.Canceled`.
- **Cost.** The worst-case wait in the late path is about three read timeouts: start, grace,
  then close. That is acceptable.

## Finding B (replay crashed on, or accepted, a Run with no events): resolved
- `admit` refuses a Run with no events, and `verdictOf` guards against it as well.
- Probes:
  - An empty Run labelled QUESTION: Refused "a Run begins with its Reset", with no panic. The
    same through `umpire replay` exits 2.
  - An empty Run labelled EVENT_BUDGET with limit 0: Refused.
- Both are in `TestReplayRejectsAlteredRuns`.

## Earlier reproductions (no regressions)
- **Tampering stays rejected.** Truncated and relabelled Runs are Refused with "termination does
  not fit". The fault inserted before the Unexplained judgment is Refused at event 6, and a call
  appended after QUIESCENT is Refused.
- **Budgets and Reset refusals hold.** Event budgets 2 and 4 are exact, and a refused Reset
  replays to the same verdict.
- **Inherent cases are unchanged.** A truncation whose limit is set to its own length, and a
  truncation at a Propose labelled QUESTION, equal genuine Runs. As noted last round, this
  needs no action.

## Nits: resolved
- **proto-go.** In a scratch copy of model/, the check now fails (rc 1) on a stray top-level
  file, a dotfile, a symlink to a generated directory, a stray directory and a stale generated
  file. A clean tree gives rc 0.
- **Stale prose.** No reference to umpire-run, -assess, -replay, -fuzz or -repeat remains under
  tests/.

## Finding 7 (deferrals not recorded in the .6 task): now resolved
12a9c38c71 adds to the .6 task: reduce the canary to lease, fencing and policy with
`runner.Publish`, and retire testpilot/publish and its eight importers.

No remaining findings.
