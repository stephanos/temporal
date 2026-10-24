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
TimedOut. StartToClose or Heartbeat expiry while cancellation is pending closes Canceled;
other expiries close TimedOut. Ordinary and nonexpiring edges have empty facts. Tick is
disabled with no live budget, including after closure.

Atomic Cancel closes immediately. Cooperative Scheduled Cancel also closes immediately;
Started Cancel records pending without changing any budget. A fresh distinct pending
Cancel rejects FailedPrecondition with an exact self-loop; same-request-ID replay has no
outcome promise. Heartbeat preserves pending and never closes. Current pending
`RespondCanceled(attempt)` closes Canceled; Complete and nonretryable failure can win.
Pending retryable failure closes Canceled even at the cap, without retry or increment.
Nonpending retryable failure retries below the cap and otherwise fails. Stale Poll
retains FailedPrecondition. Stale live acknowledgement selects NotFound as an independent
attempt-token identity policy; current unsolicited and terminal worker actions remain
disabled. These choices follow the approved cancellation boundary contract, independently
of observed server responses.

`fair_worker` selects exact Poll(1)..Poll(maximum_attempts) and RespondCompleted values,
with one weak-fairness obligation per value. Every other ordinary action remains allowed.
The independent `eventually_closed` claim applies even when every timeout is absent.
Separate one-transition claims scope the two concrete failure booleans and preserve the
original nonpending retry promises alongside the cooperative exception.

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
closing or retry witnesses in their atomic scope. Cooperative and larger deadline Worlds
are checked separately; verification records their exact counts and backend evidence.

`temporal/feature/activity/compatibility/` retains all 15 original artifacts and their
independent SHA256 authority. The final one-attempt atomic World has one-tick waiting
and running budgets. Projection validates full state, maps each Tick using its singleton
expiry fact and erases only an Accepted, empty-fact, unchanged Started Heartbeat.
Failure payload assignments deduplicate after projection. Exact comparison retains one
start, six states and eleven old visible edges, including all four terminal Cancel
NotFound loops; changing an old edge fails the same relation. Original bytes, manifest
and verifier remain unchanged.

Live constructors retain actual admitted maximum_attempts with all four timeouts absent
and atomic cancellation. The production runtime default uses the same policy at maximum
three. Request timeout/protocol admission, heartbeat and acknowledgement program rows,
and cancellation observation evidence remain downstream work. The existing unheld
atomic cancellation refusal remains exact. A checker World grants no live capability.

See the [verification record](../../../../.flow/artifacts/fn-237-activity-timeouts-heartbeat-and/activity-policy-verification.md) for selected World counts, actual TLC/replay controls, exact fairness lists and timing. The [retained lifecycle projection](compatibility/README.md) specifies every erased field and edge.
