# fn-228.4 handoff: the Temporal Driver acts and observes; the live activity slice runs through `umpire explore`

All gates below passed; no Unexplained interleaving appeared.

## Commits (branch umpire, base 39d79ffa54)

| Commit | Subject |
| --- | --- |
| 9ddd74f15b | feat(temporal): a read instruction says how many worker tasks it admits |
| 780d249590 | feat(runner): a Driver may redact the evidence of its judged Runs |
| 9da5c6f44c | feat(testpilot): the Temporal Driver acts and observes for the Runner |
| ae2ae4a39a | feat(umpire): register the Temporal Driver with the umpire command |
| d7a144ea4f | test(umpire): the live activity slice runs through umpire explore |

The coordinator's 12f5645a68 and 8df7e6f142 landed on top while the required gate was running. I did not touch them.

## Acceptance

### The Temporal Driver implements runner.Driver without interpreting Temporal messages

The Driver is `common/testing/testpilot/temporal/act`.

**Unary calls.**
- **Exact bytes.** The proposal's bytes go out unchanged, through a pass-through codec, to the one WorkflowService method whose input is the request's message. The method is found from the generated service descriptor. The response's exact bytes are recorded.
- **Server status is evidence.** A status the server sent becomes the Observation's error. A call the server never ended (a stats-handler check for the trailer) is a Fault.

**Reads (`ObserveActivity`).**
- **Spacing.** Continued reads are spaced by `poll_interval_ms`. An unanswered read is retried after `backoff_ms`.
- **Read timeout.** If `read_timeout_ms` passes without a final Explained read, the Driver emits "read timeout without Model acceptance".
- **Only the Model's decision ends a read.**

**SDK worker (D2).**
- **Setup.** One in-process SDK worker per Run, with a dynamic activity and a lazy client.
- **Poll gate.** The gate admits a poll only after a read the Model asked to continue, and only `worker_tasks` times per read. Reads pause while that poll or its task is outstanding.
- **Callbacks.** A callback carries the actual task the poll returned. The reply goes back through the SDK: the one result payload of a completion, or the failure through `FailureToError`.

**Close.**
- **Cleanup.** Close stops the worker and terminates any activity the Driver started that is still open.
- **Parsing.** It reads the Start identity only for that cleanup.

**Redaction (D5).** Field paths from `UMPIRE_TEMPORAL_REDACT` are applied through the new optional `runner.Redactor`, which only takes effect when `Options.Redact` is nil.

**Evidence (unit tests in `act_test.go`):**
- the exact bytes sent;
- a server status as evidence;
- an unreachable endpoint as a Fault;
- `continue_polling` on a call that is not a read;
- reads repeating until accepted;
- the read timeout, the retried unanswered read, and the `MaxReadTimeout` cap;
- Respond rejections;
- the gate: no poll before a continue, reads paused while a task is unanswered, one task per instruction, and `worker_tasks=0` never admitting;
- the reply conversions and the redaction.

`runner_test.go` drives the real Temporal guest against a fake server.

**The Realization change (D2)** is `ObserveActivity.worker_tasks = 5`: 1 for `Worker::Poll(n)`, 0 for response reads.
- The activity Model and its behavioral goldens are unchanged.
- Exactly one snapshot changed (two lines of the `activity-realization-authored` expansion), and I accepted that one alone.
- The guest was rebuilt (sha256 23dceb5e… → d80dd567…); the freshness check passes.
- The Rust (`proposal.rs`) and Go (`proposal_test.go`) parity tests now pin the field.

### The live slice passes through `umpire explore`; the Run is the only artifact

`TestTestpilotModelSliceThroughUmpire` runs the command as a separate process, in an empty working directory, against the committed guest. The test also asserts that the guest's bytes equal `temporal.Guest()`. After each run the working directory is still empty, and the store holds only `.binpb` Runs, each matching a `published` line.

| Subtest | Result |
| --- | --- |
| Complete | Exit 0. The namespace was provisioned through `UMPIRE_TEMPORAL_CREATE` and deleted afterwards. One request shows a continued read, then an Answer, then a final read. The completion is explained. |
| Cancellation | An accepted Cancel is explained. |
| RetryThenComplete | Two Answers. The retry and the completion are explained. |
| Fail | The non-retryable failure is explained. |
| HostRefusal | Exit 2, with Answer refused UNSUPPORTED. No activity is left running. |
| UnreachableEndpoint | A DriverFault at event 2. Exit 2. |
| ReplayWithoutServer | A Run that includes a worker callback replays with exit 0, and the refused Run replays with exit 2, with no Driver configured. |

