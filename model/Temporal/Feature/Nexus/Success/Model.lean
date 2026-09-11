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
`Producer.lean` derives the operation-correlated clauses the Case carries from the `require` lines
below and the Action order the Scenario fixes, so the same-step requirement written here is what the
runtime capability reads.

Read from top to bottom: vocabulary → allowed behavior → requirement → scenario → question.
The five blocks are the intentionally small Nexus success authoring surface. Their elaborator
expands into the existing Umpire Target, Property, Scenario, and Query owners.
-/

namespace Temporal.Feature.Nexus.Success

/-- The operation has three success-lifecycle states. -/
enum State
  | scheduled
  | started
  | succeeded

/-- Waiting recognizes an observed change; it does not cause the operation to change. -/
enum Action
  | awaitStart
  | awaitSuccess

enum Outcome
  | acknowledged
  | completed

model lifecycle
  role: operation
  states: State
  actions: Action
  outcomes: Outcome
  starts: [scheduled]
  ends: [succeeded]
  -- Read `before + action → after` as one permitted model step, not an execution instruction.
  steps:
    scheduled + awaitStart → started, outcome: acknowledged
    started + awaitSuccess → succeeded, outcome: completed

/- `awaitSuccess` must expose the complete Target-owned success result. -/
property successfulResult
  model: lifecycle
  when: awaitSuccess
  require:
    state: succeeded
    outcome: completed

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

