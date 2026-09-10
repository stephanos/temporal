import Temporal.Feature.Nexus3.Nexus

/-!
# A second lifecycle through the same five commands

This module authors a lifecycle that shares no state, Action or Fact spelling, no arity and no role
name with the Nexus3 success slice: five states, four Actions, four Model Outcomes, three Facts,
four transitions, two terminal states, a losing row that records no Fact, a two-clause Property,
and a three-occurrence Behavior over a four-Action model. It exists to prove the command surface
elaborates whatever the declaring inductives declare rather than one widened spelling list.
-/

namespace Temporal.Feature.Nexus3.RaceSyntax

open Umpire

inductive Setup where
  | queued
  deriving BEq, DecidableEq, Repr

/- Only the Action catalog is order-checked, and the `model` command rejects an unsorted one in
place. The other domains are declared in lifecycle order to keep that distinction visible. -/
inductive State where
  | queued
  | running
  | cancelRequested
  | canceled
  | completed
  deriving BEq, DecidableEq, Repr

inductive Action where
  | complete
  | initiate
  | requestCancel
  | resolve
  deriving BEq, DecidableEq, Repr

inductive Outcome where
  | accepted
  | cancellationRequested
  | canceled
  | completed
  deriving BEq, DecidableEq, Repr

inductive Fact where
  | running
  | cancelRequested
  | settled
  deriving BEq, DecidableEq, Repr

/- A cancellation request can lose: both `canceled` and `completed` are terminal, and the losing
row records no Fact at all. -/
model raceLifecycle
  role handler
  states State
  actions Action
  outcomes Outcome
  facts Fact
  initial [queued]
  terminal [canceled, completed]

  transitions
    begin: queued + initiate →
      { state := running, outcome := accepted, facts := [running] }
    request: running + requestCancel →
      { state := cancelRequested, outcome := cancellationRequested, facts := [cancelRequested] }
    settle: cancelRequested + resolve →
      { state := canceled, outcome := canceled, facts := [settled] }
    lose: cancelRequested + complete →
      { state := completed, outcome := completed, facts := [] }

/- Two `require` clauses, not the success slice's three. -/
property cancellationSettles on raceLifecycle
  for handler
  when action resolve
  require settledState: resultingState canceled
  require settledFact: fact settled

behavior cancellationRace on raceLifecycle handler starts queued
  actions exactly [begin: initiate, request: requestCancel, settle: resolve]

limits raceTrace
  transitions 4
  selected_actions 3
  candidate_evaluations 32

query cancellation on raceLifecycle
  witness cancellationSettles
  in cancellationRace
  limits raceTrace

/- The same second lifecycle also carries the verify form, which claims the requirement over every
trace the Behavior admits instead of selecting one. -/
query cancellationVerified on raceLifecycle
  all cancellationSettles
  in cancellationRace
  limits raceTrace

#guard (do
  let checked ← cancellationVerified.toOption
  pure (checked.witness.isNone && checked.query.form.name == "verify")) == some true

/- The second lifecycle's derived identities come from its own spellings, its witness runs the whole
three-step trace against a two-clause Property, and the Behavior admits only the three Actions it
names while the model declares four. Admission canonicalizes clause order, so the checked clauses
read in sorted-ID order rather than declaration order. -/
#guard raceLifecycle.stateIds.map (·.value) ==
  ["temporal.nexus3.state.raceLifecycle.queued",
    "temporal.nexus3.state.raceLifecycle.running",
    "temporal.nexus3.state.raceLifecycle.cancelRequested",
    "temporal.nexus3.state.raceLifecycle.canceled",
    "temporal.nexus3.state.raceLifecycle.completed"]

#guard raceLifecycle.operationRoleId.value == "temporal.nexus3.role.raceLifecycle.handler"

#guard raceLifecycle.terminal == [State.canceled, State.completed]

/- The losing row declares no Fact, which is a different shape from every success-slice row. -/
#guard (raceLifecycle.resultsAt 3).map (·.facts) == [[]]

#guard (do
  let checked ← cancellation.toOption
  let selected ← checked.witness
  pure (selected.trace.steps.map (·.state.value) ==
      ["running", "cancelRequested", "canceled"] &&
    checked.behavior.allowedActions ==
      [raceLifecycle.actionIdAt 1, raceLifecycle.actionIdAt 2, raceLifecycle.actionIdAt 3] &&
    checked.property.clauses.map (·.id.value) ==
      ["temporal.nexus3.property.cancellationSettles.settledFact",
        "temporal.nexus3.property.cancellationSettles.settledState"])) == some true

/- Nothing is shared with the success slice: the two Targets are different declarations. -/
#guard raceLifecycle.targetId != lifecycle.targetId

end Temporal.Feature.Nexus3.RaceSyntax
