# Temporal SDK worker Driver

This package owns the SDK-worker half of the Temporal Testpilot Driver. A `Driver` keeps compatible
queue registrations alive across Runs, while each `Session` owns its prepared entrypoints,
reservation-delivery ledger, completion bridge, failure state, and bounded diagnostics.

At construction, the Driver freezes the Profile binding snapshot. `Validate` compares the prepared binding
IDs for StartWorkflow, StartActivity, GetHistory, and StartNexus carriers before any registry or target effect;
equal resolved strings reached through different IDs still reject. `Open` derives the physical
namespace, queues, and named Nexus routes from the validated prepared roles. The SDK client and
worker lifecycle configuration remain caller-owned physical inputs.

Task-queue registrations are complete before a worker starts. A registration consists of the
allowlisted workflow types, activity types and Nexus service/operation pairs assigned to that
physical queue. Two Runs share a worker only when all three agree, so a Program that runs an
activity never shares the worker of one that runs none, and a queue that names no activity type
registers no activity implementation.

A Program that declares no `InjectFault` shares compatible registrations across Runs, as above.
Deliberate outages belong to the registry's `outage` module. Definition preparation calls
`PlanOutages` once, and `Validate` and `Open` read the same `OutagePlan`: it resolves each fault's
task-queue role to its queue and refuses a fault on a queue no entrypoint of the Program registers a
worker on. A plan that declares a fault `Requires` a dedicated worker group keyed by its Run instead,
so an outage this Run asks for can never reach another Run's workers. `Session.InjectFault` asks the
Run's `Outage` to `Begin` the transition, which records it under the registry lock and returns the
`Settle` the effect handle's `Wait` runs: `FAULT_KIND_WORKER_STOP` stops that group's SDK worker and
suppresses its fatal-failure callback for the stop window, and `FAULT_KIND_WORKER_RESUME`
re-registers with the same structural signature; both are bounded by the instruction's own timeout.
A second transition in the same direction conflicts, and a fault on a group that is not dedicated is
an unsupported operation. Closing the Session calls `Restore`, which refuses any transition it did
not start, resumes a worker still stopped, and always reaches the registry, so a resume that cannot
finish is reported as a failed cleanup rather than leaving the Run's hold behind. Tasks queued
during the stop window wait in matching and dispatch after the resume. A transition the Driver
refuses outright, or cannot complete, is a failed instruction outcome plus a Driver invariant
diagnostic: the Run records that the fault was requested and not realized, and the Verdict is left
to the Contract.
The typed worker instructions (fn-85 R10) reach the SDK through `typed.go`: a `WorkflowCommand`
carrying `ScheduleNexusOperationCommandAttributes` becomes `workflow.ExecuteNexusOperation` with the
carried payload as an unconverted `converter.RawValue`, the carried timeouts, and the carried Nexus
header merged under the Run's routing header; a `WorkflowCommand` carrying
`ScheduleActivityTaskCommandAttributes` becomes `workflow.ExecuteActivity` on the queue its
task-queue role binds, with the carried payloads unconverted, the carried timeouts and retry policy,
and no eager execution; an `AwaitInstruction` reads either future's payload; a `NexusHandlerReply` becomes the handler's return, a
synchronous payload unconverted, an asynchronous reply through the completion authority the Session
publishes under its own token, a handler error with its type and retry behavior, or a failed start
as an operation error; and a `NexusOperationCompletion` becomes the completion callback's body, a
payload verbatim or a failure converted as the Nexus SDK converts a Temporal failure, canceled when
its failure info says so. A handler error or failed start the entrypoint instructed is the reply
the SDK carries back, not a failure of the activation that produced it: the start interceptor
finishes that activation as succeeded, so the Run runs on to the caller's recorded failure, while a
start that failed any other way still fails its activation. Which fields of each message reach the
SDK is the Driver-reach table in
`internal/execution/typed.go`; the interpreter reads only the fields that table names realized, and
`CommandTypes` names the command types this Driver realizes for `DeriveProfile`. Every registered
Nexus operation reads its input as a `RawValue`, since a schedule command carries any payload.
The workflow implementation receives arbitrary SDK arguments through `converter.EncodedValues`,
then rejects workflow types outside that allowlist before reservation admission.

