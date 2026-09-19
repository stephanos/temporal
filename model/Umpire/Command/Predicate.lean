import Umpire.Command.Authoring

/-!
# A Property's predicate, enumerated into clauses

A `property` names a machine and an ordinary Lean predicate: `Step → Bool` for a claim about the
step one Action produces, or `Step → Step → Bool` for a claim about the step before and the step
after. What Search, the Behavior Fingerprint and Contract lowering read is the clause language a
Property has always carried -- a trigger pattern and the state, outcome or fact it requires -- so
this module is the bridge, the way `Umpire.Command.Finite` is the bridge from a step function to
the finite table: it evaluates the predicate over the machine's own table and writes down the
clauses that say the same thing.

The reading is the one a `require:` block used to spell out by hand. Over the steps the trigger
admits, the predicate **fixes** a value when every step it accepts carries that value and it
rejects any accepted step with the value changed: the state, when changing the state to any other
of the machine's states is rejected; the outcome likewise; a fact, when removing it is rejected.
The fixed values are the requirements, and they must carry the predicate exactly -- every step of
the table the requirements admit is one the predicate accepts, and the other way round. A predicate
the clause language cannot carry that way (a disjunction across fields, a claim that reads the step
before beyond its state) is refused with the step that tells the two apart, rather than
approximated by clauses that would let Search find a witness the predicate rejects.

The requirements are read off the predicate, not off the table's coincidences: a predicate that
only names the outcome does not gain a state clause because every accepted step happened to share
one, since changing that state is not something the predicate rejects.
-/

namespace Umpire.Command

open Umpire

/-- Why a predicate could not be enumerated into clauses, in the author's terms. -/
inductive PredicateRefusal where
  /-- The trigger admits no step of the table, so there is nothing to claim about. -/
  | noStep (trigger : String)
  /-- The predicate accepts no step the trigger admits. -/
  | neverHolds (trigger : String)
  /-- The predicate accepts every step the trigger admits and fixes no value, so it claims nothing. -/
  | fixesNothing (trigger : String)
  /-- The requirements the predicate fixes admit a step it rejects, or reject one it accepts. -/
  | notCarried (trigger witness : String)
  deriving BEq, Repr, Inhabited

def PredicateRefusal.message : PredicateRefusal → String
  | .noStep trigger =>
      s!"no step of this machine is admitted at {trigger}, so the predicate has nothing to hold on"
  | .neverHolds trigger =>
      s!"the predicate holds on no step of this machine at {trigger}; a Property claims something \
the machine does, so it fixes a state, an outcome or a fact some step reaches"
  | .fixesNothing trigger =>
      s!"the predicate holds on every step of this machine at {trigger} and fixes no state, \
outcome or fact, so it claims nothing"
  | .notCarried trigger witness =>
      s!"the predicate is not a conjunction of one state, one outcome and facts at {trigger}: the \
clauses it fixes cannot tell {witness} apart from the steps it accepts; a Property is one such \
conjunction, so split it or restate it"

/-- How the members of a machine's domains are spelled, which is what a clause names. -/
structure StepKeys (State Outcome Fact : Type) where
  state : State → String
  outcome : Outcome → String
  fact : Fact → String

/-- What a `property` command reads back after evaluating its predicate: the groups to emit, or
the refusal to report at the predicate. Plain data, because it is evaluated during elaboration. -/
structure EnumeratedProperty where
  refusal : Option PredicateRefusal := none
  groups : List PropertyGroup := []
  deriving Repr, Inhabited

private def render (keys : StepKeys State Outcome Fact) (step : Step State Outcome Fact) : String :=
  s!"the step to {keys.state step.state} with outcome {keys.outcome step.outcome} and facts \
[{", ".intercalate (step.facts.map keys.fact)}]"

/-- The requirements one predicate fixes over the steps a trigger admits, given the predicate at
the trigger -- already closed over the step before, for a transition claim -- and the steps the
trigger admits. `[]` with nothing rejected is a predicate that claims nothing there.

