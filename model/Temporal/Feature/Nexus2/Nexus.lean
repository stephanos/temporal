/-!
# Nexus — user-facing authoring draft

Proposed syntax for discussion; this file does not compile and is not imported by the model.
The declarations below describe the desired authoring surface, not an implemented Umpire API.

Scope: one operation, asynchronous start, successful completion, and cancellation that can lose
to completion. Cancellation requests are distinct from cancellation outcomes. This deliberately
refines the old baseline's immediate cancellation into a request followed by resolution.

Retries, failures, timeouts, caller closure, repeated requests, and late events are outside this
first draft. Model transitions are not RPCs or evidence that Temporal performed an operation.

## How to read this file

Read from top to bottom: vocabulary → allowed behavior → requirements → scenarios → questions.

* A State describes where the operation is now.
* An Action describes an event that can change that state.
* The model defines which transitions and outcomes are possible.
* A Property states a requirement to check against those possibilities.
* A Behavior restricts the traces considered for one scenario.
* A Query asks for an example or checks a requirement within explicit limits.

A trace is an initial state followed by state-changing steps, for example:
`scheduled → started → succeeded`. Actions explain why each arrow occurs.
Properties do not repair or filter the model's transitions to make requirements pass.

## Lean syntax versus proposed syntax

`namespace`, `inductive`, `where`, and the constructor bars `|` are ordinary Lean syntax.
The `model`, `property`, `behavior`, `limits`, and `query` blocks are proposed authoring syntax.
Their comments describe intended meaning; no parser, checker, or Case compiler for this spelling
exists yet. In particular, `+` and `→` in a transition row are visual separators here, not Lean
addition or a function type. `/-- ... -/` introduces documentation; `/- ... -/` is a block comment.

Executing this description against Temporal would require lowering it into a Case Program and
Contract, including declared runtime Observations. This draft has no such connection yet.
-/

-- A namespace groups related names: outside it, `State` would be referred to as `Nexus.State`.
namespace Nexus

/-- An `inductive` declaration introduces a type with exactly the listed alternatives.
This model follows one operation, so a single state value is enough; it has no IDs or collection
of concurrent operations. `cancelRequested` is nonterminal: asking to cancel is not confirmation
that cancellation happened. -/
inductive State where
  /-- The operation is scheduled, but the handler has not yet acknowledged asynchronous work. -/
  | scheduled
  /-- The handler has acknowledged the operation and may complete it later. -/
  | started
  /-- Cancellation was requested; the operation's final result is still undecided. -/
  | cancelRequested
  /-- Terminal result: the operation settled as canceled. -/
  | canceled
  /-- Terminal result: the operation completed successfully. -/
  | succeeded

/-- Actions include both requests and modeled environment events. They are not a list of RPCs
the test controller can invoke. Separating an Action from its result lets one Action admit
multiple outcomes without allowing a test to choose which outcome the system produces. -/
inductive Action where
  /-- Represents asynchronous acknowledgment, rather than initial scheduling. -/
  | start
  /-- Represents successful handler completion before any cancellation request in this model. -/
  | reportSuccess
  /-- Asks for cancellation without promising which terminal result will follow. -/
  | requestCancel
  /-- Abstracts resolution of the cancellation/completion race into one progress step. -/
  | resolve

/-
`on lifecycle` below refers to this model. Its explicit `id` is intended to provide a stable
identity independent of the Lean declaration name; automatic identity derivation for the other
declarations is still an authoring-interface decision.

The state and Action declarations supply the complete vocabulary, including terminal states
with no outgoing rows. `initial` requires every scenario to begin at `scheduled`.
Reaching `started` requires an explicit `start` step, including in cancellation scenarios.

`terminal` identifies final results. A future checker must check consistency with the transition
table rather than silently remove a row that contradicts this declaration.
-/
model lifecycle
  id "temporal.nexus-draft.lifecycle"
  states State
  actions Action
  initial [scheduled]
  terminal [canceled, succeeded]

  -- Read `before + action → after` as one permitted step, not an instruction to execute it.
  -- `oneOf` lists alternatives, not priorities or a random distribution; both must be considered.
  transitions
    scheduled       + start         → started
    started         + reportSuccess → succeeded
    started         + requestCancel → cancelRequested
    cancelRequested + resolve       → oneOf [canceled, succeeded]

/-
Each row lists all allowed results; absent pairs have no transition. In this draft each result
reports its destination state as its Model Outcome. `resolve` represents environment progress:
the model chooses among its alternatives, and a scenario cannot force cancellation to win.

