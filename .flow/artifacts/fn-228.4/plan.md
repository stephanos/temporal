# fn-228.4 plan: the Temporal Driver acts and observes; the live activity slice runs through `umpire explore`

Phase 1 (read and plan) only. Nothing was built, run or edited. Base: `umpire` at 47e2d38822. Other
sessions' uncommitted files (MILESTONES.md, model/design/native-symbolic-backend.md, fn-240 specs)
stay untouched.

## What I read, and the facts the plan depends on

- Runner contract (`model/umpire/hosts/go/umpire/runner`):
  - `Driver.Start(ctx, request, payload) (Exchange, error)` returns once the request is sent.
  - `Exchange` has `Events()`, `Respond(ctx, id, Decision)` and `Close()`.
  - Event kinds are Observation, Callback, Annotate, Done and Fault. Event IDs increase strictly,
    and every event carries the request number Start received.
  - A Decision carries a Judgment (which may set `continue_polling`), a Reply, an Annotation or a
    Refusal.
  - The Runner faults on Done before a final Explained observation, on a closed stream, on crossed
    or duplicate IDs, and when no event arrives within the read timeout.
  - The Runner closes a Driver through `io.Closer`, without a timeout. Fresh Host and Driver per Run.
  - `tools/umpire/cmd/umpire/main.go` registers `drivers = map[string]runner.NewDriver{}`, so
    explore, reduce and repeat exit 3.
- Temporal Realization (`model/temporal/feature/activity/temporal.rs`, `realization/src/live.rs`):
  - Reset takes `configuration = Any(StartActivityExecutionRequest)`, the World.
  - Propose #1 is BOOTSTRAP: the World's Start request, with `request_id` filled from the seed.
    Admission compares the observed `request` Any with the pending bootstrap **byte for byte**.
  - Action proposals:
    - `Client::Cancel` is a plain `RequestCancelActivityExecutionRequest`.
    - `Worker::Poll(n)` and the worker responses (`RespondCompleted`, `RespondFailed{..}`) are all
      `umpire.temporal.v1.ObserveActivity{request: Describe, poll_interval_ms 10, read_timeout_ms
      30000, backoff_ms 10}`.
  - At most two actions are eligible: Cancel at index 0, plus Poll or a response. The choice is
    `seed % 2`, so whether request r is Cancel depends only on `ProposalSeed(seed, r) % 2`.
  - Evidence is `realization.v1.Observation{method, request, response|error}` and
    `Callback{task: PollActivityTaskQueueResponse}`. A Reply is
    `Reply{request: RespondActivityTaskCompleted|FailedRequest}`, and the worker sends it
    immediately; it is never resent.
  - **A Describe that shows Started before the callback is Answered is Unexplained**:
    `project_phase` has no path from Scheduled to Started except through Answer's delivery.
  - An attempt-2 callback while the Model is at Started(1) is refused with STATE.
- Existing live slice: `tests/testpilot_model_slice_test.go`.
  - It drives the Host directly, choosing the actions in Go.
  - Its SDK worker body reconstructs the task from `activity.GetInfo`.
  - It gates `PollActivityTaskQueue` with an interceptor permit, which is why the scripted test is
    deterministic.
  - `make umpire-check-model-slice` requires the subtests Complete, Fail, HostRefusal and
    Cancellation, with a 2-minute timeout. The cluster comes from `activityEnvironment(t)`
    (testcore, CHASM plus standalone activities and operator commands).
- Testpilot's `temporal/*` Driver is Case- and Program-bound (server Session, worker reservations,
  carriers). Only `temporal/provision` (namespace create and delete over the public services) is
  reusable as is. Deleting the rest is .6's job.
- Boundaries:
  - The Testpilot boundary tests forbid `go.temporal.io/server/{tests,tools/umpire,tools/canary,model,api/umpire}`.
  - The ownership test lets `cmd` import Testpilot packages.
  - `model-check` runs Go tests only for the two `model/.../hosts/go` modules, so server-module
    packages are tested by focused `go test` runs and the live targets.

## 1. Where the Driver lives

