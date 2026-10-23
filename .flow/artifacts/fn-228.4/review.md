# fn-228.4 review: the Temporal Driver and the live slice through `umpire explore`

Range `39d79ffa54..d7a144ea4f`: 9ddd74f15b, 780d249590, 9da5c6f44c, ae2ae4a39a, d7a144ea4f. I did not review 12f5645a68 or 8df7e6f142.

## Verdict: NEEDS_WORK

The overall shape is right, and most of the handoff holds up:
- The Driver never inspects a Temporal phase.
- Only `continue_polling` decides whether a read ends.
- Request and response bytes are exact.
- Event IDs and correlations are sound.
- The Realization sets `worker_tasks`. The Driver only counts it against tasks taken, so it is not a Go chooser.

Four major problems block shipping. Three are in the worker gate and in redaction, and two of those I reproduced. The fourth is the binding "timeout/cancellation at each wait" tests, which are missing.

## What I ran

| Check | Result |
| --- | --- |
| `go test -tags test_dep -count=2` on `temporal/act` and `cmd/umpire` | ok |
| Testpilot boundary tests | ok |
| `model/temporal/hosts/go` tests (proposal parity) | ok |
| Runner `Redact` tests | ok |
| `gofmt -l` and `go vet` on the changed packages | clean |
| `make umpire-check-model-slice` | rc 0, 58 s wall; all 7 new subtests and the scripted ones pass |
| `tools/umpire/ir` ownership tests | Only `TestEveryToolingPackageHasALiveCaller` (umpire-gen-cases) fails. It reads the Makefile; this range touches only the slice target, consistent with the handoff calling it pre-existing. |

Probes went in through `go test -overlay`; nothing was written to the repo. The probe sources are in the session scratchpad (`probe/probe_test.go`, `probe/redact_probe_test.go`).

## Findings

### 1. major: the poll gate misattributes stale poll outcomes, so a read exchange can hang or read while a poll is outstanding

**Where:**
- `common/testing/testpilot/temporal/act/worker.go:207-238` (`work`, timer case at 225-229)
- `worker.go:153-158` (`notify` drops when the buffer is full)
- `exchange.go:53` (`polls` buffer of 2)

**The race.**
1. A poll is admitted late in the gate window and ends at once, empty or with an error.
2. `finish` clears `owner` before `notify(pollEmpty)` runs.
3. If `work`'s `select` then picks `timer.C` before it reads the buffered `pollAdmitted`, `closeIdle` sees `owner != x` and returns `(false, true)`.
4. `pollAdmitted` and `pollEmpty` are left queued in `x.polls`.
5. The next `work` call consumes those stale outcomes. It returns while the fresh poll it just admitted is still in flight, so the read loop reads during an outstanding poll; this is the interleaving the gate exists to prevent.
6. Because the buffer is full, the fresh poll's own outcomes are dropped. If that poll took a task, `pollTook` is lost: no callback is emitted, the task stays owned, and later windows can never admit.

**Reproduced** with an overlay probe that drives `gate.intercept` with instant empty polls and jittered poller arrival:
- In 3 of 3 runs, the read loop read with stale outcomes queued (71, 86 and 76 reads).
- Each run then hung with no event for 15 s, twice after the task was taken.

Through the Runner, this becomes a spurious "read timeout without Model acceptance" DriverFault: a Refused Run. If a Describe lands between the task being taken and its Answer, it can become a spurious Unexplained instead, which is a false verdict about the system.

**Production trigger:** any poll that returns immediately. Examples are a poll error (ResourceExhausted, namespace not ready, a canceled SDK context) or `UMPIRE_TEMPORAL_WORKER_POLL_TIMEOUT` below 2 s (see finding 9). With the default 5 s, the trigger is rare.

**Fix:** tag each gate window or admission with a generation. Then:
- `notify` carries the generation, and `work` ignores outcomes from other windows.
- On the timer, `work` closes the window only if no poll was admitted in *this* window, checked under the gate lock, not `owner != x`.
- `notify` must never drop the outcome of a current-window poll.