For example, there is no `scheduled + reportSuccess` row. This omits synchronous completion
from the draft; it does not claim that real Nexus operations cannot complete synchronously.
Likewise, the absence of late-event rows says nothing about whether a real server ignores or
rejects a late event. Those behaviors need explicit modeling before they can be checked.
-/

/-- A safety requirement: every cancellation-request step must leave the operation pending
cancellation. `when` selects the triggering step, and `resultingState` reads that same step's
output. A transition directly to `canceled` would violate this particular request/response model.
A trace with no request satisfies this conditional requirement without exercising it. -/
property cancellationIsARequest on lifecycle
  when action requestCancel
  require resultingState cancelRequested

/-- A bounded progress requirement: a request must be followed by either terminal result within
one additional model transition. The intended existing temporal semantics also allow a response
on the triggering step; our request row does not produce one, so resolution needs the next step.
This is a bound in model steps, not seconds. The Behavior below explicitly includes progress.
A trace ending immediately after the request cannot demonstrate the required response. -/
property cancellationResolves on lifecycle
  when action requestCancel
  require eventually state oneOf [canceled, succeeded]
    within 1 semantic_transition

/-- A structural requirement on the transition relation: terminal states have no outgoing rows.
This proposed spelling needs a defined checking path; none of the Queries below checks this
Property. Merely writing a Property is not evidence that verification has run or passed. -/
property terminalIsFinal on lifecycle
  require no transition from [canceled, succeeded]

/-- A Behavior selects model traces, not runtime instructions. `exactly` fixes the selected
Action sequence and its length. In `completion: reportSuccess`, the name before `:` labels this
occurrence, while the name after `:` identifies the Action. Labels distinguish occurrences if
the same Action is later allowed more than once in a scenario. -/
behavior successfulCompletion on lifecycle
  starts scheduled
  actions exactly [start: start, completion: reportSuccess]

/-- This scenario admits two traces:
`scheduled → started → cancelRequested → canceled` and
`scheduled → started → cancelRequested → succeeded`.
The requests are the same in both traces; the model supplies the resolution alternatives.
Completion before the request and a request with no later resolution are outside this Behavior.
Claims checked only here therefore do not cover every possible cancellation interleaving. -/
behavior cancellationRace on lifecycle
  starts scheduled
  actions exactly [start: start, request: requestCancel, resolution: resolve]

/-
Queries share these explicit model-search bounds. Exhaustive verification must report
limit-reached if the budget is insufficient. The cancellation scenario includes resolution;
its progress property does not claim fairness or promise a wall-clock completion deadline.

The three limits bound different quantities. Transitions bound trace steps; selected Actions
bound Action occurrences; candidate evaluations bound search work. They coincide in some small
examples, but are not interchangeable. The initial state is not itself a transition.
The cancellation scenario needs three steps; successful completion still needs only two.
The search budget of 32 is a proposed budget, not a measured guarantee of exhaustive coverage.
-/

limits shortTrace
  transitions 3
  selected_actions 3
  candidate_evaluations 32

/-- A witness Query asks whether at least one admitted trace ends successfully. Its expected
witness is `scheduled → started → succeeded`. Finding one establishes possibility, not that
every trace succeeds. `witness finalState ...` is proposed shorthand whose lowering remains
to be designed; this draft does not add a second Property evaluator. -/
query completion on lifecycle
  witness finalState succeeded
  in successfulCompletion
  limits shortTrace

/-- Verification asks whether every trace admitted by this scenario satisfies the Property.
Both race outcomes should satisfy request safety because the request step leaves both traces
in `cancelRequested`.
An exhaustive claim is valid only if the search actually covers the entire bounded space.
An impossible scenario must report unsatisfiable, not a passing verification result. -/
query cancellationSafety on lifecycle
  verify cancellationIsARequest
  in cancellationRace
  limits shortTrace
  search exhaustive

/-- Both terminal alternatives satisfy progress; cancellation need not win for this to pass.
The question is deliberately conditional on the three-step scenario. It does not establish that
the environment always schedules resolution, or that a live operation completes by a deadline. -/
query cancellationProgress on lifecycle
  verify cancellationResolves
  in cancellationRace
  limits shortTrace
  search exhaustive

/-- The successful branch is a witness that cancellation can lose. If found, it refutes the
stronger claim "every cancellation request ends in cancellation" in this model. It gives no
probability or frequency for either outcome, and is not evidence of a real Temporal execution.
All expected results in these comments remain expectations until the draft is implemented and
the corresponding checked Queries actually run. -/
query completionCanWin on lifecycle
  witness finalState succeeded
  in cancellationRace
  limits shortTrace

end Nexus