**`common/testing/testpilot/temporal/act`, package `act`.** It is the R4 "retained temporal/*
driver reduced to act-and-observe". The spec and the module map put the Temporal Driver under
`common/testing/testpilot/temporal`, and .6 counts it as a Runner adapter. It is outside `model/umpire`.

Imports:
- stdlib;
- `google.golang.org/grpc` (with `credentials/insecure`, `encoding`, `stats` and `status`) and
  `google.golang.org/protobuf/{proto,reflect/protoreflect,reflect/protoregistry,types/known/anypb}`;
- `go.temporal.io/api/workflowservice/v1`, plus `operatorservice/v1` through provision;
- `go.temporal.io/sdk/{client,worker,activity,converter,temporal,log}`;
- `umpire.local/model/umpire/runner`, `umpire.local/model/umpire/realization/v1` and
  `umpire.local/temporal/umpire/temporal/v1` (ObserveActivity);
- `go.temporal.io/server/common/testing/testpilot/temporal/provision`.

There are no new third-party modules: every one of these is already in the server `go.mod`.
`CGO_ENABLED=0`.

Files:
- `act.go`: Options, Factory and Driver.
- `exchange.go`: the unary and read exchanges.
- `rpc.go`: method resolution, the pass-through codec, the completion check and evidence checks.
- `worker.go`: the SDK worker, the poll gate, task capture and reply delivery.
- `redact.go`
- Tests alongside.

Boundary updates, as explicit edges:
- `dependency_boundary_test.go` and `ownership_test.go` permit `umpire.local/model/umpire/{runner,
  run/v1,model/v1,realization/v1}` and `umpire.local/temporal/umpire/temporal/v1` for `temporal/act`
  only.
- A new negative case asserts that every other Testpilot package still imports no `umpire.local/*`.
- The module map (`.plans/UMPIRE_MODULES.md`) gains an `act` row: its job, public interface and
  permitted imports.

## 2. Start, Exchange, Events, Respond and Close over the generated API

`Driver.Start(ctx, request, payload)` launches the exchange goroutine and returns at once. The
type URL selects the exchange kind, which is a mechanic, not a meaning.

- `umpire.temporal.v1.ObserveActivity` starts a **read exchange**. The instruction's inner
  `request` is the read.
- Any other message starts a **unary exchange**.
- The method is resolved from the generated `WorkflowService` descriptor
  (`workflowservice.File_temporal_api_workflowservice_v1_service_proto`). It must be the single
  method whose input type is the payload's message. Otherwise Start returns an error, which the
  Runner records as a DriverFault.

The pieces the exchange kinds share:
- **Pass-through codec.** The request bytes are sent exactly as proposed (`payload.Value`). The
  response's exact wire bytes are captured with a bytes `encoding.Codec` forced per call.
  - Bootstrap admission compares Any bytes, so re-marshalling in Go could fail admission.
  - Observation `request` is the proposal's Any itself. `response` is
    `Any{type_url: output type, value: received bytes}`.
- **RPC completion check.**
  - A `grpc/stats` handler marks a call answered when an `InHeader` or `InTrailer` arrived.
  - A call the server answered becomes an Observation: its response, or `Error{code, message}`.
    A server status is evidence for the Model, whatever its code.
  - A call the server never answered is a transport failure: unreachable, a local deadline,
    cancellation or a codec failure.
- **Evidence well-formedness, checked before any event reaches Judge.** Each failure emits
  `EventFault "malformed evidence: …"`, so the Runner records a DriverFault, the verdict is
  Refused, and no Judgment is recorded.
  - The response bytes must decode strictly (`proto.Unmarshal`, UTF-8 checked) as the method's
    output type.
  - An error code must be non-OK and inside the gRPC range.
  - A callback must have a captured task whose token equals the activity's.
- **Unary exchange.**
  - One RPC, bounded by `Options.RPCTimeout`, then the Observation.
  - The Driver waits for the decision. A final Explained decision emits Done.
  - A Judgment with `continue_polling` emits a Fault, because a unary call cannot be re-read.
  - A Refusal decision ends the exchange.
  - A transport failure emits a Fault at once; Start and Cancel calls are not retried.