An activity entrypoint (`ActivityActivation`) runs as a standalone activity, the one a controller's
`StartActivityExecution` starts, or as the activity a workflow entrypoint's schedule command
schedules. Its script is a linear sequence of attempt groups. Each group has zero or one
`ActivityHeartbeat` prefix and exactly one terminal disposition. A `Finish` completes its attempt with its result, whatever value that is, and the
activity closes when the server accepts that completion. An `ActivityAttemptFailure` fails its attempt with the application failure it
carries, which the server retries unless the failure says otherwise. An
`ActivityAttemptCancellation` answers its attempt as canceled, which a worker may do only for a
cancellation the server asked for: the attempt heartbeats until the server answers a heartbeat with
the requested cancellation, and then hands the SDK a canceled error. The SDK tells Temporal an
attempt is canceled only for a delivery the server asked to cancel and reports any other canceled
error as a failure, so the worker never offers a cancellation the SDK would not send. An
`ActivityHeartbeat` invokes SDK `RecordHeartbeat` with each carried Payload as a raw converter value,
preserving its bytes and metadata. The attempt record's `heartbeat_invoked` reports invocation only;
the SDK returns no receipt, and a separate declared Describe read must establish server receipt.
`ActivityAttemptWithholding` in `CONTEXT` mode offers nothing: the attempt waits for its context to end, and when its
deadline, the start-to-close or schedule-to-close timeout the server applies, ends it, the SDK sends
nothing and the server times the attempt out; any other end of the context is a refusal. Its start
must carry a positive start-to-close or schedule-to-close timeout. `SDK_PENDING` mode requires a
standalone start with a positive heartbeat timeout, or the prepared activity-local external
settlement declaration, and returns the exact SDK `ErrResultPending`.
The SDK sends no answer RPC, while the reservation settles immediately with `PENDING`.
An external settlement publishes the actual pending attempt before exposing its single-assignment
publication slot. The controller awaits that slot before a By-ID answer, and before requesting
cancellation on the cancellation path. Namespace name, activity ID and the execution run learned
from the start must match the actual SDK delivery; By-ID's synthesized zero attempt is no worker
cancellation authority. The external basis does not select or credit a timer.
Preparation reserves one activation per group, so every attempt is its own activation under its own reservation, and
the script's values carry across the attempts as a Nexus handler's do across its deliveries, so a
later attempt's guard reads what an earlier one admitted.

A workflow-scheduled activity is delivered through its workflow's start. Preparation reserves its
attempts on the carrier that reserves the workflow and routes the schedule command to the
activity entrypoint whose type and task-queue role it names. When the workflow is admitted, the
Session prepares one header entry per such command, `temporal-testpilot-reserved-scheduled-activity-v1`,
naming the workflow's activation and the command, and the workflow's outbound `ExecuteActivity`
interceptor writes it into the command's header; a command that reaches no activity entrypoint is
issued unrouted, for an ordinary worker. The attempts carry the entry, the Driver offers them to the
Session that prepared it, and they are admitted as a standalone activity's are, except that they
belong to the workflow run that scheduled them, which their outcomes name as the run, and share
the activity ID the SDK gave the command. When the workflow closes, the attempts declared after the
last one delivered are settled as not needed, as a standalone activity's are when the server
reports it closed.

A queue that names activity types registers one dynamic activity. The inbound activity interceptor
rejects a type outside the allowlist and admits the task against the reservations its start request
carried. Three identities stay apart and are recorded together. The activity run the start answered
with is the logical operation, pinned against the delivery in either order, and every attempt
belongs to it. The SDK attempt is the server's number, never the coordinate's attempt, which is the
carrying instruction's. The delivery identity is the SHA-256 digest of the task token, a bounded
opaque name of the first delivery of the attempt. The attempt numbered `first + N - 1`, as the
entrypoint's `attempt_numbering` declares, is the activation of the Nth reservation and performs the
Nth group, whatever order the attempts reach the worker in, and an attempt interprets nothing
until every earlier attempt has settled.

