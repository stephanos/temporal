# Activity deadlines and cancellation

The independent `temporal/feature/activity/model.rs` declares a retry World with
`maximum_attempts` one through three, four optional `Ticks(1..=3)` deadlines and
`asynchronous_cancel`. State carries phase, attempt, a pending cancellation bit and
four optional `Remaining { ticks }` countdowns. The explicit offline fixture factory
checks each configuration field. Raw `World::new` remains infallible; authored bounds
and arming invariants independently check native states and updates. Source binding
rejects configured zero. Terminal transitions clear every budget and the pending bit.

| Budget | Arming and reset |
| --- | --- |
| ScheduleToStart | Armed initially and on retry; disarmed on Poll. |
| StartToClose | Armed on Poll; preserved by Heartbeat. |
| ScheduleToClose | Armed initially; spans Poll, retries, Heartbeat and pending Cancel. |
| Heartbeat | Armed on Poll and reset only by Heartbeat. |

The single internal `Deadline::Tick` decrements every live budget once, without
underflow, then selects ScheduleToClose > StartToClose > Heartbeat > ScheduleToStart.
Exactly the selected `ExpiredDeadline` fact records the trigger. A global expiry closes
TimedOut. StartToClose or Heartbeat expiry while cancellation is pending closes TimedOut.
Nonpending local expiry below the attempt cap schedules exactly the next attempt when
ScheduleToClose has not expired. It uses the worker-failure retry transition: waiting is
rearmed from the selected World, running/heartbeat are cleared, and the decremented
global budget is preserved. The next accepted Poll arms the new attempt timers. At the
cap, local expiry closes TimedOut; ScheduleToStart expiry always closes TimedOut. Ordinary and nonexpiring edges have empty facts. Tick is
disabled with no live budget, including after closure.

Atomic Cancel closes immediately. Cooperative Scheduled Cancel also closes immediately;
Started Cancel records pending without changing any budget. A fresh distinct pending
Cancel rejects FailedPrecondition with an exact self-loop; same-request-ID replay has no
outcome promise. Heartbeat preserves pending and never closes. Current pending
`RespondCanceled(attempt)` closes Canceled; Complete and nonretryable failure can win.
Pending retryable failure closes Failed even at the cap, without retry or increment.
Nonpending retryable failure retries below the cap and otherwise fails. Stale Poll
retains FailedPrecondition. Stale live acknowledgement selects NotFound as an independent
attempt-token identity policy; current unsolicited and terminal worker actions remain
disabled. The corrected pending failure and timeout outcomes follow the approved pinned API
contract below, independently of observed server responses.

`fair_worker` selects exact Poll(1)..Poll(maximum_attempts) and RespondCompleted values,
with one weak-fairness obligation per value. Every other ordinary action remains allowed.
The independent `eventually_closed` claim applies even when every timeout is absent.
Separate one-transition claims scope the two concrete failure booleans and preserve the
original nonpending retry promises alongside pending cancellation termination.

For ScheduleToClose=Some(Ticks(B)), complete reachable graphs check live presence and
1<=R<=B, universal nonincrease, strict descent on every live Tick, global expiry at one,
terminal permanence and the absence of terminal Ticks. After k actual Ticks before closure,
R<=B-k; positivity allows at most B-1 live-after Ticks and the closing Tick adds at most
one. Thus every interleaving contains at most B actual Ticks, regardless of retries,
heartbeats or rejected actions. Separate transition-availability controls detect missing
Ticks, which the numerical upper bound alone cannot exclude. No history counter or cut
supplies this proof.

The default fixture remains atomic with three attempts, one-tick ScheduleToStart and
StartToClose, and absent global/heartbeat deadlines. Its complete graph has 18 states,
63 emitted choices from 216 evaluations, 153 disabled evaluations and depth six.
Its 17 registered claims pass natively. `Cases(3 * 7)` retains the 21 scheduled/started
closing or retry witnesses in their atomic scope, including timeout retries below the cap. Cooperative and larger deadline Worlds
are checked separately; verification records their exact counts and backend evidence.

`temporal/feature/activity/compatibility/` retains all 15 original artifacts; the active
gate compares their independently retained lifecycle semantics. The final one-attempt
atomic World has one-tick waiting and running budgets. Projection validates full state, maps each Tick using its singleton
expiry fact and erases only an Accepted, empty-fact, unchanged Started Heartbeat.
Failure payload assignments deduplicate after projection. Exact comparison retains one
start, six states and eleven old visible edges, including all four terminal Cancel
NotFound loops; changing an old edge fails the same relation. Original bytes and the
semantic verifier remain unchanged.

Live constructors retain actual admitted maximum_attempts with all four timeouts absent
and atomic cancellation. The production runtime default uses the same policy at maximum
three. Request timeout/protocol admission, heartbeat and acknowledgement program rows,
and cancellation observation evidence remain downstream work. The existing unheld
atomic cancellation refusal remains exact. A checker World grants no live capability.