A value is fixed when every accepted step carries it, the domain has another value to change it to,
and the predicate rejects every accepted step with it changed. The second condition is what keeps a
one-member domain from being "fixed" by every predicate: on a machine with one outcome, a claim
about the outcome claims nothing, and it is not read as if it did. -/
private def fixedRequirements [BEq State] [BEq Outcome] [BEq Fact]
    (states : List State) (outcomes : List Outcome)
    (keys : StepKeys State Outcome Fact)
    (results : List (Step State Outcome Fact))
    (accepts : Step State Outcome Fact → Bool)
    (label : String := "") :
    Except (String → PredicateRefusal) (List PropertyRequirement) := do
  let accepted := results.filter accepts
  let some first := accepted.head?
    | throw PredicateRefusal.neverHolds
  let altered := fun (step : Step State Outcome Fact) (state : State) =>
    ({ step with state } : Step State Outcome Fact)
  let alteredOutcome := fun (step : Step State Outcome Fact) (outcome : Outcome) =>
    ({ step with outcome } : Step State Outcome Fact)
  let without := fun (step : Step State Outcome Fact) (fact : Fact) =>
    ({ step with facts := step.facts.filter (· != fact) } : Step State Outcome Fact)
  let state? :=
    if accepted.all (·.state == first.state) && states.any (· != first.state) &&
        accepted.all (fun step => states.all fun other =>
          other == first.state || !accepts (altered step other)) then
      some first.state
    else none
  let outcome? :=
    if accepted.all (·.outcome == first.outcome) && outcomes.any (· != first.outcome) &&
        accepted.all (fun step => outcomes.all fun other =>
          other == first.outcome || !accepts (alteredOutcome step other)) then
      some first.outcome
    else none
  let facts := first.facts.eraseDups.filter fun fact =>
    accepted.all (fun step => step.facts.contains fact && !accepts (without step fact))
  let carried := fun (step : Step State Outcome Fact) =>
    state?.all (step.state == ·) && outcome?.all (step.outcome == ·) &&
      facts.all step.facts.contains
  -- The requirements are a conjunction, and they must be the predicate over the table: a step the
  -- conjunction admits that the predicate rejects would let Search find a witness the author's
  -- claim does not accept, and a step it rejects that the predicate accepts would hide one.
  match results.find? fun step => carried step != accepts step with
  | some witness => throw fun trigger => .notCarried trigger (render keys witness)
  | none => pure ()
  pure <|
    (state?.toList.map fun state =>
      PropertyRequirement.stateClause (label ++ "state-" ++ keys.state state) (keys.state state)) ++
    (outcome?.toList.map fun outcome =>
      PropertyRequirement.outcomeClause (label ++ "outcome-" ++ keys.outcome outcome)
        (keys.outcome outcome)) ++
    (facts.map fun fact =>
      PropertyRequirement.factClause (label ++ "fact-" ++ keys.fact fact) (keys.fact fact))

/-- A same-step claim: the predicate over the steps one Action produces, wherever the machine
takes it. `actionKey` is the Action as its clauses name it. -/
def enumerateSameStep [BEq State] [BEq Outcome] [BEq Fact]
    (states : List State) (outcomes : List Outcome)
    (keys : StepKeys State Outcome Fact)
    (actionKey : String)
    (results : List (Step State Outcome Fact))
    (holds : Step State Outcome Fact → Bool) : EnumeratedProperty :=
  let trigger := "`" ++ actionKey ++ "`"
  if results.isEmpty then { refusal := some (.noStep trigger) } else
  match fixedRequirements states outcomes keys results holds with
  | .error refusal => { refusal := some (refusal trigger) }
  | .ok [] => { refusal := some (.fixesNothing trigger) }
  | .ok requirements => { groups := [{ trigger := .action actionKey, requirements }] }

/-- The steps the machine arrives at one state by, which is what "the step before" a transition
claim reads. A state nothing arrives at -- a start state -- is entered by no step, so the claim
sees it under each outcome with no facts, which is every step before that could be. -/
private def arrivals [BEq State]
    (outcomes : List Outcome)
    (transitions : List (FiniteTransitionRow State Action Outcome Fact))
    (state : State) : List (Step State Outcome Fact) :=
  let arrived := transitions.flatMap fun row => row.results.filter (·.state == state)
  if arrived.isEmpty then
    outcomes.map fun outcome => { outcome, state, facts := [] }
  else arrived

/-- A transition claim: the predicate over the step before and the step after, enumerated into one
group per prior state it constrains. A prior state at which the predicate accepts every step and
fixes nothing is unconstrained and contributes no group; one at which it accepts none is a refusal,
because a claim that no step leaves a state the table leaves is a claim about the wrong machine. -/
def enumerateTransition [BEq State] [BEq Outcome] [BEq Fact]
    (states : List State) (outcomes : List Outcome)
    (keys : StepKeys State Outcome Fact)
    (transitions : List (FiniteTransitionRow State Action Outcome Fact))
    (holds : Step State Outcome Fact → Step State Outcome Fact → Bool) : EnumeratedProperty :=
  let groups := states.filterMap fun prior =>
    let results := (transitions.filter (·.source == prior)).flatMap (·.results)
    if results.isEmpty then none else
    let befores := arrivals outcomes transitions prior
    -- The claim at this prior state is the predicate closed over every step before that reaches
    -- it: a step after is accepted only if it is accepted whichever step came before.
    let accepts := fun (after : Step State Outcome Fact) =>
      befores.all fun before => holds before after
    let trigger := "prior state `" ++ keys.state prior ++ "`"
    -- The label carries the prior state, so two prior states that fix the same value are two
    -- clauses rather than one id declared twice.
    match fixedRequirements states outcomes keys results accepts
        (label := "from-" ++ keys.state prior ++ "-") with
    | .error refusal => some (Except.error (refusal trigger))
    | .ok [] => none
    | .ok requirements => some (Except.ok
        ({ trigger := .priorState (keys.state prior), requirements } : PropertyGroup))
  match groups.findSome? (fun group => match group with | .error refusal => some refusal | .ok _ => none) with
  | some refusal => { refusal := some refusal }
  | none =>
      let emitted := groups.filterMap fun group => match group with
        | .ok group => some group
        | .error _ => none
      if emitted.isEmpty then { refusal := some (.fixesNothing "every prior state") }
      else { groups := emitted }

end Umpire.Command
