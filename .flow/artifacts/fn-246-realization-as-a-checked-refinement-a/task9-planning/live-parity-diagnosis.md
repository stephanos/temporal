# fn-256 live parity diagnosis

The failed gate reports raw/SDK judgment-path equality failure, while both seed-21 Runs report `verdict=explained`. The strongest supported explanation is one extra polling observation whose judgment repeats an existing complete witness prefix. The available evidence does not establish an extra activity attempt or a Model-vs-SUT refutation.

This was a bounded read-only diagnosis. I read systematic-debugging and diagnosing-bugs, inspected the existing failure log and source, and wrote only this report. I did not run Go, tests, builds, retries, cache mutations, source edits or Flow commands. Reproduction/minimization phases remain deferred under the conductor's explicit bounds. AGENTS.md:8 requires updating the Model independently and consulting the human on conformance failures; no Model, expected judgment, assertion or timing change is proposed here.

## Observed failure

Log supplied by fn2561_retirement:

`/home/agent/.cache/codex-build/umpire-milestones-20261024/fn2561-retirement/logs/live-tests.log`

- Lines 9754-9770 identify `tests/testpilot_model_slice_umpire_test.go:456`, seed 21. Expected raw flattened judgment path length is 103; actual SDK length is 105. The diff inserts one `Worker::Poll(1) Accepted` and one `Worker::RespondFailed { retryable: true } Accepted`.
- Lines 8356-8363 report raw seed 21, request 3 beginning at event 31, 46 total events, `verdict=explained termination=proposal_budget`.
- Lines 9739-9746 report SDK seed 21, request 3 beginning at event 32, 47 total events, the same explained termination. Requests 1 and 2 begin at events 1 and 3 in both Runs. Thus the extra event precedes request 3, inside the pending read for request 2.
- Parsing the two printed arrays yields raw/SDK counts of 36/37 for both Poll(1) and retryable failure; both have 11 Poll(2), 11 completion and 9 Cancel(NotFound) entries. This repeated material is already evidence that the flattened array does not count actual attempts.
- Both invocations pass `requireSupportedActivityRuns` before the equality assertion. That helper validates the narrow expected unheld-cancel refusal and requires all other Runs to be Explained (`testpilot_model_slice_umpire_test.go:248-263`). Seed 21 is explicitly Explained in the log.
- The raw and SDK Complete, Fail, RetryThenComplete, cancellation and unheld-scope subtests pass. The raw HostRefusal subtest passes. SDK HostRefusal lies after the fatal equality assertion and does not run in this failed invocation. Mutation controls, provisioning, unreachable-endpoint and replay checks pass. Do not describe this gate as fully passing.

## Mechanism and configuration

`testpilot_model_slice_umpire_test.go:153-161` concatenates `Explanation.Steps` from every judgment event. `judgmentPath` at 164-172 only collapses adjacent `Client::Cancel Rejected(NotFound)` strings. It leaves every repeated two-step retry prefix intact.

`model/umpire/protocol/runtime/src/runtime.rs:637-665` extends the retained prefix for every admitted observation, then calls `Self::explanation(&prefix)`. Lines 1012-1021 emit every retained prefix edge. A repeated observation with no new edges therefore returns the same full witness again. Concatenating those responses makes the parity assertion sensitive to observation count.

`model/temporal/feature/activity/temporal.rs:150-155` authors Describe polling at 10 ms, backoff 10 ms and read timeout 30 s. Its `polling` method at 269 continues while the activity remains open. `common/testing/testpilot/temporal/act/exchange.go:296-339` performs a real Describe, submits its observation, and waits the authored interval before the next read while ContinuePolling remains true. Actual RPC and scheduler duration affect the number of reads before terminal evidence.

The test's World fixes maximum attempts at 2, retry initial/maximum intervals at 1 s and start-to-close at 20 s (`world`, lines 63-80). It explores with seed 21, proposal budget 7, event timeout 60 s and run timeout 2 min. Raw and SDK use separate randomly named task queues/activity IDs in the same test cluster and execute sequentially. The seed does not pin RPC duration or scheduler timing.

