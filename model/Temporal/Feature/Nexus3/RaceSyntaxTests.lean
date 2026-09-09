import Temporal.Feature.Nexus3.Nexus

/-!
# A second lifecycle through the same five commands

This module authors a lifecycle that shares no member spelling, no arity and no role name with the
Nexus3 success slice: four states, three Actions, three Model Outcomes, three Facts, three
transitions and a three-occurrence Behavior. It exists to prove the command surface elaborates
whatever the declaring inductives declare rather than one widened spelling list.
-/

namespace Temporal.Feature.Nexus3.RaceSyntax

open Umpire
open Temporal.Feature.Nexus3

inductive Setup where
  | queued
  deriving BEq, DecidableEq, Repr

inductive State where
  | queued
  | running
  | cancelRequested
  | canceled
  deriving BEq, DecidableEq, Repr

inductive Action where
  | initiate
  | requestCancel
  | resolve
  deriving BEq, DecidableEq, Repr

inductive Outcome where
  | accepted
  | cancellationRequested
  | canceled
  deriving BEq, DecidableEq, Repr

inductive Fact where
  | running
  | cancelRequested
  | settled
  deriving BEq, DecidableEq, Repr

model raceLifecycle
  role handler
  states State
  actions Action
  outcomes Outcome
  facts Fact
  initial [queued]
  terminal [canceled]

  transitions
    begin: queued + initiate →
      { state := running, outcome := accepted, facts := [running] }
    request: running + requestCancel →
      { state := cancelRequested, outcome := cancellationRequested, facts := [cancelRequested] }
    settle: cancelRequested + resolve →
      { state := canceled, outcome := canceled, facts := [settled] }

property cancellationSettles on raceLifecycle
  for handler
  when action resolve
  require settledState: resultingState canceled
  require settledOutcome: outcome canceled
  require settledFact: fact settled

behavior cancellationRace on raceLifecycle handler starts queued
  actions exactly [begin: initiate, request: requestCancel, settle: resolve]

limits raceTrace
  transitions 3
  selected_actions 3
  candidate_evaluations 32

query cancellation on raceLifecycle
  witness cancellationSettles
  in cancellationRace
  limits raceTrace

/- The second lifecycle's derived identities come from its own spellings, and its witness runs the
whole three-step trace. -/
#guard raceLifecycle.stateIds.map (·.value) ==
  ["temporal.nexus3.state.raceLifecycle.queued",
    "temporal.nexus3.state.raceLifecycle.running",
    "temporal.nexus3.state.raceLifecycle.cancelRequested",
    "temporal.nexus3.state.raceLifecycle.canceled"]

#guard raceLifecycle.operationRoleId.value == "temporal.nexus3.role.raceLifecycle.handler"

#guard (do
  let checked ← cancellation.toOption
  pure (checked.witness.trace.steps.map (·.resultingState.value) ==
      ["running", "cancelRequested", "canceled"] &&
    checked.behavior.allowedActions == raceLifecycle.actionIds &&
    checked.property.clauses.length == 3)) == some true

/- Nothing is shared with the success slice: the two Targets are different declarations. -/
#guard raceLifecycle.targetId != lifecycle.targetId

end Temporal.Feature.Nexus3.RaceSyntax