This covers the four tests retired in fn-228.3: the separate process with provisioning and cleanup, the unreachable endpoint, the recorded-and-replayed callback Run, and assessment (now replay).

### DriverFault and malformed-evidence controls

- A response that does not decode is a Fault before Judge. Through the Runner, the Run is Reset, Proposal(bootstrap), DriverFault: no Judgment is recorded, and the Run is Refused (`TestMalformedStartEvidenceRefusesTheRunBeforeJudge`).
- A task the worker took that is not the captured one is a Fault before Answer.
- An unreachable endpoint ends Refused, at three levels:
  - the Driver (`TestAnUnreachableEndpointRefusesTheRun`);
  - the command, exit 2 (`main_test.go`);
  - the live test.

## Gates

Logs are in `.flow/tmp/fn-228.4/`. Each gate ran alone.

**Focused:**
- `go test -tags test_dep -count=5 ./common/testing/testpilot/temporal/act/ ./tools/umpire/cmd/umpire/`: rc 0.
- The same with `-race -count=3` (CGO_ENABLED=1, this check only): rc 0.
- Runner `go test ./umpire/runner/`: rc 0.
- The Testpilot boundary tests: rc 0.
- `cargo test -p umpire-temporal -p umpire-temporal-realization-macros`: rc 0.
- The Temporal Go host module tests: rc 0.
- `proto-temporal-go.sh --check`: rc 0.

**Lint and vet:**
- golangci-lint v2.13.1 on `act` and `cmd/umpire`: 0 issues.
- `go vet -tags test_dep,integration ./tests/`: rc 0.

**Live:**
- `make umpire-check-model-slice`: rc 0, 23 s (`slice-gate.log`).
- The new test with `-count=3`: rc 0 (`live-count3.log`).
- `make umpire-check-model-gaps`: rc 0, 405 s, 2338 PASS lines (`gaps-gate.log`).

**Model gates:**
- `make model-check`: rc 0, 574 s (`default-gate.log`).
- `UMPIRE_BACKENDS=require make model-check`: rc 0, 1141 s (`required-gate.log`). It started before 12f5645a68 and ran on d7a144ea4f.

Not run, per the brief: `model-time`, review, `flowctl done`.

## Decisions taken in implementation

1. **Defaults.**
   - RPC timeout: 10 s.
   - Worker poll timeout: 5 s (the frontend refuses long polls under 2 s).
   - Close timeout: 10 s.
   - The SDK logger is discarded so that no payload reaches stderr.
2. **Configuration and cleanup exits.**
   - An invalid `UMPIRE_TEMPORAL_*` configuration exits 3 for explore, reduce and repeat; question and replay ignore it.
   - A failed cleanup of a provisioned namespace turns exit 0 into 3.
   - Address, namespace and task queue are all required.
3. **Close terminates the activities this Driver started.** NotFound and FailedPrecondition are ignored. Any other termination error becomes a recorded teardown fault.
4. **The boundary.** The Testpilot boundary now rejects `umpire.local/*` imports anywhere except `temporal/act`. Negative cases cover generic Testpilot and the Case Driver. The module map gained the `act` row.
5. **Test environment.** The live test builds its own environment: the activity settings plus the worker service, which the system namespace-delete workflow needs. It builds the command before the cluster starts.
6. **Coverage by seed.** Seeds are chosen with `runner.ProposalSeed`. Cancel is index 0 of at most two eligible actions, so `seed % 2` picks the path. That makes each World's coverage deterministic.

## Open

- **Pre-existing failures in `tools/umpire/ir`, unchanged by this task:** `umpire-gen-cases` without a caller, `TestModelNamesNoRetiredFrontEnd` (a hit in `model/framework/design`), `TestAdmissionChecksProgressOwnerStateAndBound`, and the totals fixtures under the retired `model/irgen`.
- **Pre-existing formatting issues:** gofmt/gci flag `common/testing/testpilot/correlated_facade_test.go` and `temporal/profile.go`.
- **For .6:** `TestTestpilotModelSlice` (the scripted host-level test) is kept, per D8. Fold `act` into the parent `temporal` package once the Case Driver is deleted, if wanted.
- **For fn-241:** it should rebase over the `temporal.rs` change in 9ddd74f15b.

