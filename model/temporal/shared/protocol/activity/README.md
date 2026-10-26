# Standalone activity protocol

`model.rs` independently declares the selected standalone protocol. Its full state
contains queue, attempt, execution and heartbeat countdowns. `view.rs` projects only
public classifications. A `Snapshot` constrains candidate views through
`Observed::Unknown` or `Observed::Known`; it cannot construct a `Model` or a
countdown. `Known(None)` run state differs from an unobserved run state.

The primary API pin is `go.temporal.io/api`
`v1.63.6-0.20260909222256-20151aa90480`. The independent protocol uses its public
declarations and the discrete laws below. Feature transitions and server
implementation are comparison inputs, never sources of protocol edges.

## Primary evidence and the approved correction

| Source at the pinned revision | Authority |
| --- | --- |
| [Standalone status enum, lines 32–63](https://github.com/temporalio/api-go/blob/20151aa90480/enums/v1/activity.pb.go#L32-L63) | Running substate, completion race, failure after cancellation, explicit cancellation acknowledgement, local timeout retry and terminal timeout categories. |
| [Start request, lines 16585–16611](https://github.com/temporalio/api-go/blob/20151aa90480/workflowservice/v1/request_response.pb.go#L16585-L16611) | Waiting timeout default, nonretryable waiting expiry, retryable attempt expiry, spanning retry limit and required execution timeout. |
| [Heartbeat RPC, lines 266–279](https://github.com/temporalio/api-go/blob/20151aa90480/workflowservice/v1/service_grpc.pb.go#L266-L279) | Heartbeat timeout ends an attempt; retry depends on policy; heartbeat response conveys pending cancellation. |
| [Worker replies, lines 287–331](https://github.com/temporalio/api-go/blob/20151aa90480/workflowservice/v1/service_grpc.pb.go#L287-L331) | Complete, Fail and Canceled are distinct operations; invalid expired or completed task tokens have NotFound evidence. Surrounding workflow-history text does not establish standalone history. |
| [Cancellation RPC, lines 799–805](https://github.com/temporalio/api-go/blob/20151aa90480/workflowservice/v1/service_grpc.pb.go#L799-L805) | Request acceptance, notification and worker acknowledgement are separate events. |
| [Execution info, lines 258–379](https://github.com/temporalio/api-go/blob/20151aa90480/activity/v1/message.pb.go#L258-L379) | Status, run state, scheduling-based attempt number, conditional last failure and cross-attempt heartbeat count. |
| [Describe request, lines 16889–16912](https://github.com/temporalio/api-go/blob/20151aa90480/workflowservice/v1/request_response.pb.go#L16889-L16912) | Outcome/failure inclusion is optional; long polling can skip state changes. |
| [RetryPolicy, lines 535–552](https://github.com/temporalio/api-go/blob/20151aa90480/common/v1/message.pb.go#L535-L552) | Maximum attempts and retryability are separate constraints; the public domain includes unlimited retries and backoff beyond this finite selection. |

The pinned enum specifies Failed after pending worker failure and TimedOut after
pending attempt expiry. Historical fn237 boundary policy instead selected Canceled
for these cells. The owner's
[API-aligned correction approval](../../../../../.flow/artifacts/fn-246-realization-as-a-checked-refinement-a/task13/planning/approved-api-correction.json)
authorizes precisely this correction. Scheduled cancellation and explicit running
worker acknowledgement still produce Canceled. Task15's approved nonpending local
timeout retry and the original atomic compatibility checkpoint retain their scope.

Published [request-cancel guidance, lines 25–33](https://github.com/temporalio/documentation/blob/56a8a19b04e7fe642e4f8da03ff997afc42b40bd/docs/encyclopedia/activities/activity-operations/request-cancel.mdx#L25-L33)
describes cancellation at a non-heartbeating attempt boundary. This conflicts with
the pinned enum's explicit acknowledgement requirement. The selected protocol
follows the enum under the approval; it does not claim the publications agree.

Minimal discriminators use cooperative cancellation, maximum attempts two and
attempt one. Deliver then RequestCancel then retryable Fail closes Failed without
increment. With StartToClose one, Deliver then RequestCancel then Tick closes
TimedOut with StartToClose. With Heartbeat one and no competing attempt/global
deadline, the corresponding Tick closes TimedOut with Heartbeat. The historical
policy selected Canceled for all three traces. Deliver then RequestCancel then
AcknowledgeCancel(1) still closes Canceled.

## Selected state, operations and laws

Queued starts at attempt one with no failure, pending request or terminal outcome.
Its waiting budget and spanning budget arm from effective parameters. Accepted
current-attempt Deliver enters Executing, clears waiting and arms attempt and
heartbeat budgets. Each retry increments once, returns to Queued, rearms waiting
and clears attempt/heartbeat budgets. Spanning Remaining survives retry unchanged
except for consumption by a preceding Tick. These laws realize the published
per-task waiting, per-attempt running and whole-execution intervals. See the
[pinned timeout definitions, lines 23–48 and 96–137](https://github.com/temporalio/documentation/blob/56a8a19b04e7fe642e4f8da03ff997afc42b40bd/docs/encyclopedia/detecting-activity-failures.mdx#L23-L48).

Complete closes Completed. Nonretryable Fail closes Failed. Retryable Fail retries
only below the selected cap with no pending request; otherwise it closes Failed.
The last failure remains separate from terminal outcome, including after a later
successful completion. Heartbeat remains Executing, preserves pending state and
every other budget, and resets only its heartbeat interval.

Cooperative running RequestCancel records pending without resetting a timer.
Scheduled cancellation closes immediately. Current-attempt AcknowledgeCancel
requires Executing and pending. `HeartbeatReply` represents notification;
`Acknowledgement` represents separately correlated worker evidence. Neither type
converts a snapshot into state or produces evidence from a desired transition.

The selected discrete Tick consumes every armed budget once, then chooses exactly
one expired cause in order ScheduleToClose, StartToClose, Heartbeat,
ScheduleToStart. Global and waiting expiry close TimedOut. Local expiry retries
below cap without pending cancellation, otherwise closes TimedOut. Exact priority,
single-cause facts, decrement-before-selection, positive reached countdowns and
clearing all timers/pending on closure are authored abstraction laws. The public
API does not prescribe these representation details or a simultaneous-timer order.
Ordinary actions and nonexpiring Ticks carry no expiry fact. Without a live budget
Tick has no accepted progression. Closed states cannot reopen.

Fresh repeated pending Cancel selects FailedPrecondition; terminal Cancel selects
NotFound. Stale Deliver selects FailedPrecondition and stale acknowledgement selects
NotFound. These exact selected rejection cells preserve full state. Public
[operation error categories](https://github.com/temporalio/documentation/blob/56a8a19b04e7fe642e4f8da03ff997afc42b40bd/docs/encyclopedia/activities/activity-operations/index.mdx#L68-L75)
do not enumerate every cell. Same-request-ID replay, unsolicited acknowledgement,
pause, terminate and reset are outside the selected actions. `asynchronous_cancel`
false selects the retained atomic abstraction, not a public Start request option.

The authored invariants check selected attempts, outcome/closure equivalence,
pending only in cooperative Executing, exact timer arming and positive bounded
Remaining. Postconditions check terminal permanence, rejection identity, retry-only
attempt increments, heartbeat reset isolation, retry timer rearming, Tick descent,
global expiry dominance and pending failure/expiry outcome separation from ACK.
These are proof obligations, not a claim that a backend run has passed. Concrete
native/TLC counts and verification receipts belong to the task evidence.

## Raw requests, effective budgets and observations

`Parameters` validates exact effective values without defaults. Attempts range
from one through three; each present timeout ranges from one through three ticks.
Absent and present-zero values remain distinct; zero is refused. Timeoutless
effective Worlds support abstract progress checks but do not establish an accepted
raw standalone Start.

`RawStart::into_effective` is the named boundary that applies the pinned standalone
waiting default from ScheduleToClose and refuses absent StartToClose together with
absent ScheduleToClose. General pinned documentation instead describes an infinite
[default waiting timeout](https://github.com/temporalio/documentation/blob/56a8a19b04e7fe642e4f8da03ff997afc42b40bd/docs/encyclopedia/detecting-activity-failures.mdx#L59-L79).
That discrepancy remains explicit. The adapter does not silently alter effective
checker Worlds or claim complete normalization of all Temporal defaults. Retry
backoff and wall-clock timing are outside this selected immediate-retry abstraction.

Every full-state field has explicit accounting.

| Full-state field | View accounting |
| --- | --- |
| `lifecycle` | Supplies Scheduled/Started run state or absent closed run state. |
| `attempt` | Supplies the current attempt. |
| `cancellation_pending` | Supplies pending classification and CancelRequested run state. |
| `terminal_outcome` | Supplies execution status and independent outcome classification. |
| `retained_failure` | Supplies a candidate failure classification independently of terminal outcome. Actual snapshots use Unknown unless requested, present and classifiable. It remains source-owned when no such observation exists. |
| `queued_budget` | Latent per-queue countdown, excluded from view. |
| `attempt_budget` | Latent per-attempt countdown, excluded from view. |
| `execution_budget` | Latent spanning countdown, excluded from view. |
| `heartbeat_budget` | Latent heartbeat countdown, excluded from view. |

Public equality therefore permits distinct full states. Full-state comparison must
account for every latent field; view matching cannot erase budget changes or
authorize an evidence-free timer edge. Failure history alone does not establish
the cause of a terminal transition. Snapshot decoding and actual worker evidence
production remain with their separate binding owners.

The retained classification is a selected protocol-history abstraction: worker
failure and each selected expiry replace it, and later success preserves it. The
pinned `last_failure` description does not determine whether global/waiting expiry
updates that protobuf field. This abstraction therefore makes no such guarantee.
A snapshot cannot manufacture this classification from terminal timeout status,
configured duration or an old failure; it stays Unknown without actual applicable
and classifiable evidence.

## Checked finite scopes

The authored fixture retains starts, states, exact outcome/fact-bearing rows and
all sixteen own-law verdicts for each of twelve selected Worlds under
`expected/worlds/`. `worlds.txt.snap` identifies their exact effective parameters,
4096-state bound and raw-realizable versus abstract timeout scope. Atomic
compatibility cancellation is separately labelled. The default cooperative
maximum-two fixture has 23 states and 80 edges; the largest selected World has
90 states and 750 edges. All twelve native/interpreted graphs are complete.

The required protocol test run passed 22 tests in 45.96 seconds (66.566 seconds
including recompilation). Real TLC checked each selected World's own laws and
complete state-set agreement. A refuted timeout-absence claim replays against the
healthy Model, and a changed expiry fact refuses that replay. Twenty source-only
timer/reset/priority/outcome/fact defects disagree with the native graph; schema,
adapter, erased-budget and present-zero controls are separate checks. The final
native Temporal fixture run passed 62 tests with two inherited opt-in ignores;
both snapshot-ownership tests and strict clippy passed.

These are native/TLC protocol checks and the approved feature-correction checks.
They establish neither online binding conformance nor a TLC refinement mapping.
The original feature lifecycle checkpoint remains independently six states and
eleven edges; this richer protocol retains additional history classes.
