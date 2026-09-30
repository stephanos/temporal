import Umpire.Command

/-!
Mutation helpers for tests of the Model commands. Every one of them takes a checked declaration
apart and puts it back differently, which is how a test asks what the admission layer rejects. They
are deliberately outside the production module: nothing a command emits calls them.
-/

namespace Umpire.Command.Tests

open Umpire
open Umpire.Command

section Mutations

-- Bound here rather than auto-bound, because `Umpire.Command` declares an `Action` of its own.
variable {Setup State Action Outcome Fact : Type}

def withStates
    (table : FiniteTable Setup State Action Outcome Fact)
    (states : List (FiniteCatalogEntry State)) : FiniteTable Setup State Action Outcome Fact :=
  { table with states }

def withTransitions
    (table : FiniteTable Setup State Action Outcome Fact)
    (transitions : List (FiniteTransitionRow State Action Outcome Fact)) :
    FiniteTable Setup State Action Outcome Fact :=
  { table with transitions }

def transitionRow
    (key : String)
    (source : State)
    (selectedAction : Action)
    (results : List (Step State Outcome Fact)) :
    FiniteTransitionRow State Action Outcome Fact :=
  { key, source, action := selectedAction, results }

def withOccurrences (spec : Scenario) (occurrences : List Scenario.Step) : Scenario :=
  spec.withSteps occurrences

def withClauses (spec : Property) (clauses : List PropertyClause) : Property :=
  { spec with clauses }

def reorderedAndDocumented (spec : Property) (documentation : String) : Property :=
  { spec with clauses := spec.clauses.reverse, documentation }

/-- One Scenario occurrence, addressed through the declaring file's own family. -/
def occurrence (origin : Origin) (key : String) (selectedAction : DefinitionId) : Scenario.Step :=
  { id := origin.family.id "occurrence" key, action := selectedAction }

end Mutations

/-! ### A claim on one field of a machine's state

A lever is up or down and worn or not; a pull takes an up lever down and leaves its wear as it was.
A claim that a pull leaves the lever down fixes the phase while the wear varies, and one that names
both fields fixes the whole state, as it always has. Neither a product of instances nor a
refinement exposes the lever's own fields, so each refuses the field claim. -/

namespace Lever

model_conventions root "umpire" under Umpire.Command.Tests.Lever

entity lever

enum LeverPhase
  | up
  | down

structure LeverState where
  phase : LeverPhase
  worn : Bool
  deriving BEq, DecidableEq, Repr, Finite

enum LeverOutcome
  | accepted

inductive LeverFact
  deriving BEq, DecidableEq, Repr, Finite

action pull
  party: agent
  on: lever

action lift
  party: agent
  on: lever

def pullStep (state : LeverState) : List (Step LeverState LeverOutcome LeverFact) :=
  if state.phase != .up then [] else
  [{ outcome := .accepted, state := { state with phase := .down }, facts := [] }]

def liftStep (state : LeverState) : List (Step LeverState LeverOutcome LeverFact) :=
  if state.phase != .down then [] else
  [{ outcome := .accepted, state := { phase := .up, worn := true }, facts := [] }]

machine leverMachine
  for: lever
  state: LeverState
  starts: [up]
  ends: [up, down]
  steps:
    pull: pullStep
    lift: liftStep

property pulledDown
  machine: leverMachine
  when: pull
  holds: fun step => step.state.phase == .down

#guard pulledDown.names.groups.map (fun group => (group.trigger, group.requirements)) ==
  [(.action "pull", [.stateFieldClause "field-phase-down" "phase" "down"])]

/- Both fields named: one whole state, and no field clause. -/
property pulledFresh
  machine: leverMachine
  when: pull
  holds: fun step => step.state.phase == .down && !step.state.worn

#guard pulledFresh.names.groups.map (·.requirements) == [[.stateClause "state-down-false" "down-false"]]

limits two
  steps: 2
  actions: 2
  search: 16

scenario twoPulls
  model: leverMachine
  instances: 2
  starts: up
  actions: [pull 1, pull 2]

/--
error: the Property fixes the state field 'phase', and a Query over several instances reads each instance's whole state as one field of the product, so there is no field 'phase' to address; a Scenario over several instances names a Property that fixes a whole state
-/
#guard_msgs in
query pulledTwice
  verify: pulledDown
  in: twoPulls
  limits: two

structure CrankState where
  phase : LeverPhase
  worn : Bool
  greased : Bool
  deriving BEq, DecidableEq, Repr, Finite

def crankPullStep (state : CrankState) : List (Step CrankState LeverOutcome LeverFact) :=
  if state.phase != .up then [] else
  [{ outcome := .accepted, state := { state with phase := .down }, facts := [] }]

def crankLiftStep (state : CrankState) : List (Step CrankState LeverOutcome LeverFact) :=
  if state.phase != .down then [] else
  [{ outcome := .accepted, state := { state with phase := .up, worn := true }, facts := [] }]

/-- The lever a crank is: its grease is hidden. -/
def leverOf (state : CrankState) : LeverState := { phase := state.phase, worn := state.worn }

machine crankMachine
  for: lever
  state: CrankState
  refines: leverMachine
  map: leverOf
  starts: [up]
  ends: [up, down]
  steps:
    pull: crankPullStep
    lift: crankLiftStep

/- A claim on the crank's own phase fixes that field: the lever state it reads as moves with the
phase, and is not a field the claim holds. -/
property crankPulledDown
  machine: crankMachine
  when: pull
  holds: fun step => step.state.phase == .down

#guard crankPulledDown.names.groups.map (·.requirements) ==
  [[.stateFieldClause "field-phase-down" "phase" "down"]]

scenario crankPull
  model: crankMachine
  starts: up
  actions: [pull]

/--
error: the Property fixes the state field 'phase' of 'Umpire.Command.Tests.Lever.leverMachine', and 'Umpire.Command.Tests.Lever.crankMachine' reads each of its states as a whole state of 'Umpire.Command.Tests.Lever.leverMachine' through its `map:`, so it holds no field 'phase' to address; a Property read through a refinement fixes a whole state
-/
#guard_msgs in
query crankPulled
  verify: pulledDown
  in: crankPull
  limits: two

end Lever

end Umpire.Command.Tests