## Review fixes (review.md, NEEDS_WORK)

All twelve findings are fixed; the guest is unchanged, so the gaps gate did not need to run again.

Commits:

| Commit | Subject |
| --- | --- |
| 510b0a2f12 | fix(runner): redaction reaches refused calls and the Run's own world |
| 22c20a7eae | fix(testpilot): the Temporal Driver's gate counts outcomes per window and Close joins its work |

### Majors

**1. Poll-gate race.**
- **Fix:** every gate opening is now a `window` with its own outcome channel. A window admits at most one poll, and only while no other poll or task is outstanding (`held`). `work` reads only its own window's channel, so an outcome counts only in the window that admitted its poll and is never dropped. On the timer, `closeIdle` decides under the gate lock from `w.admitted`, not from `owner`.
- **Deterministic regression:** `TestAnOutcomeCountsOnlyInTheWindowThatAdmittedItsPoll` is the reviewer's scenario: a poll admitted in the first window ends exactly as that window closes, then a later window opens.
- **Stress test:** `TestNoReadWhilePollsEndAtOnce` runs 200 reads against three jittered pollers making instant empty polls. The fake server records any read made while a poll is held; none was.

**2. Close joins owned work.**
- `gate.stop()` refuses held polls with Canceled and cancels the admitted poll through the gate's own context. Close calls it first.
- Close now runs inline, with no unjoined goroutine.
- `TestCloseRefusesHeldPollsAndCancelsTheAdmittedOne` covers it, and so do the Close cases of `TestEveryWaitEndsAtTheReadTimeoutAndAtClose`.
- Live: Terminate and ShutdownWorker are now about 1 ms apart, where they were 1.00 s apart before the fix (fix-live-focused.log). The live subtests also got faster: Complete went from 2.5 s to 0.6 s.

**3. Redaction.**
- The Runner decodes a refused call's request by its method, redacts it, and re-encodes it deterministically; it removes the bytes when it cannot decode or re-encode them.
- `Run.world` and the Reset request are copies of the caller's world, and the world is redacted explicitly.
- Tests:
  - `TestRedactionReachesRefusedCallsAndTheWorld` (runner): the caller's world is untouched.
  - `TestAHostRefusedRunRedactsItsEvidenceAndIsTerminatedByClose` (act) runs the real guest with a refused Answer. The task token and the World's identity are absent from the whole Run.

**4. Timeout and cancellation at each wait.** `TestEveryWaitEndsAtTheReadTimeoutAndAtClose` covers three waits, each at its read timeout and at Close:
- an admitted poll that never returns;
- a taken task with no callback;
- a callback decision that never comes.

Further tests:
- `TestAnEmptyPollResumesReads`
- `TestCloseTerminatesWhatTheDriverStarted`: answered, answered too late (terminated by ID), NotFound and FailedPrecondition tolerated, other errors reported
- `TestAnOutOfRangeStatusIsMalformed`

### Minors and nits

- **Driver-invented responses.** A refused, unclaimed or abandoned task is held until Close. Close terminates first and releases the task afterwards. A response the SDK still completes after the release is a teardown fault, recorded by an interceptor (`TestALateSDKResponseIsATeardownFault`). The live HostRefusal now asserts TERMINATED.
- **Live assertions.**
  - The refused Run's replay must print the same `refused at event N: model refused METHOD_ANSWER: REFUSAL_CODE_UNSUPPORTED` line as explore.
  - Cancellation asserts both specifics: a Run whose request 2 is Cancel, Accepted; and a `Client::Cancel Rejected(NotFound)` after `Worker::RespondCompleted`.
- **Boundary.** `act` may import exactly `model/umpire/{runner,run/v1,model/v1,realization/v1}` and `temporal/umpire/temporal/v1`. `realization/v1` is the Observation, Callback and Reply envelopes the Driver builds and unpacks. The guest, `umpire.local/temporal`, is allowed in `_test.go` only. New negative cases cover `act` importing `umpire.local/model` and `umpire.local/temporal`.
- **Temporal prose.** `model/umpire/hosts/go/README.md` now reads "registers the Drivers its build links"; the Temporal specifics stay in `tools/umpire/README.md`.
- **Configuration.** Redaction paths are resolved through `protoregistry` and the field descriptors; an unknown message, an unknown field or a non-message intermediate is refused. A worker poll timeout under 2 s is refused. Either way the command exits 3.
- **Nits.**
  - A Start that was sent but never answered, or answered unreadably, is terminated by activity ID. "Sent" is decided by the stats `OutHeader`, so an unreachable endpoint triggers no termination.
  - A zero poll interval, backoff or read timeout is a Start error.
  - A reply that sets a field the SDK cannot send (anything besides result or failure, and the SDK-owned task_token, namespace and identity) makes Respond return an error, which the Runner records as a fault.

