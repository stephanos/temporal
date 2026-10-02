import Temporal.Feature.Nexus.Pair.Model

/-!
# The negative control

A Model whose one claim the platform contradicts, on purpose. It is the caller Model's `operation`
entity and action classes over a machine that keeps the real rows of every reply and adds one row
the platform never takes: a non-retryable handler error completing the operation as succeeded,
with the completed event recording it. Its one Query selects that row and its Property names it, so
the Case the caller realization produces expects a completed event where the platform writes a
failed one, and every Run of it is violated, deterministically and honestly.

That is what a replay needs before any reducer exists: one Case whose Run is violated and can be
recorded, admitted, replayed offline and rerun. The control enters no functional, canary or
exploratory set of the caller Model and no regression view; its Case is registered under
`temporal.case.nexusCallerControl.<query>` so that `umpire-case` renders its fixture and the live
suite runs it, and nothing reads it as a claim about the platform. The proposal a replay compiles
from it proves the promotion mechanism only: its expected trace is the row the platform never
takes, and it is never reviewed into a regression set.

The forged row is a second result of a row the platform has, not a replaced one: the Case Runtime
admits a recorded step only as an authorized row, and a rule is violated only when the observed
step is such a row whose response fails. The real row stays authorized, and since the Producer
declares the evidence of every result of a witnessed row, the failed event the platform writes is
lifted and projected to the real row rather than left unread.
-/

namespace Temporal.Feature.Nexus.Control

open Umpire
open Umpire.Command
open Temporal.Feature.Nexus.Caller
open Temporal.Feature.Nexus.Pair

-- authoring: derived

/-! ### The machine

The pair Model's machine -- the operation without its deadlines and retries -- plus the forged row.
The machine is derived from it rather than copied: every row is the pair machine's, and the one
function written here returns the one result the platform never takes. -/

/-- The forged row: the non-retryable error completes the operation and records the completed
event. The platform's row for that error, which fails the operation and records the failed event,
is the pair machine's and stays beside it. -/
def controlForgedStep (state : PairState) (reply : Reply) :
    List (Step PairState PairOutcome PairFact) :=
  if state.phase != .scheduled then [] else
  match reply with
  | .handlerError false =>
      [{ outcome := .accepted, state := { phase := .succeeded },
         facts := [.nexusOperationCompleted] }]
  | _ => []

machine nexusControl
  from: pair
  extend:
    handlerReply: controlForgedStep

-- authoring: end

/-! ### The claim the platform contradicts -/

/- A non-retryable handler error completes the operation, and the completed event records it. The
platform fails it instead, so a Run of a Case realizing this claim is violated. -/
property forgedSuccess
  machine: nexusControl
  when: handlerReply (handlerError false)
  holds: fun step =>
    step.state.phase == .succeeded && step.facts.contains .nexusOperationCompleted

/-! ### The path, the Query, the set and the Case -/

scenario nonRetryableErrorForged
  model: nexusControl
  starts: unscheduled
  actions: [schedule (unset, unset, unset), handlerReply (handlerError false)]

limits controlTwo
  steps: 2
  actions: 2
  search: 16

query forgedCompletion
  find: forgedSuccess
  in: nonRetryableErrorForged
  limits: controlTwo

set nexusCallerControl
  purpose: functional
  bind:
    caller: driven
    handler: driven
  queries: [forgedCompletion]

/- The caller realization serves the control: the schedule and the handler's error reply are the
classes it binds, and the Producer declares the failed event's evidence beside the completed one's
for the witnessed row. -/
case nexusCallerControlCases
  realizes nexusCallerControl
  as nexusCallerCases.realization

end Temporal.Feature.Nexus.Control
