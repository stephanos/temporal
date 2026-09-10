import Temporal.Feature.Nexus.Success.Syntax
import Temporal.Feature.Nexus.Race.Terminal

/-!
# Compact Nexus success model

This executable slice models only `scheduled → started → succeeded`. `awaitStart` and
`awaitSuccess` wait for recorded Temporal outcomes; they do not manufacture those outcomes.
Cancellation remains an unsupported design sketch in `Nexus.md`; the imported `Terminal` module
is the historical already-started Target described in `Integration.md`, not this slice's.
This slice authors no correlated Property of its own. It does not need one: the Producer in
`Producer.lean` derives the operation-correlated clauses the Case carries from the `require` lines
below and the Action order the Scenario fixes, so the same-step requirement written here is what the
runtime capability reads.

Read from top to bottom: vocabulary → allowed behavior → requirement → scenario → question.
The five blocks are the intentionally small Nexus success authoring surface. Their elaborator
expands into the existing Umpire Target, Property, Scenario, and Query owners.
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
  starts [scheduled]
  ends [succeeded]

  -- Read `before + action → result` as one permitted model step, not an execution instruction.
  steps
    start: scheduled + awaitStart →
      { state := started, outcome := acknowledged, facts := [started] }
    success: started + awaitSuccess →
      { state := succeeded, outcome := completed, facts := [succeeded] }

/- `awaitSuccess` must expose the complete Target-owned success result. -/
property successfulResult on lifecycle
  for operation
  when awaitSuccess
  require successState: state succeeded
  require successOutcome: outcome completed
  require successFact: fact succeeded

/- `exactly` fixes both the selected Action sequence and its length. -/
scenario successfulCompletion on lifecycle
  operation starts scheduled
  actions exactly [start: awaitStart, completion: awaitSuccess]

/-
The three limits bound different quantities. Steps count model steps, actions count occurrences,
and search bounds the work the planner may spend. The start state is not a step.
-/
limits shortTrace
  steps 2
  actions 2
  search 16

/- Finding this trace establishes possibility within the declared bounds. -/
query completion on lifecycle
  find successfulResult
  in successfulCompletion
  limits shortTrace

end Temporal.Feature.Nexus.Success

