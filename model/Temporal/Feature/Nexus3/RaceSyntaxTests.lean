import Temporal.Feature.Nexus3.Nexus

/-!
# A second lifecycle through the same five commands

This module authors a lifecycle that shares no member spelling, no arity and no role name with the
Nexus3 success slice: five states, four Actions, four Model Outcomes, three Facts, four transitions,
two terminal states, a losing row that records no Fact, a two-clause Property, and a
three-occurrence Behavior over a four-Action model. It exists to prove the command surface
elaborates whatever the declaring inductives declare rather than one widened spelling list.
-/

namespace Temporal.Feature.Nexus3.RaceSyntax

open Umpire

inductive Setup where
  | queued
  deriving BEq, DecidableEq, Repr

/- The planner admits a Target only when its Action catalog is in canonical order, so each domain
is declared in the order its derived Definition IDs sort. -/
inductive State where
  | cancelRequested
  | canceled
  | completed
  | queued
  | running
  deriving BEq, DecidableEq, Repr

inductive Action where
  | complete
  | initiate
  | requestCancel
  | resolve
  deriving BEq, DecidableEq, Repr

inductive Outcome where
  | accepted
  | canceled
  | cancellationRequested
  | completed
  deriving BEq, DecidableEq, Repr

inductive Fact where
  | cancelRequested
  | running
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

/- The second lifecycle's derived identities come from its own spellings, its witness runs the whole
three-step trace against a two-clause Property, and the Behavior admits only the three Actions it
names while the model declares four. -/
#guard raceLifecycle.stateIds.map (·.value) ==
  ["temporal.nexus3.state.raceLifecycle.cancelRequested",
    "temporal.nexus3.state.raceLifecycle.canceled",
    "temporal.nexus3.state.raceLifecycle.completed",
    "temporal.nexus3.state.raceLifecycle.queued",
    "temporal.nexus3.state.raceLifecycle.running"]

#guard raceLifecycle.operationRoleId.value == "temporal.nexus3.role.raceLifecycle.handler"

#guard raceLifecycle.terminal == [State.canceled, State.completed]

/- The losing row declares no Fact, which is a different shape from every success-slice row. -/
#guard (raceLifecycle.resultsAt 3).map (·.observations) == [[]]

#guard (do
  let checked ← cancellation.toOption
  pure (checked.witness.trace.steps.map (·.resultingState.value) ==
      ["running", "cancelRequested", "canceled"] &&
    checked.behavior.allowedActions ==
      [raceLifecycle.actionIdAt 1, raceLifecycle.actionIdAt 2, raceLifecycle.actionIdAt 3] &&
    checked.property.clauses.length == 2)) == some true

/- Nothing is shared with the success slice: the two Targets are different declarations. -/
#guard raceLifecycle.targetId != lifecycle.targetId

end Temporal.Feature.Nexus3.RaceSyntax
