import Temporal.Feature.Nexus.Success.Syntax
import Temporal.Feature.Nexus.Race.Terminal

/-!
# Compact Nexus success model

This executable slice models only `scheduled → started → succeeded`. `awaitStart` and
`awaitSuccess` wait for recorded Temporal outcomes; they do not manufacture those outcomes.
Cancellation remains an unsupported design sketch in `Nexus.md`; the imported `Cancellation` module
is the historical already-started Target described in `Integration.md`, not this slice's.
This slice authors no correlated Property of its own. It does not need one: the Producer in
`Testpilot.lean` derives the operation-correlated clauses the Case carries from the `require` lines
below and the Action order the Behavior fixes, so the same-step requirement written here is what the
runtime capability reads.

Read from top to bottom: vocabulary → allowed behavior → requirement → scenario → question.
The five blocks are the intentionally small Nexus.Success success authoring surface. Their elaborator
expands into the existing Umpire Target, Property, Behavior, and Query owners.
-/

namespace Temporal.Feature.Nexus.Success

/-- The operation has one scheduled setup and three success-lifecycle states. -/
inductive Setup where
  | scheduled
  deriving BEq, DecidableEq, Repr

inductive State where
  | scheduled
  | started
  | succeeded
  deriving BEq, DecidableEq, Repr

/-- Waiting recognizes an observed change; it does not cause the operation to change. -/
inductive Action where
  | awaitStart
  | awaitSuccess
  deriving BEq, DecidableEq, Repr

inductive Outcome where
  | acknowledged
  | completed
  deriving BEq, DecidableEq, Repr

/-- Facts are model claims. Runtime evidence must establish them through later integration. -/
inductive Fact where
  | started
  | succeeded
  deriving BEq, DecidableEq, Repr

model lifecycle
  role operation
  states State
  actions Action
  outcomes Outcome
  facts Fact
  initial [scheduled]
  terminal [succeeded]

  -- Read `before + action → result` as one permitted model step, not an execution instruction.
  transitions
    start: scheduled + awaitStart →
      { state := started, outcome := acknowledged, facts := [started] }
    success: started + awaitSuccess →
      { state := succeeded, outcome := completed, facts := [succeeded] }

/- `awaitSuccess` must expose the complete Target-owned success result. -/
property successfulResult on lifecycle
  for operation
  when action awaitSuccess
  require successState: resultingState succeeded
  require successOutcome: outcome completed
  require successFact: fact succeeded

/- `exactly` fixes both the selected Action sequence and its length. -/
behavior successfulCompletion on lifecycle
  operation starts scheduled
  actions exactly [start: awaitStart, completion: awaitSuccess]

/-
The three limits bound different quantities. Transitions count model steps, selected Actions count
occurrences, and candidate evaluations bound search work. The initial state is not a transition.
-/
limits shortTrace
  transitions 2
  selected_actions 2
  candidate_evaluations 16

/- Finding this witness establishes possibility within the declared bounds. -/
query completion on lifecycle
  witness successfulResult
  in successfulCompletion
  limits shortTrace

end Temporal.Feature.Nexus.Success