### Gates

Logs are in `.flow/tmp/fn-228.4/`, each run alone.

| Gate | Result |
| --- | --- |
| act and cmd/umpire `-count=5` | rc 0 |
| `-race -count=5`, CGO_ENABLED=1 (fix-race.log) | rc 0 |
| Runner tests | rc 0 |
| Boundary tests | rc 0 |
| golangci-lint on act and cmd/umpire | 0 issues |
| golangci-lint on the integration test file | no findings |
| `go vet -tags test_dep,integration ./tests/` | rc 0 |
| `make umpire-check-model-slice` (fix-slice-gate.log) | rc 0, 23 s, 26 passes |
| New live test `-count=3` (fix-live-count3.log) | rc 0, 19 s |
| `make model-check` (fix-default-gate.log) | rc 0, 713 s |
| `UMPIRE_BACKENDS=require make model-check` (fix-required-gate.log) | rc 0, 1172 s; Apalache is out of this gate since 12f5645a68 |
| Gaps gate | not rerun: guest bytes unchanged |

## Re-review fixes (review-2.md, SHIP with two minors)

Commit: aa7c23e6f1 fix(testpilot): only a response to a task the Model left unanswered is a teardown fault

**N1. Flaky `TestCloseRefusesHeldPollsAndCancelsTheAdmittedOne`.**
- Root cause: the test assumed the first worker window admits the intended poll. A window closes after the poll interval if no poller has reached the gate yet; the next window then opens only after another continued read, which the test never answered.
- Fix: a test helper `continueUntil` answers Observations with continue until the gate holds the poll. The hanging poll is admitted first; only then does the second poll start, so it can only be held.
- Every gate test with the same assumption now uses the helper.
- Read-timeout variants now use 2 s, which leaves room under `-race`.
- Result: 0 of 300 runs failed under `-race` (n1-race-300.log).

**N2. A late Model-answered response counted as a teardown fault.**
- `hold` now records the task tokens of the tasks it held because the Model did not answer them.
- `observeResponses` faults only a successful RespondActivityTask* whose task token is among those, whenever it completes.
- New test `TestOnlyAResponseToAnUnansweredTaskIsATeardownFault`:
  - a task answered by the Model, whose response completes after the release, is no fault;
  - a held unanswered task's response is exactly one fault.

**Unattributed Runner failure.**
- Reproduced: `go test -race -count=200 ./umpire/runner/` failed 12 of 200 runs, all `TestModelRefusalsRefuseTheRun/pre-admission_callback`: "the Driver did not receive the refusal: []" (runner-race-200.log).
- Cause: in the fake exchange (fake_test.go, from fn-228.3), Respond buffers an accepted decision. The Runner then closes the exchange at once, because the refusal ended the Run. The fake's wait chose between the buffered decision and the closed context at random, so sometimes it never recorded the decision.
- Fix: on close, the fake still records the decisions Respond already accepted.
- The test passes 300 of 300 under `-race` (runner-refusals-race-300.log).
- The production Runner is unchanged.

**Gates**, each run alone; logs in `.flow/tmp/fn-228.4/`:

| Gate | Result |
| --- | --- |
| The two changed tests, `-race -count=300` | rc 0 |
| `act`, `-race -count=50` (fix2-act-race-50.log) | rc 0 |
| `act` and `cmd/umpire`, `-race -count=10` | rc 0 |
| Runner, `-race -count=3` after the fix (fix3-runner-race.log) | rc 0 |
| golangci-lint on `act`, `cmd/umpire` and the runner | 0 issues |
| `make umpire-check-model-slice` (fix3-slice-gate.log) | rc 0, 138 s, 26 passes |
| `make model-check` (fix3-default-gate.log) | rc 0, 1465 s, at load around 30 |

model-time was left to the coordinator, as asked.