The [original pre-correction verification record](../../../../.flow/artifacts/fn-237-activity-timeouts-heartbeat-and/activity-policy-verification.md) retains the original selected World counts, TLC/replay controls, fairness lists and timing; it is historical evidence, not proof of the corrected transitions. The [retained lifecycle projection](compatibility/README.md) specifies every erased field and edge.

## Temporal deadline pattern

This is the feature-owned authoring pattern selected by the
[countdown design](../../../design/deadline-countdowns.md#temporal-layer-and-independence).
The executable consumer is [the activity Model](model.rs); the
[independent protocol Model](../../shared/protocol/activity/model.rs) authors its
semantics separately. Generic countdown arithmetic supplies no Temporal arming,
retry, cancellation or expiry-priority policy.

The pattern consumes effective checker World budgets. `None` means that budget is
absent; a present configuration is positive, and every armed budget remains positive
between transitions. An expiring step clears or rearms zero before publishing its
successor. All-absent and other non-API-realizable Worlds remain abstract checker
scopes. Raw Start admission and its documented defaults are a separate boundary;
this pattern neither substitutes a timeout for `None` nor clamps a supplied value.

| Lifecycle step | Budget rule |
| --- | --- |
| Initial Scheduled state | Arm configured ScheduleToStart and ScheduleToClose; leave StartToClose and Heartbeat absent. |
| Accepted current Poll | Clear ScheduleToStart; arm configured StartToClose and Heartbeat; preserve ScheduleToClose. |
| Started Heartbeat | Reset only configured Heartbeat; preserve phase, attempt, pending cancellation and the other budgets. |
| Retry | Increment the attempt once and enter Scheduled; rearm configured ScheduleToStart, clear StartToClose/Heartbeat and preserve ScheduleToClose, including any decrement already performed by a retrying Tick. The next Poll arms the new attempt's timers. |
| Terminal transition | Clear every remaining budget and the pending cancellation bit. |

One enabled `Deadline::Tick` decrements every armed budget once with saturation,
then selects one expiry in this order: ScheduleToClose, StartToClose, Heartbeat,
ScheduleToStart. It emits exactly that singleton `ExpiredDeadline` fact. Global or
ScheduleToStart expiry closes TimedOut. A local StartToClose/Heartbeat expiry retries
only when cancellation is not pending and another attempt remains; otherwise it
closes TimedOut. The global budget is armed only initially and never replenished by
Poll, retry, Heartbeat or Cancel. Nonexpiring and ordinary actions emit no expiry
fact; no Tick is enabled after closure or with every budget absent.

The pending-state rules apply only to Started cooperative cancellation
(`asynchronous_cancel = true`): Cancel preserves budgets, Heartbeat preserves the
pending bit, worker failure closes Failed without retry, and local expiry closes
TimedOut. Current pending acknowledgement closes Canceled; completion may still win.
The existing atomic mode and [lifecycle compatibility control](compatibility/README.md)
remain in scope. Removing the mode flag belongs to the separately gated
[cancellation task](../../../../.flow/tasks/fn-253-reusable-deadline-countdowns-for-models.10.md),
not to adoption of this pattern.

Current implementation owners are `initial` and `worker` for arming/reset,
`deadline` plus `decrement` for Tick/priority, `time_out_attempt` for local expiry,
`retry` for rearming, and `time_out` plus `clear_terminal` for terminal clearing.
`policy_in_range`, `attempt_within_policy`, `countdowns_within_budget`,
`deadline_arming` and `pending_is_cooperative` check live-state constraints;
`countdowns_decrement`, `tick_priority`, `heartbeat_preserves_attempt_and_global`,
`attempt_changes_only_on_retry`, `global_never_replenished` and
`tick_descends_and_expires` check transition obligations. The
[consumer refactor](../../../../.flow/tasks/fn-253-reusable-deadline-countdowns-for-models.7.md)
will adopt countdown library calls while preserving these obligations and the
corrected behavior.

Authority is the existing pinned API
`go.temporal.io/api@v1.63.6-0.20260909222256-20151aa90480`:
[timeout fields and retry limits](https://github.com/temporalio/api-go/blob/20151aa90480/workflowservice/v1/request_response.pb.go#L16585-L16611),
[cooperative cancellation](https://github.com/temporalio/api-go/blob/20151aa90480/workflowservice/v1/service_grpc.pb.go#L2771-L2777),
and [distinct terminal statuses](https://github.com/temporalio/api-go/blob/20151aa90480/enums/v1/activity.pb.go#L100-L114).
The [approved retry correction](#approved-local-timeout-retry-correction-fn-24615)
and [approved pending-outcome correction](#approved-pending-cancellation-outcome-correction-fn-24613)
below retain the owner decisions and supporting public timeout documentation.
The expiry ordering and abstract retry scheduling retain the existing Model choices;
the approvals cover their stated correction cells. This pattern adds no backoff,
clock inference, raw-request admission or live capability claim.

## Approved local timeout retry correction (fn-246.15)

The [owner approval](../../../../.flow/artifacts/fn-246-realization-as-a-checked-refinement-a/planning/timeout-retry-correction/owner-approval.json)
authorizes only the two nonpending local-expiry cells below the attempt cap. Primary
contract provenance is `go.temporal.io/api@v1.63.6-0.20260909222256-20151aa90480`,
`workflowservice/v1/request_response.pb.go:16585-16611` and `service_grpc.pb.go:266-271`,
and the [public activity timeout contract](https://docs.temporal.io/encyclopedia/detecting-activity-failures#start-to-close-timeout).
Server implementation is corroboration only. No timeout defaults, backoff, clock inference,
World domain or cancellation policy is introduced; absent ScheduleToStart admission stays
with the independent protocol task. The original disagreement and fn-237 records remain intact.

The renewed controls select maximum attempts 1/2/3, optional positive `Ticks(1..=3)`
parameters and all seven full-state fields. Native exact-edge checks cover each local
cause at every attempt, pending/nonpending, absent/positive/tied global budgets,
StartToClose/Heartbeat ties, rearmed waiting/attempt timers, repeated timeouts,
completion after retry, stale/current Poll and terminal rejection. The existing
compiled/interpreted/TLC fixture harness explores complete selected Worlds with a
4,096-state ceiling and no depth cut; it retains all 17 laws, fairness controls and
the global Tick bound. Exact retry traces retain Accepted and a singleton timeout
fact, independently of unchanged aggregate counts. Source mutants retain the old
terminalization and exercise erroneous cap/pending retry, global reset/priority,
waiting reset, attempt increment and replaced/missing timeout facts. A timeout fact
cannot be erased to present a worker-failure retry.

Only current semantic snapshots are renewed. The original max-one compatibility
checkpoint still compares one start, six states and eleven visible edges, including
the changed-old-edge negative control. The current default remains 18 states/63 edges;
its rows and two timeout-retry witnesses change. Required verification from `model/`:

```sh
mise exec -- cargo test -p umpire-temporal-tests --test fixtures
UMPIRE_BACKENDS=require mise exec -- cargo test -p umpire-temporal-tests --test fixtures -- --nocapture --test-threads=2
mise exec -- cargo test -p umpire-temporal --test realization named_guest_adapter_preserves_the_independent_report_and_ordered_rows
```

TLC uses the pinned `tla2tools` 1.7.4 artifact (TLC 2.19), SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
Apalache is opt-in and provides no required correction evidence. Joined guest
publication/freshness, current host checks and fresh TLC-required `make model-validate`
remain conductor-owned before task closure and independent protocol admission.

## Approved pending cancellation outcome correction (fn-246.13)

The [explicit owner approval](../../../../.flow/artifacts/fn-246-realization-as-a-checked-refinement-a/task13/planning/approved-api-correction.json)
corrects only pending cancellation's worker-failure and local-timeout outcomes. The pinned
primary API is `go.temporal.io/api@v1.63.6-0.20260909222256-20151aa90480`:
`workflowservice/v1/service_grpc.pb.go:2771-2777` specifies cooperative cancellation,
heartbeat delivery of its pending bit and explicit acknowledgement for Canceled;
`enums/v1/activity.pb.go:100-114` retains separate Failed, Canceled and TimedOut statuses.
A pending intent does not turn a worker failure or an expired deadline into acknowledgement.
The former chosen-policy artifact and original verification records remain historical evidence.

Pending retryable worker failure now closes Failed, with no retry or attempt increment.
Pending StartToClose and Heartbeat expiry close TimedOut, retaining the exact singleton
expiry cause; global priority is unchanged. Scheduled Cancel and current pending
RespondCanceled still close Canceled. All terminal budgets and the pending bit clear.
The task15 nonpending local-timeout retries and original atomic six-state/eleven-edge
projection remain unchanged. All-absent checker Worlds remain abstract scopes; this
correction adds no raw Start admission, hidden timeout defaults or live capability.

Native exact-edge controls cover pending failures and both local expiry causes at every
attempt. Source-only old-Canceled mutants must violate independently authored failure
progress and timeout-priority laws. Required TLC controls replay both concrete failure
payloads and both pending local-timeout causes with exact outcome/facts, and reject
changed/missing/duplicated causes plus the former Canceled policy and erroneous retries.
Only current generated fixtures are renewed. Backend records and final joined validation
are retained by the task conductor; these fixture controls claim no TLC refinement proof.