Raw `raw.go:53-78` polls serially and follows each assignment; `selectedResponse` at 160 matches the program using the observed scenario and attempt. SDK `sdk.go:96-148` starts the native SDK worker with SDK defaults and a one-second stop timeout; `followTask` at 284-335 matches the same program with `activity.GetInfo().Attempt`. SDK is pinned to `go.temporal.io/sdk@v1.48.0`. Both obtain their program from the same guest contract, with namespace/queue tailored to their World. Neither executor's ordinary program lookup invents a new attempt.

The authored retry scenario chooses retryable failure for attempt 1 and completion afterward (`temporal.rs:554-555`). Existing Rust control `read_remains_pending_through_retry_and_delivery_until_terminal_evidence` in `model/temporal/realization/tests/proposal.rs:273` explicitly admits repeated retry-state Describe observations and expects the completed explanation to contain the full four-step witness. This is source evidence, not a test executed during this diagnosis.

## Attribution and remaining uncertainty

The test helper and runtime explanation function are preexisting code. `git blame` attributes concatenation to `d7a144ea4ff` (2026-10-23), path normalization to `6bf9e7e705d`/`22c20a7eaea`, and complete-prefix explanation to `00964fe9e40` (2026-10-25). Byte comparison of the root and retirement worktree found identical test, runtime, authored activity realization, read-loop, raw executor and SDK executor files. Earlier import-closure evidence excludes the removed six Go trees from these consumers.

This establishes preexisting mechanism, not a historically reproduced identical failure. No before-retirement failing run was obtained. Do not label the observed failure definitively preexisting or fixed by a green rerun.

Ranked explanations and discriminating evidence:

1. **Strongly supported inference:** SDK produced one extra Describe judgment during retry waiting, repeating the same two-step full prefix. Prediction: the saved Runs show identical semantic terminal witnesses and worker RPC attempts, with one extra Describe observation at the retry-state prefix. The one-event and two-flattened-step deltas fit precisely.
2. **Unconfirmed alternative:** another actual worker event or evidence delivery changed the trace. Prediction: per-event Runs show a differing worker RPC/task token/attempt or an additional admitted edge, rather than a repeated Describe prefix. The printed arrays alone cannot rule this out.
3. **Unconfirmed timing source:** SDK poller/handler scheduling, cluster load or real RPC latency shifts read sampling across the one-second retry interval. Prediction: repeat invocations vary read count while preserving the terminal witness. No scheduler measurements or repeated failures were collected here.

## Smallest next diagnostic

The retirement worker is already running its bounded unchanged gate retry; retain both failure and retry logs. No additional build or broad suite is needed for this report.

For the fn-246 owner, first preserve the two seed-21 `.binpb` Runs before `t.TempDir` cleanup on the next targeted reproduction of `TestTestpilotModelSliceThroughUmpire`. Existing `runObserved` loads them in memory at lines 136-148 but logs only summaries; the worker reports no known saved files from this failure. Capture only each event's sequence/request, observation RPC method and attempt identity, ContinuePolling, full explanation, and final verdict. Compare the request-2 segment and the final explanation. That distinguishes an extra sample from an actual extra attempt without changing any judgment rule, time bound, Model or assertion.

If a future instrumentation change is authorized, a failure-only artifact copy or event-summary log at the equality boundary is the narrow diagnostic seam. Keep the assertion failing while collecting evidence. A correction to the parity control's interpretation of complete-prefix explanations belongs to the fn-246 owner after confirming this evidence; a Model semantic change is not justified by this failure.

Evidence commands were bounded `sed`, `rg`, Python read-only extraction/counting of log strings, byte comparisons, `git log` and `git blame`. No claimed reproduction or passing gate result was generated by this agent.
