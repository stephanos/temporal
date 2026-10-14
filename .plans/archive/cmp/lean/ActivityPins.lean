import Temporal.Feature.Activity.Standalone
import Umpire.Shared.Test

/-!
# What the standalone activity Model says

The spec's pins for Model 2, in the style of `Temporal/Feature/Nexus/Caller/Tests.lean`. Not
compiled; written so the two known gaps of `StandaloneActivity.lean` (the missing realization and
the status observations) would surface here first.
-/

namespace Temporal.Feature.Activity.Standalone.Tests

open Umpire
open Umpire.Command
open Temporal.Feature.Activity.Standalone

/-! ### The product machine -/

/- Nine phases, and the five the design ends on. -/
#guard activityProduct.table.states.length == 9
#guard activityProduct.ends.length == 5

/- A canceled answer settles only an activity whose cancellation was requested. -/
#guard attemptResultStep { phase := .started } .canceled == []
#guard (attemptResultStep { phase := .cancelRequested } .canceled).map (·.state.phase) == [.canceled]

/- Unlike the Nexus product, a retry is visible: the caller reads scheduled again. -/
#guard (attemptResultStep { phase := .started } (.failed (retryable := true))).map (·.state.phase) ==
  [.scheduled]
#guard (attemptResultStep { phase := .cancelRequested } (.failed (retryable := true))).map
  (·.state.phase) == [.canceled]

/- A paused activity is dispatched to no worker: no product row leaves paused for started. -/
#guard (activityProduct.transitions.filter fun row =>
  row.source.phase == .paused && row.results.any (·.state.phase == .started)).isEmpty

assert_axioms [activityProduct] allowing [propext]

#guard activityProduct.stuck == none

/-! ### The protocol machine -/

private def at' (phase : Phase) (attempts : Fin (attemptBound + 1) := 0)
    (scheduleToClose : Timeout := .unset) (scheduleToStart : Timeout := .unset)
    (startToClose : Timeout := .unset) : ProtocolState :=
  { phase, attempts, scheduleToClose, scheduleToStart, startToClose }

/- Twelve phases, three attempt counts and three deadlines, and the five phases the design ends on. -/
#guard activityProtocol.table.states.length == 12 * (attemptBound + 1) * 2 * 2 * 2
#guard activityProtocol.ends.length == 5 * (attemptBound + 1) * 2 * 2 * 2

/- Eight start requests, one attempt start, four answers, four controls, the fault and four timers. -/
#guard activityProtocol.actionKeys.size == 8 + 1 + 4 + 4 + 1 + 4

#guard activityProtocol.starts == [at' .unstarted]

/- A retryable failure backs the attempt off and is read as scheduled again with the count raised. -/
#guard protocolAttemptResultStep (at' .started 1) (.failed (retryable := true)) ==
  [{ outcome := .accepted, state := at' .backingOff 1, facts := [.statusScheduled, .attemptCount] }]

/- Under a cancel request the same failure settles the activity as canceled. -/
#guard (protocolAttemptResultStep (at' .cancelRequested 1) (.failed (retryable := true))).map
  (·.state.phase) == [.canceled]

/- Under a pause request it lands in paused rather than backing off. -/
#guard (protocolAttemptResultStep (at' .pauseRequested 1) (.failed (retryable := true))).map
  (·.state.phase) == [.paused]

/- A pause of a held attempt is a request; of a scheduled one it takes effect at once. -/
#guard (protocolControlStep (at' .started 1) .pause).map (·.state.phase) == [.pauseRequested]
#guard (protocolControlStep (at' .scheduled) .pause).map (·.state.phase) == [.paused]

/- A control on an activity that is over is not found. -/
#guard protocolControlStep (at' .completed 1) .terminate ==
  [{ outcome := .notFound, state := at' .completed 1, facts := [] }]

/- Each deadline covers its own span. -/
#guard startToCloseStep (at' .scheduled (startToClose := .expires)) == []
#guard (startToCloseStep (at' .pauseRequested 1 (startToClose := .expires))).map (·.state.phase) ==
  [.timedOut]
#guard (scheduleToStartStep (at' .backingOff 1 (scheduleToStart := .expires))).flatMap (·.facts) ==
  [.statusTimedOut (timeoutType := .scheduleToStart)]

#guard activityProtocol.stuck == none

assert_axioms [activityProtocol] allowing [propext]

/-! ### The refinement

`refines: activityProduct` with `map: productOf`: a pause request reads as started, a retry as
scheduled again. -/

#guard activityProtocol.refinement.rejected == none
#guard activityProtocol.refinement.rows.length == activityProtocol.transitions.length

/- The visible retry is the product's retryable-failure row; the pause request is a stutter; the
unpause of a requested pause is a stutter too. -/
#guard activityProtocol.refinement.rows.lookup
  "started-1-unset-unset-unset-attemptResult-failed-true" == some (some "attemptResult-failed-true")
#guard activityProtocol.refinement.rows.lookup "started-1-unset-unset-unset-control-pause" == some none
#guard activityProtocol.refinement.rows.lookup
  "pauseRequested-1-unset-unset-unset-control-unpause" == some none

/- A retryable failure under a pause request is the product's pause. -/
#guard activityProtocol.refinement.rows.lookup
  "pauseRequested-1-unset-unset-unset-attemptResult-failed-true" == some (some "control-pause")

/-! ### The Queries -/

#guard completion.answer.found
#guard nonRetryableFailure.answer.found
#guard retry.answer.found
#guard cancel.answer.found
#guard terminate.answer.found
#guard pauseResume.answer.found
#guard scheduleToStartTimeout.answer.found
#guard startToCloseTimeout.answer.found
#guard terminalHolds.answer.verified
#guard pauseHolds.answer.verified
#guard stoppedWorkerStartsNothing.answer.verified
/- Not vacuous: the scenario performs an attempt start while the worker polls, so the claim is
exercised, not merely never contradicted. -/
#guard stoppedWorkerStartsNothing.answer.coverage == .exercised

end Temporal.Feature.Activity.Standalone.Tests
