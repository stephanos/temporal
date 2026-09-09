/-!
# Nexus3 — user-facing authoring draft

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
* An Action either requests a change or waits for an observed change.
* The model defines which transitions and outcomes are possible.
* A Property states a requirement to check against those possibilities.
* A Behavior restricts the traces considered for one scenario.
* A Query asks for an example or checks a requirement within explicit limits.

A trace is an initial state followed by steps with an Action and a result, for example:
`scheduled → started → succeeded`. Actions explain why each arrow occurs.
Properties do not repair or filter the model's transitions to make requirements pass.

## Lean syntax versus proposed syntax

`namespace`, `inductive`, `where`, and the constructor bars `|` are ordinary Lean syntax.
The success-only forms of the `model`, `property`, `behavior`, `limits`, and `query` blocks compile
in `Nexus.lean`; the broader forms below remain proposed. In particular, `+` and `→` in a
transition row are visual separators, not Lean addition or a function type. `/-- ... -/` introduces
documentation; `/- ... -/` is a block comment.

IDs derive from the feature namespace, declaration kind, and name. `Integration.md` specifies
the convention and Case boundary, including the optional per-declaration compatibility ID this
broader draft keeps for renames; neither derivation nor that override needs a parallel identity
registry to maintain. The success-only slice is delivered and produces a checked Case. The
proposed forms below, broader cancellation model admission, and cancellation Case integration
remain design work.
-/

-- A namespace groups related names; the full type name is `Temporal.Feature.Nexus3.State`.
namespace Temporal.Feature.Nexus3

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

/-- Actions distinguish a command from a wait for evidence. Waiting does not cause completion.
Separating an Action from its result lets one Action admit multiple outcomes without allowing
a test to choose which outcome the system produces. The command/wait classification is checked
in `Integration.md`; it is not inferred from these names. -/
inductive Action where
  /-- Wait for asynchronous acknowledgment after scheduling the operation. -/
  | awaitStart
  /-- Wait for successful handler completion before cancellation in this model. -/
  | awaitSuccess
  /-- Asks for cancellation without promising which terminal result will follow. -/
  | requestCancel
  /-- Wait for cancellation or successful completion; neither result is chosen by the wait. -/
  | awaitResolution

/-- An outcome says what happened on a step, independently of the state it leaves behind.
Later extensions can distinguish, for example, accepted and rejected requests with the same
destination state. This draft includes only the outcomes listed here. -/
inductive Outcome where
  | acknowledged
  | cancellationRequested
  | operationCanceled
  | completed

/-- A Model Fact is information exposed by a transition to Properties. It is not runtime
Evidence; the integration must establish it from correlated Observations before using it. -/
inductive Fact where
  | terminal

/-
`on lifecycle` below refers to this model. Its derived Target ID is
`temporal.nexus3.target.lifecycle`.
Likewise, `cancellationResolves` gets `temporal.nexus3.property.cancellationResolves`. IDs survive
builds, comment edits, and declaration reordering. Renaming a declaration changes its ID and the
IDs of its owned members; affected generated fixtures must be regenerated. A declaration that must
keep its old ID across a rename may instead carry an explicit compatibility ID. That override is
declaration-local — it does not cascade to owned members, which keep deriving from the owner's new
name — and it preserves identity only: the Behavior Fingerprint still follows the checked meaning,
and admission judges the override like any other ID. The success-only syntax that compiles in
`Nexus.lean` accepts no such input; `Integration.md` holds the convention.

The state and Action declarations supply the complete vocabulary, including terminal states
with no outgoing rows. `initial` requires every scenario to begin at `scheduled`.
Reaching `started` requires an explicit `awaitStart` step, including in cancellation scenarios.