- **Read exchange.** It uses the instruction's fields and never reads the response. The fields
  bound the reads; only the Model's decision ends the loop.
  - The loop is: read, emit the Observation, wait for the decision.
  - On final Explained it emits Done.
  - On `continue_polling` it waits `poll_interval_ms`, then reads again. If the instruction admits
    a worker task, the worker step in §3 comes first.
  - A read the server did not answer is retried after `backoff_ms`.
  - If no final acceptance arrives within `read_timeout_ms` of the exchange start, it emits
    `Fault "read timeout without Model acceptance"`. `Options.MaxReadTimeout` caps the timeout so
    that the tests stay short.
- **Events and Respond.**
  - One goroutine per exchange owns the event IDs (1, 2, …), sends on an unbuffered channel under
    the exchange context, and ends with Done or Fault before closing the channel.
  - `Respond` hands the decision to that event's waiter (a buffered channel of size 1) and never
    blocks.
  - Respond returns an error for an unknown or already decided ID, a decision kind that does not
    match the event, a response after Close, and a Reply the SDK cannot deliver.
- **Close** cancels the exchange context, releases the poll gate and any waiting worker body, and
  joins every goroutine.

## 3. Callbacks from the in-process Go SDK worker

One SDK client and one worker per Driver, so per Run, on `Options.TaskQueue`:
- the client is `client.NewLazyClient`, with a discard logger so that no payload reaches stderr;
- the worker has `MaxConcurrentActivityTaskPollers: 1` and one dynamic activity
  (`RegisterDynamicActivity`), so no type name needs configuring;
- the worker starts lazily, at the first read exchange that admits a worker task, so Cancel-only
  Runs and the unreachable control never start it.

**Poll gate.** A unary client interceptor on the SDK connection holds every
`PollActivityTaskQueue` until a read exchange opens the gate.
- The gate opens only after a read is decided `continue_polling` and only while that exchange
  still has worker tasks left (see decision D2).
- It admits one poll, with a deadline of `Options.WorkerPollTimeout` (default 5 s; the frontend's
  minimum long poll is 2 s, and matching returns an empty response one second before the
  deadline).
- **Reads pause while an admitted poll is in flight.** Otherwise a Describe could see Started
  before Answer, which is Unexplained.
- An empty poll lets the read loop go on.
- A poll that returns a task (non-empty `task_token`) keeps reads paused until the callback is
  decided.

**Capture.** The interceptor keeps a clone of the actual `PollActivityTaskQueueResponse`, keyed by
task token. The worker never rebuilds the task from `activity.GetInfo`.

**Delivery.**
1. The activity body looks up its captured task by `GetInfo(ctx).TaskToken`.
2. It hands the task to the exchange that admitted the poll. That exchange emits
   `EventCallback{Payload: Any(Callback{task}), Call: n}`, where n is the Driver's worker-call
   counter.
3. The body waits for the decision, the exchange's close or its own context.
4. A Reply unpacks to `realization.v1.Reply.request`, which the body delivers through the SDK:
   - `RespondActivityTaskCompletedRequest` with one payload: return `converter.NewRawValue(payload)`.
   - `RespondActivityTaskFailedRequest`: return `temporal.GetDefaultFailureConverter().FailureToError(failure)`.
   - Anything else: Respond errors, the Runner records a fault, and the body returns a
     non-retryable error.
5. A Refusal makes the body return a non-retryable "the Model refused the callback". The Run is
   already Refused.

The SDK then sends its response at once, as fn-228.2 requires (no duplicate response RPCs). Answer
reaches the Host only through the Runner's serialized session, in receive order. Shim callbacks in
.5 can enter the same exchange entry point.

## 4. Registration with the `umpire` command

`tools/umpire/cmd/umpire/main.go` changes as follows.
1. Configuration comes from the environment:
   - `UMPIRE_TEMPORAL_ADDRESS` (required)
   - `UMPIRE_TEMPORAL_NAMESPACE` and `UMPIRE_TEMPORAL_TASK_QUEUE` (required for worker tasks). They
     must match the World; the Driver never reads the World.
   - `UMPIRE_TEMPORAL_CREATE=1`: register the namespace before the first Run and delete it after
     the last, through `provision`.
   - `UMPIRE_TEMPORAL_RPC_TIMEOUT` and `UMPIRE_TEMPORAL_WORKER_POLL_TIMEOUT`
   - `UMPIRE_TEMPORAL_REDACT`: comma-separated field paths.