Add a unit test with instant empty polls and a late-arriving poller, like the probe.

### 2. major: Close never releases polls held in the gate, so every Run's worker Stop burns its full timeout and leaks the poller

**Where:**
- `act/act.go:277-303` (`close`)
- `worker.go:82-99` (`admit` waits only on the gate or the SDK context)

**Mechanism.** The server's default `frontend.enableCancelWorkerPollsOnShutdown=true` makes SDK v1.48 use drain mode. In that mode `doPoll` does not cancel in-flight polls on Stop, and relies on the server completing them after ShutdownWorker. The poll held in `admit` never reached the server, so nothing completes it:
- `worker.Stop()` waits its full `WorkerStopTimeout` (1 s).
- The poller goroutine stays blocked until the SDK's 70 s poll deadline.

**Seen live** in this review's slice log. Every umpire Run with a worker shows ShutdownWorker followed by Terminate exactly 1.00x s later, with `cancelled-count: 0`:

| ShutdownWorker | Terminate |
| --- | --- |
| 30.998 | 32.004 |
| 32.916 | 33.920 |
| 37.700 | 38.707 |
| 40.907 | 41.917 |

**Cost.** This breaks the binding "Close cancels and joins owned work". `explore --seed 1000` pays about 1000 s of Close time and keeps up to about 70 s of leaked pollers alive at any moment.

`Driver.Close` also returns on `CloseTimeout` without joining `d.close` (`act.go:264-275`).

**Fix:**
- Add `gate.stop()`, which sets a closed flag and broadcasts, so `admit` returns a canceled status at once. Call it in `close` before `worker.Stop()`.
- Test that a poll held in the gate returns promptly on `Driver.Close`, and that Close completes well under `WorkerStopTimeout`.

### 3. major: redaction leaves the configured fields in every refused call

**Where:**
- `act/redact.go:13-56`
- `model/umpire/hosts/go/umpire/runner/session.go:56-62`
- `model/umpire/proto/umpire/run/v1/run.proto:73-78`

`runv1.Refusal.request` holds the exact request as `bytes`. `redact` walks only message fields and packed Anys, so it never looks inside these bytes.

**Reproduced.** An overlay probe built a refused `METHOD_ANSWER` event and redacted it with the path `temporal.api.workflowservice.v1.PollActivityTaskQueueResponse.task_token`. The result was `removed=false`, and the token was still present in the bytes.

**Impact.** The HostRefusal Run is exactly this shape: its deciding event carries the whole task (token, input, header). Refused Runs are the ones most likely to be shared for triage. Judge refusals likewise keep the whole Observation.

**A fragile part that works today.** `Run.world` is redacted only because it is the same `*anypb.Any` pointer as the Reset request's configuration (`session.go:38` and `:244`), and `redact` rewrites `Value` in place. Any copy of that Any reopens the leak.

**Fix:**
- In the Runner's `finish`, decode `Refusal.request` by `Method` into its typed request and apply `Redact` to it. Re-encode it deterministically and mark the event redacted; or refuse to publish when decoding fails.
- Redact `Run.world` explicitly rather than through aliasing.
- Add a test with a refused Answer and the World.

### 4. major: the binding "timeout/cancellation at each wait" tests are missing

**Where:** `act/act_test.go`.