`terminal` identifies final states. Model admission must reject any outgoing transition from
them, rather than silently removing a contradictory row. This is a structural requirement on
the transition relation, checked once during admission, not an extra trace Property or Query.
-/
model lifecycle
  role operation
  states State
  actions Action
  outcomes Outcome
  facts Fact
  initial [scheduled]
  terminal [canceled, succeeded]

  -- Read `before + action → result` as one permitted step, not an instruction to execute it.
  -- `oneOf` lists alternatives, not priorities or a random distribution; both must be considered.
  -- Each result has a state and outcome; omitted facts mean the empty set, not inferred facts.
  transitions
    start: scheduled + awaitStart → { state := started, outcome := acknowledged }
    success: started + awaitSuccess →
      { state := succeeded, outcome := completed, facts := [terminal] }
    request: started + requestCancel →
      { state := cancelRequested, outcome := cancellationRequested }
    resolution: cancelRequested + awaitResolution → oneOf [
      { state := canceled, outcome := operationCanceled, facts := [terminal] },
      { state := succeeded, outcome := completed, facts := [terminal] }
    ]

/-
Each row lists all allowed results; absent pairs have no transition. Model Outcomes are now
separate from destination states, and every alternative supplies a complete result.
`awaitResolution` permits either terminal result; a scenario cannot force cancellation to win.

For example, there is no `scheduled + awaitSuccess` row. This omits synchronous completion
from the draft; it does not claim that real Nexus operations cannot complete synchronously.
Likewise, the absence of late-event rows says nothing about whether a real server ignores or
rejects a late event. Those behaviors need explicit modeling before they can be checked.
-/

/-- A safety requirement: every cancellation-request step must leave the operation pending
cancellation. `when` selects the triggering step, and `resultingState` reads that same step's
output. A transition directly to `canceled` would violate this particular request/response model.
A trace with no request satisfies this conditional requirement without exercising it. -/
property cancellationIsARequest on lifecycle
  for operation
  when action requestCancel
  require requestState: resultingState cancelRequested

/-- A bounded progress requirement: a request must be followed by either terminal result within
one additional transition of that same operation. The intended existing temporal semantics allow a
response on the triggering step; our request row does not produce one, so resolution needs the next
step. This is a bound in model steps, not seconds. The Behavior below explicitly includes progress.
`for operation` binds the trigger and response to one operation; other operations do not consume
its bound. Operation-scoped counting is now a delivered generic capability, qualified through
non-cancellation fixtures; it was never Nexus2 functionality, which counts global transitions.
What stays unsupported is this cancellation-specific use of it, which `Integration.md` rejects at
Case production. A trace ending immediately after the request cannot demonstrate the required
response. -/
property cancellationResolves on lifecycle
  for operation
  when action requestCancel
  require terminalResponse: eventually fact terminal
    within 1 operation_transition

/-- A reusable condition for witness Queries. It inspects the final state of this operation,
not whether a controller instruction returned successfully. Merely declaring a Property does
not run it; the Queries below explicitly select it. -/
property successfulResult on lifecycle
  for operation
  require successState: finalState succeeded

/-- A Behavior selects model traces, not runtime instructions. `exactly` fixes the selected
Action sequence and its length. In `completion: awaitSuccess`, the name before `:` labels this
occurrence, while the name after `:` identifies the Action. Labels distinguish occurrences if
the same Action is later allowed more than once in a scenario. -/
behavior successfulCompletion on lifecycle
  operation starts scheduled
  actions exactly [start: awaitStart, completion: awaitSuccess]

/-- This scenario admits two traces:
`scheduled → started → cancelRequested → canceled` and
`scheduled → started → cancelRequested → succeeded`.
The requests are the same in both traces; the model supplies the resolution alternatives.
Completion before the request and a request with no later resolution are outside this Behavior.
Claims checked only here therefore do not cover every possible cancellation interleaving. -/
behavior cancellationRace on lifecycle
  operation starts scheduled
  actions exactly [start: awaitStart, request: requestCancel, resolution: awaitResolution]

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
every trace succeeds. `witness successfulResult` refers to the named Property above, so Queries
reuse the Property language rather than introducing their own predicate evaluator. -/
query completion on lifecycle
  witness successfulResult
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
  witness successfulResult
  in cancellationRace
  limits shortTrace

end Temporal.Feature.Nexus3
