import Temporal.Feature.Nexus.Success.Model

/-!
# A second lifecycle through the same five commands

This module authors a lifecycle that shares no state, Action or Fact spelling, no arity and no role
name with the Nexus success slice: five states, four Actions, four Model Outcomes, three Facts,
four transitions, two terminal states, a losing row that records no Fact, a two-clause Property,
and a three-occurrence Behavior over a four-Action machine. It exists to prove the command surface
elaborates whatever the declaring inductives declare rather than one widened spelling list.
-/

namespace Temporal.Feature.Nexus.Success.RaceSyntax

open Umpire

/-- The entity this lifecycle tracks. It is named `handler` because the role name a Model carries is
the entity's, and this module exists to share no spelling with the success slice. -/
entity handler

/- The domains are declared in lifecycle order. A machine emits its Action catalog in canonical
order whatever order the `steps:` lines are written in, so declaration order is the author's to read
and nothing else's. -/
enum State
  | queued
  | running
  | cancelRequested
  | canceled
  | completed

/-- A machine's states are the members of a structure. This one has a single field, so a state key
is the phase the author writes and nothing else. -/
structure Phase where
  phase : State
  deriving BEq, DecidableEq, Repr, Umpire.Command.Finite

inductive Outcome where
  | accepted
  | cancellationRequested
  | canceled
  | completed
  deriving BEq, DecidableEq, Repr, Umpire.Command.Finite

inductive Fact where
  | running
  | cancelRequested
  | settled
  deriving BEq, DecidableEq, Repr, Umpire.Command.Finite

action complete
  party: worker
  on: handler

action initiate
  party: caller
  on: handler

action requestCancel
  party: caller
  on: handler

action resolve
  party: worker
  on: handler

private def moves (phase : State) (outcome : Outcome) (recorded : List Fact) :
    List (Umpire.Step Phase Outcome Fact) :=
  [{ outcome, state := { phase }, facts := recorded }]

def initiateStep (current : Phase) : List (Umpire.Step Phase Outcome Fact) :=
  if current.phase != .queued then [] else moves .running .accepted [.running]

def requestCancelStep (current : Phase) : List (Umpire.Step Phase Outcome Fact) :=
  if current.phase != .running then [] else
  moves .cancelRequested .cancellationRequested [.cancelRequested]

def resolveStep (current : Phase) : List (Umpire.Step Phase Outcome Fact) :=
  if current.phase != .cancelRequested then [] else moves .canceled .canceled [.settled]

/-- A cancellation request can lose: both `canceled` and `completed` are terminal, and the losing
step records no Fact at all. -/
def completeStep (current : Phase) : List (Umpire.Step Phase Outcome Fact) :=
  if current.phase != .cancelRequested then [] else moves .completed .completed []

machine raceLifecycle
  for: handler
  state: Phase
  starts: [queued]
  ends: [canceled, completed]
  steps:
    initiate: initiateStep
    requestCancel: requestCancelStep
    resolve: resolveStep
    complete: completeStep

/- Two `require` clauses, not the success slice's three. -/
property cancellationSettles
  model: raceLifecycle
  when: resolve
  require:
    state: canceled
    fact: settled

scenario cancellationRace
  model: raceLifecycle
  starts: queued
  actions: [initiate, requestCancel, resolve]

limits raceTrace
  steps: 4
  actions: 3
  search: 32

query cancellation
  find: cancellationSettles
  in: cancellationRace
  limits: raceTrace

/- The same second lifecycle also carries the verify form, which claims the requirement over every
trace the Behavior admits instead of selecting one. -/
query cancellationVerified
  verify: cancellationSettles
  in: cancellationRace
  limits: raceTrace

#guard (do
  let checked ← cancellationVerified.toOption
  pure (checked.witness.isNone && checked.query.form.name == "verify")) == some true

/- The second lifecycle's derived identities come from its own spellings, its witness runs the whole
three-step trace against a two-clause Property, and the Behavior admits only the three Actions it
names while the model declares four. Admission canonicalizes clause order, so the checked clauses
read in sorted-ID order rather than declaration order. -/
#guard raceLifecycle.stateIds.map (·.value) ==
  ["temporal.nexus.success.raceSyntax.state.raceLifecycle.queued",
    "temporal.nexus.success.raceSyntax.state.raceLifecycle.running",
    "temporal.nexus.success.raceSyntax.state.raceLifecycle.cancelRequested",
    "temporal.nexus.success.raceSyntax.state.raceLifecycle.canceled",
    "temporal.nexus.success.raceSyntax.state.raceLifecycle.completed"]

#guard raceLifecycle.operationRoleId.value == "temporal.nexus.success.raceSyntax.role.raceLifecycle.handler"

#guard raceLifecycle.terminal == [{ phase := .canceled }, { phase := .completed }]

/- The losing row declares no Fact, which is a different shape from every success-slice row. The
rows are enumerated states-major over the canonical Action order, so the losing row is the first of
the two out of `cancelRequested` rather than the last row written. -/
#guard (raceLifecycle.resultsAt 2).map (·.facts) == [[]]
#guard raceLifecycle.transitions.map (·.key) ==
  ["queued-initiate", "running-requestCancel", "cancelRequested-complete",
    "cancelRequested-resolve"]

#guard (do
  let checked ← cancellation.toOption
  let selected ← checked.witness
  pure (selected.trace.steps.map (·.state.value) ==
      ["running", "cancelRequested", "canceled"] &&
    checked.behavior.allowedActions ==
      [raceLifecycle.actionIdAt 1, raceLifecycle.actionIdAt 2, raceLifecycle.actionIdAt 3] &&
    checked.property.clauses.map (·.id.value) ==
      ["temporal.nexus.success.raceSyntax.property.cancellationSettles.fact-settled",
        "temporal.nexus.success.raceSyntax.property.cancellationSettles.state-canceled"])) == some true

/- Nothing is shared with the success slice: the two Targets are different declarations. -/
#guard raceLifecycle.targetId != lifecycle.targetId

end Temporal.Feature.Nexus.Success.RaceSyntax