2. `act.Configure(os.Getenv)` builds `act.Factory`.
3. main registers `drivers["temporal"] = factory.New`.
4. If explore, reduce or repeat is requested with an invalid configuration, main prints
   `umpire: temporal driver: …` and exits 3, before `runner.Main`.
5. After `runner.Main` returns, main calls `factory.Close(ctx)` (provisioning cleanup) before
   `os.Exit`. A cleanup failure is printed and turns exit 0 into 3.

`runner.Main`'s signature is unchanged. Provisioning is lazy, so question and replay never touch
the network.

`Driver.Close`, bounded by `Options.CloseTimeout`:
1. Stop the worker.
2. **Terminate any activity this Driver started that is still open.** The Driver takes the
   identity from the Start invocation it made: the request's `namespace` and `activity_id` and the
   response's `run_id`. NotFound and FailedPrecondition are ignored.
3. Close the connections.

Termination is needed because a later seed's Start reuses the World's `activity_id`. Any other
cleanup failure is returned, and the Runner records it as a fault.

The docs to update: `tools/umpire/README.md` (the Driver's variables), the `umpire` paragraph of
`model/umpire/hosts/go/README.md`, and a short `act` section in
`common/testing/testpilot/temporal/README.md`. There is no new README.

## 5. Live tests that replace the four retired ones

New file `tests/testpilot_model_slice_umpire_test.go` (`test_dep && integration`) with
`TestTestpilotModelSliceThroughUmpire`.

Setup:
- `activityEnvironment(t)`.
- Build the command once, with `buildUmpireCommand(t, "umpire")`.
- Pass `--model` the committed `model/temporal/hosts/go/umpire_temporal_guest.wasm`, and assert
  its bytes equal `temporal.Guest()`. That preserves the embedding and freshness path.
- Build each World as a `StartActivityExecutionRequest` written as a binary Any. It sets the
  namespace, a unique activity ID and task queue, the `ModelSlice` type, the input, a
  start-to-close timeout, a retry policy and `ALLOW_DUPLICATE` reuse.
- The test chooses `--seed n` with `runner.ProposalSeed`, because the choice at request r depends
  only on `ProposalSeed(s, r) % 2` (Cancel is even). That makes the coverage deterministic.

Every invocation:
- runs as a separate process, in an empty working directory, with
  `explore --budget 6 --event-timeout 60s --store <dir>`;
- afterwards the working directory is still empty, and the store holds exactly n files
  `<hash>/<seed>/<sha256>.binpb`, matching the `published` lines. That is "the Run is the only
  artifact".

Subtests:

| Subtest | Replaces | Proves |
| --- | --- | --- |
| `Complete` (with `UMPIRE_TEMPORAL_CREATE=1` and a fresh namespace) | `UmpireRunRunsACheckedInCaseAgainstAnyEndpoint` (separate process, exit 0, provisioning and cleanup) | Exit 0, `explained: proposal_budget`. Every Run is Explained. Some Run shows, within one request: Bootstrap, admission, then `Judgment(continue_polling)` → `Answer` → final Judgment explaining `Worker::RespondCompleted` (in the read or the next proposal). After exit the namespace is gone (DescribeNamespace returns NotFound or DELETED). |
| `Cancellation` | (new R4 item) | Over the Complete store: a Run whose request 2 is Cancel, with a `Client::Cancel` Accepted judgment, and a later rejected Cancel after completion. |
| `RetryThenComplete` | (new R4 item) | `maximum_attempts 2` and a 1 s retry interval. A seed whose requests 2 to 4 are non-Cancel. Two Answers (attempts 1 and 2), a judgment explaining `Worker::RespondFailed { retryable: true }`, then completion. Exit 0. |
| `Fail` | (keeps the slice's Fail) | A non-retryable failure is explained (`RespondFailed { retryable: false }`). Exit 0. |
| `HostRefusal` | (keeps the slice's HostRefusal) | Exit 2. The deciding Run has `Refusal{METHOD_ANSWER, UNSUPPORTED}`. After exit no activity of that World is RUNNING (Close's termination). |
| `UnreachableEndpoint` | `UmpireRunRejectsAnUnreachableEndpoint` | `UMPIRE_TEMPORAL_ADDRESS` at a closed local port. Exit 2. The Run is Reset, Proposal(bootstrap), DriverFault. The `refused at event 2: driver fault` line. |
| `ReplayWithoutServer` | `NexusControlReplaysThroughTheCommand` (the replay half) | `umpire replay --run <the callback-bearing Complete Run>` with every `UMPIRE_TEMPORAL_*` unset exits 0 with identical judgments. The HostRefusal Run replays to exit 2. |

`AssessRecordedRuns` maps to `ReplayWithoutServer` plus the runner's publication tests; profiles
and receipts are gone by spec.

`TestTestpilotModelSlice`, the scripted host-level test, stays unchanged: its frozen, mismatch and
isolated-host controls have no Runner path. .6 maps or retires it (decision D8).

Makefile `umpire-check-model-slice`:
- run `-run '^TestTestpilotModelSlice(ThroughUmpire)?$'` with `-timeout 10m`;
- add `TestTestpilotModelSliceThroughUmpire` and its seven subtests to the awk required list.

## 6. Error cases

- **Read timeout without Model acceptance.**
  - The Driver emits Fault when `read_timeout_ms` (capped by `MaxReadTimeout`) elapses without a
    final Explained read. The Runner records a DriverFault, the verdict is Refused, and the
    command exits 2.
  - The Runner's own wait timeout covers a Driver that goes silent.
- **Malformed evidence is refused before Judge.** These are the §2 checks; a Fault is emitted
  instead of the Observation or Callback.

Controls in `act` (unit tests, no live server). They use an in-process fake `WorkflowService` gRPC
server on 127.0.0.1:0 (an `UnknownServiceHandler` with a bytes codec) and a fake worker seam.

- **Unary exchange.**
  - The exact proposal bytes reach the server, and the exact response bytes are recorded.
  - A server status becomes `Observation.error`.
  - An unreachable endpoint is a Fault.
  - Undecodable response bytes are a Fault with no Observation.
  - `continue_polling` on a unary call is a Fault.
- **Read exchange.**
  - Continue, continue, final gives three reads spaced at least the poll interval apart, then Done.
  - An incomplete read is retried after the backoff.
  - The read timeout produces a Fault.
  - An instruction with `worker_tasks=0` never opens the gate.
  - With `worker_tasks=1`, one poll is admitted. Reads pause while it is in flight. The Callback
    carries the captured task and a call ID. The Reply reaches the body.
  - An empty poll resumes reads.
- **Respond:** unknown or duplicate IDs, a mismatched kind and a response after Close are errors.
  Close joins its goroutines and cancels in-flight work.
- **Reply conversion:** Completed gives a RawValue, Failed gives an ApplicationError, anything
  else is an error.
- **Runner level:** `runner.New(temporal.Guest(), fake factory)` where the fake Start response is
  undecodable. The Run is `[Reset, Proposal(BOOTSTRAP), DriverFault]` with no Judgment, and
  Refused. The same through `runner.Main` exits 2.
- **Race run:** `-race -count=3` with CGO_ENABLED=1, only for this check; the gates use
  CGO_ENABLED=0.

## 7. Gates

Focused, during development:
- `CGO_ENABLED=0 go test -tags test_dep -count=1 ./common/testing/testpilot/temporal/act/... ./tools/umpire/cmd/umpire/...`
- the boundary and ownership tests: `./common/testing/testpilot -run 'Boundary|ModelTooling'` and
  `./tools/umpire/ir -run 'Ownership|LiveCaller|DependencyGraph'`
- `-race`
- golangci-lint v2.13.1 with `.github/.golangci.yml` on the changed packages
- `go vet -tags test_dep,integration ./tests/`

If D2 is accepted:
- targeted `cargo test -p umpire-temporal-realization` (proposal tests) and the macro snapshot test.
  The single changed `activity-realization-authored` snapshot is reviewed and accepted alone,
  never with a global `model-update`.
- `proto-temporal-go.sh` regeneration and `--check`
- `make umpire-guest`, with the new guest committed
- `go test` in `model/temporal/hosts/go` (proposal parity)

Live, each run alone, with background logs under `.flow/tmp/fn-228.4/`:
- `make umpire-check-model-slice`
- `make umpire-check-model-gaps` (guest bytes change under D2)

Quick gates before handoff, in the background with logs: `make model-check` and
`UMPIRE_BACKENDS=require make model-check`. The Full receipt (`UMPIRE_BACKENDS=require make
model-check` plus `make model-time`) waits for the conductor after review, per the brief.

Commits:
1. `feat(temporal): a read instruction says how many worker tasks it admits`: proto, realization,
   tests, snapshot, guest. Only if D2 is accepted.
2. `feat(testpilot): the Temporal Driver acts and observes for the Runner`: `act`, unit tests,
   boundary edges, module map, README.
3. `feat(umpire): register the Temporal Driver with the umpire command`
4. `test(umpire): the live activity slice runs through umpire explore`: the test and the Makefile.

## Open decisions (my recommendation first)

- **D1. Package `common/testing/testpilot/temporal/act`.** Alternatives: `tools/umpire/temporal`
  (not IR tooling) or `model/temporal/hosts/go` (it would pull the SDK into a model module). .6 may
  fold `act` into the parent `temporal` package once the Case Driver is deleted.
- **D2. Add `uint32 worker_tasks = 5` to `umpire.temporal.v1.ObserveActivity`.** The Realization
  sets 1 for `Worker::Poll(n)` and 0 for worker-response reads. Recommend yes.
  - Without it, the Driver cannot tell a Poll read from a response read without the action name,
    which Start does not receive.
  - An always-polling worker makes Runs fail at random. A task taken during bootstrap is refused
    before admission. A poll that is open during the `RespondFailed{retryable}` read takes the
    attempt-2 task while the Model is at Started(1), and Answer refuses it with STATE.
  - The field is a transport mechanic, carried like the poll, timeout and backoff fields. The
    Model and its goldens are unchanged.
  - Costs: rebuilt guest bytes, one macro snapshot, the gaps gate, and fn-241 rebasing onto it.
  - Fallback if the owner refuses: the gate opens on every continued read, and the retry and
    pre-admission races remain as known flakes, which I would not ship.
- **D3. Driver configuration from `UMPIRE_TEMPORAL_*` environment variables.** This leaves
  `runner.Main` untouched; main exits 3 on an invalid configuration. The alternative is a generic
  driver-option flag in `runner.Main`.
- **D4. Close terminates the activities this Driver started that are still open.** It reads the
  Start identity as a transport mechanic, which is needed for multi-seed explore over one World.
  A cleanup failure is a recorded teardown fault.
- **D5. Redaction as field paths, small.**
  - `UMPIRE_TEMPORAL_REDACT` lists the paths. The Driver implements an optional generic
    `runner.Redactor` interface (`Redact(*runv1.Event) bool`), which the session uses when
    `Options.Redact` is nil.
  - The paths are cleared from the decoded evidence after judgment, and the event is marked
    redacted; replay already refuses such a Run.
  - Capture stays whole-message and exact. Capturing a field subset before Judge would forge
    evidence.
  - Alternative: defer redaction to .6. That leaves the R4 wording "capture and redaction as field
    paths" partly unmet.
- **D6. Every status the server answered is evidence.** Only calls the server never answered
  become faults; a `grpc/stats` header or trailer check decides. A server-sent Unavailable is
  therefore judged by the Model, not swallowed.
- **D7. `backoff_ms` is the retry delay after an unanswered read.** `read_timeout_ms` bounds the
  whole read exchange until acceptance. `poll_interval_ms` spaces continued reads. Unary Start and
  Cancel are never retried.
- **D8. Keep `TestTestpilotModelSlice` beside the new test.** The slice target requires both; .6
  maps or retires the scripted test.
- **D9. Exact bytes with a pass-through codec.** It is required for bootstrap admission and makes
  `actual request == proposal` hold by construction.

Risks:
- A live Run may be Unexplained on a real interleaving (for example Cancel during Started, or a
  retry Describe before the retry timer). If so, I stop and report it with the Run file. I will
  not fit the Model or the Realization to it.
- Guest compilation per Run under load may stretch the slice test; the 10-minute timeout covers it.