Each declared attempt's reservation settles with an outcome whose `activity_attempt` says what the
worker did, and the Run's reservation event carries it. `response` is the answer the worker offered
Temporal, never the server's acceptance of it: the worker hands its answer to the SDK, which sends
it afterwards, and the send can fail. The outcome's status says whether the activation did what the
Program declared, so the two are read together.

#### The lifecycle of one declared attempt

A started activity has one reservation per attempt its script declares, in attempt order. Each is in
exactly one of these states, and settles at most once:

| State | Meaning | Recorded in the Run |
| --- | --- | --- |
| reserved | no delivery of the attempt has reached the worker | nothing yet |
| admitted | the first delivery consumed the reservation and the attempt is under way | nothing yet |
| offered-completed | its `Finish` ran and the worker offered the completion | succeeded, `OFFERED_COMPLETED`, run, SDK attempt, delivery |
| offered-failed-retryable | its `ActivityAttemptFailure` ran and the worker offered a failure the server may retry | succeeded, `OFFERED_FAILED_RETRYABLE`, run, SDK attempt, delivery |
| offered-failed-non-retryable | the same, with a failure the server does not retry | succeeded, `OFFERED_FAILED_NON_RETRYABLE`, run, SDK attempt, delivery |
| offered-canceled | its `ActivityAttemptCancellation` ran: the server answered the attempt's heartbeat with the requested cancellation, and the worker offered the canceled answer | succeeded, `OFFERED_CANCELED`, run, SDK attempt, delivery |
| withheld | its `ActivityAttemptWithholding` ran and the attempt's deadline ended it unanswered | succeeded, `WITHHELD`, run, SDK attempt, delivery |
| pending | its `ActivityAttemptWithholding` returned SDK `ErrResultPending` immediately | succeeded, `PENDING`, run, SDK attempt, delivery |
| refused | the worker performed nothing declared and offered its own non-retryable failure | SDK failure `umpire_worker` with the cause, `REFUSED`, run, SDK attempt, delivery; then the Run is incomplete |
| released-not-needed | the server reported the activity closed before any attempt was delivered for it | canceled, `NOT_NEEDED`, the run only, caused by the last recorded attempt |
| never-seen | the Run released the reservation while nothing was delivered for it and the server had said nothing | canceled with no attempt fact, which fails the Run unrecorded |

The allowed transitions, and what brings each about:

| From | To | Cause |
| --- | --- | --- |
| reserved | admitted | the first delivery of the SDK attempt whose number is the reservation's position, in the activity run the start answered with |
| admitted | offered-completed, offered-failed-retryable, offered-failed-non-retryable | the attempt's instruction ran, after every earlier attempt of the activity settled |
| admitted | offered-canceled | the attempt's instruction ran, after every earlier attempt of the activity settled, and the server answered a heartbeat of the attempt with the requested cancellation |
| admitted | withheld | the attempt's instruction ran, after every earlier attempt of the activity settled, and the attempt's deadline ended its context |
| admitted | pending | the admitted group returned SDK `ErrResultPending`, after every earlier attempt settled |
| admitted | refused | the instruction is disabled, the Run canceled the reservation, the delivery's context ended, a cancellation to answer was never requested before it did, or the SDK or the Driver failed |
| reserved | released-not-needed | the server answered the worker's long poll for the activity's outcome, or the workflow that scheduled the activity closed, and the reservation is after the last attempt admitted |
| reserved | never-seen | the Run canceled the reservation |

Everything else is refused and changes no state:

- A delivery of an attempt already admitted or settled, under whatever delivery identity, runs
  nothing and settles nothing. It waits for the first delivery's answer and returns the same one.
  A canceled answer is returned to it only once the server has asked that delivery to cancel too,
  which it heartbeats to learn; if its context ends first it is refused, and nothing is settled.