The spec's Read completion contract requires testing "plus timeout/cancellation at each wait". Existing tests cover the read-loop timeout, the backoff retry and the cap. Untested:
- **Waits with no test:**
  - an admitted poll that never returns (`work`'s `ctx.Done` path);
  - a worker that took a task and never calls back (`callback`, "never called back");
  - a timeout while awaiting the Model's decision on a callback;
  - Close or cancellation during each of those waits, including that `deliver` returns.
- **Plan §6 promises with no test:**
  - "an empty poll resumes reads" (`poll(…, "")` is never used);
  - "Close joins its goroutines and cancels in-flight work";
  - Close's termination of started activities. A fake server could assert the TerminateActivityExecution request and its NotFound/FailedPrecondition tolerance.
- **Evidence classification:** the malformed out-of-range status branch (`rpc.go:116-117`) is untested.

Findings 1 and 2 sit exactly in these untested waits.

**Fix:** add one test per wait, timeout and cancellation, using the existing `poll`/`deliver` seams. Add the empty-poll, Close-join and Terminate tests against the fake server.

### 5. minor: the Driver sends worker responses the Model never gave, and the HostRefusal test relies on that, not on Close's termination

**Where:** `act/worker.go:289-313` and `tests/testpilot_model_slice_umpire_test.go:311-321`.

The Driver itself sends three kinds of response:

| Situation | What the SDK sends |
| --- | --- |
| Refused callback | A non-retryable "the Model refused the callback" failure |
| Closed exchange | A `ctx.Err()` failure |
| Unclaimed task | A non-retryable "no read let the worker take this task" failure |

All three happen after the verdict, but they are system state changes the Model neither proposed nor answered.

**Seen live** in HostRefusal: RespondActivityTaskFailed at 40.912, then Terminate at 41.917, which hit an already-failed activity. The subtest's "the Driver leaves no activity running" assertion therefore passes because of the Driver's invented failure. It would still pass if Close's termination were removed. The handoff's claim that this proves Close's cleanup is not established.

**Fix:**
- Terminate the activities this Driver started *before* releasing blocked activity bodies (close the exchanges and stop the worker after Terminate), so a late SDK response hits NotFound.
- Make HostRefusal assert TERMINATED, or check Close's Terminate in a unit test (finding 4).

### 6. minor: two live assertions are weaker than claimed

**Where:** `tests/testpilot_model_slice_umpire_test.go:354-355` and `:258-264`.

- **Refused replay.** `replay --run <refusedRun>` asserts only exit 2. Replay also exits 2 when it refuses a mismatched or tampered Run (`replay.go:128-129`). So the test cannot tell "reproduced the Answer refusal" from "replay rejected the record". Assert the `refused at event 5: model refused METHOD_ANSWER: REFUSAL_CODE_UNSUPPORTED` line, as the output already prints it.
- **Cancellation.** It checks only that some Run has `Client::Cancel Accepted`. The plan promised two specifics, both absent:
  - a Run whose request 2 is Cancel;
  - a later rejected Cancel after completion.

### 7. minor: the boundary allowance admits all of `umpire.local/*` into `act`

**Where:** `common/testing/testpilot/dependency_boundary_test.go:274-277`.

The rule exempts any import under `umpire.local` for files under `temporal/act/`. That includes:
- `umpire.local/model`, the Host with Judge and Answer;
- `umpire.local/temporal`, the guest and the Realization.

With those, the Driver could consult or reinterpret the Model itself. The plan (§1) and the module-map row name explicit edges: `runner`, `run/v1`, `model/v1`, `realization/v1` and `temporal/v1`. Only `runner_test.go` needs `umpire.local/temporal`.

**Fix:** allow exactly those packages for non-test files, and `umpire.local/temporal` for `_test.go` only. Add a negative case for `act` importing `umpire.local/model`.

### 8. minor: Temporal-specific prose under model/umpire

**Where:** `model/umpire/hosts/go/README.md:146-148`.

The new text names "the Temporal Driver, `common/testing/testpilot/temporal/act`" and the `UMPIRE_TEMPORAL_*` environment. The owner's rule is "temporal details cannot be in model/umpire under any circumstances". The boundary gate scans only go, proto, rs and toml, so it does not catch Markdown.

**Fix:** say that the command registers the Drivers its build links, and leave the Temporal specifics in `tools/umpire/README.md`, which already has them.

### 9. minor: configuration accepts values that silently break or no-op

**Where:** `act/act.go:109`, `:112-116` and `:93-97`.

- **Redaction paths.** A path is valid if it contains a dot. A typo such as `…PollActivityTaskQueueResponse.tasktoken` redacts nothing and reports nothing: a silent leak by misconfiguration. Fix: resolve each path at Configure time through `protoregistry.GlobalTypes` and the field descriptors, and reject unknown messages or fields.
- **Worker poll timeout.** `UMPIRE_TEMPORAL_WORKER_POLL_TIMEOUT` below the frontend's 2 s minimum is accepted. The README says "at least the frontend's 2s", but every admitted poll then errors immediately; that is the trigger for finding 1, and the Run read-times out. Fix: reject values below 2 s.

### 10. nit: a Start the server executed but answered late is never cleaned up

**Where:** `act/exchange.go:175-179` and `act.go:245-260`.

If `RPCTimeout` expires after the server has started the activity, the exchange faults, but `started` is never called. Close does not terminate that activity. The next seed's Start of the same World under ALLOW_DUPLICATE then meets a running activity, so later Runs of the same explore are refused.

**Fix:** on an unanswered Start, record the request's namespace and activity ID with no run ID. Close can then terminate by ID.

### 11. nit: `poll_interval_ms = 0` closes the gate window instantly

**Where:** `act/worker.go:210` and `exchange.go:207-208`.

`work` uses the poll interval as the gate window. With 0, the timer fires before any poller can be admitted, so `worker_tasks` is never honoured, and the read loop spins Describe without delay until the read timeout. `backoff_ms = 0` spins the same way.

The Realization uses 10 today. Still, the window being the poll interval is undocumented coupling, and zero-valued instruction fields are not validated.

**Fix:** reject zero interval or backoff in `validStart` as a Start error, or give the gate window its own documented bound.

### 12. nit: `deliverable` silently drops Reply fields the SDK cannot send

**Where:** `act/worker.go:174-203`.

Only `result` (exactly one payload) and `failure` are carried. Any other field the Model sets (identity, worker version, deployment options) is discarded, while the Model believes its reply was sent as given. The current Realization fills only result and failure, so nothing breaks today.

**Fix:** fault in `Respond` when the reply's request has fields other than result or failure set, with SDK-owned fields like `task_token` and `namespace` documented as exempt.

## Handoff claims checked

**Accurate:**
- Exact request bytes through the pass-through codec, and exact response bytes recorded.
- The stats-handler `InTrailer` check decides evidence versus fault.
- Malformed evidence is refused before Judge, at both Driver and Runner level.
- Only the Model's decision ends a read.
- Exactly one snapshot changed. Behavioural goldens are unchanged; `temporal.rs` is the Realization, not the Model.
- The guest is `d80dd567…` and equals `temporal.Guest()`.
- No new modules in `go.mod`.
- The live subtests all pass. Redaction applies only when `Options.Redact` is nil.

**Inaccurate or overstated:**
- "only `worker_tasks` times per read": the Driver counts tasks taken, not polls admitted. Empty polls are unbounded.
- "HostRefusal … No activity is left running" as evidence of Close's termination (see finding 5).
- "Close … releases the poll gate … joins every goroutine" (plan §2): not implemented (finding 2).
- Plan §6's empty-poll and Close-join controls do not exist (finding 4).

## Owner rules

| Rule | Status |
| --- | --- |
| No Temporal under model/umpire | Code is clean: the Runner change is a generic `Redactor`. Prose violates it (finding 8). |
| Go is dumb | No phase checks. Completion comes only from `continue_polling`. Start identity is parsed only for cleanup. |
| Model and goldens unchanged | Held; only the accepted snapshot changed. |
| Never fit the Model to the code | Held. |
| No new third-party deps | Held. |
| Never call a test input a "case" | Held in code. "Case Driver" names the existing Testpilot Case Driver. |
| Comments sparse, why not what | Acceptable. |
