import Temporal.Case.Syntax
import Temporal.Feature.Nexus.Race.Terminal

/-!
# Compact Nexus success model

This executable slice models only `scheduled → started → succeeded`. `awaitStart` and
`awaitSuccess` wait for recorded Temporal outcomes; they do not manufacture those outcomes. It
records no Fact: every step here reaches a state named after what happened, so a Fact would only
restate it. A Model declares Facts where one carries a claim its state does not -- two paths into
the same state, something that happened without a state change, or several claims in one step.
Cancellation remains an unsupported design sketch in `Nexus.md`; the imported `Terminal` module
is the historical already-started Target described in `Integration.md`, not this slice's.
This slice authors no correlated Property of its own. It does not need one: the Producer in
`Producer.lean` derives the operation-correlated clauses the Case carries from the values the
`holds:` predicate below fixes and the Action order the Scenario fixes, so the same-step requirement
written here is what the runtime capability reads.

Read from top to bottom: vocabulary → allowed behavior → requirement → scenario → question.
The five blocks are the intentionally small Nexus success authoring surface. Their elaborator
expands into the existing Umpire Target, Property, Scenario, and Query owners.
-/

namespace Temporal.Feature.Nexus.Success

/-- The operation this slice tracks. A machine tracks one entity's instances, and recorded data
finds one through the entity's key. -/
entity operation

/-- The operation has three success-lifecycle states. -/
enum State
  | scheduled
  | started
  | succeeded

/-- A machine's states are the members of a structure, so its fields can be enumerated. This one has
a single field, so a state key is the phase the author writes and nothing else. -/
structure Lifecycle where
  state : State
  deriving BEq, DecidableEq, Repr, Umpire.Command.Finite

enum Outcome
  | acknowledged
  | completed

/-- This slice records no Fact: every step reaches a state named after what happened, so a Fact would
only restate it. An empty domain is how a machine says that -- the step functions return into it,
and the declared Model's fact catalog is empty. -/
inductive Fact where
  deriving BEq, DecidableEq, Repr, Umpire.Command.Finite

/-- Waiting recognizes an observed change; it does not cause the operation to change. -/
action awaitStart
  party: caller
  on: operation

action awaitSuccess
  party: caller
  on: operation

/-- Read a step function as what may happen, not as an execution instruction: it returns every
successor the Model permits from this state, and the empty list where the Action is not permitted at
all. -/
def awaitStartStep (current : Lifecycle) : List (Umpire.Step Lifecycle Outcome Fact) :=
  if current.state != .scheduled then [] else
  [{ outcome := .acknowledged, state := { state := .started }, facts := [] }]

def awaitSuccessStep (current : Lifecycle) : List (Umpire.Step Lifecycle Outcome Fact) :=
  if current.state != .started then [] else
  [{ outcome := .completed, state := { state := .succeeded }, facts := [] }]

machine lifecycle
  for: operation
  state: Lifecycle
  starts: [scheduled]
  ends: [succeeded]
  steps:
    awaitStart: awaitStartStep
    awaitSuccess: awaitSuccessStep

/- `awaitSuccess` must expose the complete Target-owned success result. The predicate is ordinary
Lean over the step the Action produces; the command enumerates it over the machine's table into the
state and outcome it fixes, which is what Search and the Case read. -/
property successfulResult
  machine: lifecycle
  when: awaitSuccess
  holds: fun step => step.state.state == .succeeded && step.outcome == .completed

/- `actions:` is the exact sequence the operation selects, and its length. -/
scenario successfulCompletion
  model: lifecycle
  starts: scheduled
  actions: [awaitStart, awaitSuccess]

/-
The three limits bound different quantities. Steps count model steps, actions count occurrences,
and search bounds the work the planner may spend. The start state is not a step.
-/
limits shortTrace
  steps: 2
  actions: 2
  search: 16

/- Finding this trace establishes possibility within the declared bounds.

The two Known Gaps below limit what any Case realizing this Query can prove. Both name a Property
this slice cannot declare, which is the gap: a cancellation requirement has no operation-correlated
shape here, so there is nothing to write a `require` line about. -/
query completion
  find: successfulResult
  in: successfulCompletion
  limits: shortTrace
  gap: capability
    code: "cancellation"
    subject: "cancellationResolves"
    detail: "Operation-correlated Nexus cancellation is unsupported by the success slice."
  gap: capability
    code: "operation-correlated-progress"
    subject: "cancellationResolves"
    detail: "Operation-correlated progress counting is unsupported by the success slice."

/-
The Case the selected trace realizes. `fixture` is the only identity slot: the Case ID is
`temporal.case.async-nexus`, the Program and Contract IDs derive from it, and the Run scope is the
fixture name. Each `evidence` line says which recorded history event confirms one Action; the Step
it confirms is read from the `steps` block above, along the witness trace.
-/
case asyncNexusSuccess fixture "async-nexus"
  realizes completion
  as nexusOperation service "umpire.case.service" operation "complete" responds async
  evidence
    awaitStart ← history nexusOperationStarted
    awaitSuccess ← history nexusOperationCompleted

end Temporal.Feature.Nexus.Success