- An offered answer releases nothing, whichever it is. Only the server closing the activity does.
  If the server loses the answer and redelivers the attempt, the worker answers again; if it timed
  the attempt out and issues the next, that attempt finds its reservation and runs its instruction.
- A reservation before the last admitted attempt is never released as not needed. The activity
  closing does not explain an attempt the worker never saw.
- An attempt delivered for a released or never-seen reservation is refused non-retryably.
- An attempt past the last one the script declares has no reservation. It is refused
  non-retryably and never answered from another attempt's record, and the Session reports it as
  the Driver diagnostic `activity_attempt_undeclared`, which is its only trace.
- An attempt that names another activity run than the one the start answered with, or than the one
  an earlier attempt named, is refused as a conflict before anything is consumed or replayed, and
  the Session reports the Driver diagnostic `activity_run_crossed`.

The worker asks the server once per started activity, and only while a later attempt is still
reserved after an attempt settled: `PollActivityExecution` by the activity's namespace, ID and run,
repeated while it returns no outcome, until the Session closes. The answer counts only when the
run it names is the run that was asked about. If it names another, the later reservations stay
reserved and the Session reports the Driver diagnostic `activity_closure_crossed`; if the poll
fails, they stay reserved and it reports `activity_closure_unobserved`.

The Run records the attempts of one activity in attempt order however their reservations settle,
because the scheduler observes them with one waiter in that order.

The worker refuses with a non-retryable application failure of type `umpire_worker` whose cause is
the reason. Canceling a reservation cancels the attempt's own context and sends Temporal no
cancellation request, so the worker offers a failure, never a cancellation, and the Run says the
same. `Validate` rejects, with no I/O, an activity reservation carried by
anything but its standalone activity start or its scheduling workflow's start, and a start that does
not name the worker's namespace and task queue by their binding identities. A declared heartbeat
prefix invokes the SDK once; cancellation separately heartbeats every 100 ms until the server asks
for cancellation. The worker caps the SDK heartbeat throttle at that period. Context withholding
requires a positive start-to-close or schedule-to-close request timeout; SDK-pending withholding
requires a standalone start with a positive heartbeat timeout, including on the actual SDK delivery,
unless its prepared external-settlement declaration supplies the timer-free basis. A reset
settlement keeps the heartbeat basis: its held attempt publishes as an external one does, the
controller resets it, and the timer that ends the attempt applies the reset; the server's next
delivery is attempt 1 under a new task token, which the ledger admits under the declared fresh
reservation.
The pending answer settles the local reservation immediately without offering a completion,
failure, or cancellation to the server.

Controller code reserves worker activations before dispatch and creates a `Carrier` from the
prepared carrier plan, `CreateCarrier` for a workflow start and `CreateActivityCarrier` for an
activity start. `Carrier` delegates route injection, start-response pinning and trigger
terminal release to the delivery ledger. The worker
validates callback URLs, resolves the SDK system callback against the trusted configured base, and
builds the protocol completion effect. Only that generic effect crosses the package boundary through
`HandleFactory`; its callback data remains opaque and the resulting opaque handle is published
through the Run's handle bridge.

Each actual workflow, activity or Nexus-handler interpretation constructs a fresh private
`temporal/internal/activation.State` from its prepared entrypoint. `Evaluate` owns guard/input
reference resolution; `Admit` validates and atomically retains owned outcome fields. Both operations
consume the public plan methods' returned work, including failure charges, within the prepared
runtime ceiling. The state contains no SDK objects or completion handles and is used serially.

The worker traverses the prepared DAG and owns SDK dispatch, Nexus futures, Await timeouts,
cancellation and terminal responses. Workflow interpretation keeps SDK context checks and uses a
background Go context for bounded pure operations, including admission of canceled SDK outcomes.
Replay reconstructs activation state from the prepared plan and replayed SDK results; Nexus
redelivery uses the existing result cache without creating another interpretation. Delivery identity,
Stop, late publication checks and opaque completion authority remain with the Session and ledger.
