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

A same-step claim may also fix one field of a state whose other fields vary -- a composition's
member phase while the other member moves -- by the same reading one field at a time: every
accepted step carries the field's value, and changing that field alone, to any other value its
catalog holds, is rejected. The field reading is only consulted when the state, outcome and facts
do not already carry the predicate, so a claim those carry keeps the clauses it has always had.
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
  /-- The predicate reads the step before beyond its state: it accepts a step after some of the
  steps that arrive at the prior state and rejects it after others. -/
  | readsBefore (trigger witness : String)
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
  | .readsBefore trigger witness =>
      s!"the predicate reads the step before beyond its state at {trigger}: it accepts {witness} \
after some of the steps that arrive there and not after others; a transition claim is about the \
state the step before reached, so read `before.state` or split the claim"

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

/-- The state fields a predicate fixes over the accepted steps, as (field, spelling) pairs in the
state's field order. `fields` reads a state's fields by name; the catalog of one field is every
spelling the states hold it at. A state with that one field changed is found among `states`, since
a composition's state is the author's structure and the states it reaches are the only ones there
are; a field is fixed when some such change exists and the predicate rejects every one. -/
private def fixedFields [BEq State]
    (states : List State)
    (fields : State → List (String × String))
    (accepted : List (Step State Outcome Fact))
    (accepts : Step State Outcome Fact → Bool) : List (String × String) :=
  let catalog := states.map fun state => (state, fields state)
  let heldAt := fun (state : State) => ((catalog.find? (·.1 == state)).map (·.2)).getD []
  match accepted.head? with
  | none => []
  | some first => (heldAt first.state).filter fun (field, spelling) =>
    let others := (catalog.filterMap fun (_, held) => held.lookup field).eraseDups.filter (· != spelling)
    -- The states that hold `held` with `field` changed to `other` and every other field kept.
    let changed := fun (held : List (String × String)) (other : String) =>
      catalog.filterMap fun (state, candidate) =>
        if candidate == held.map (fun (name, value) => (name, if name == field then other else value))
        then some state else none
    let alterations := accepted.flatMap fun step =>
      others.flatMap fun other => (changed (heldAt step.state) other).map fun state =>
        ({ step with state } : Step State Outcome Fact)
    accepted.all (fun step => (heldAt step.state).lookup field == some spelling) &&
      !alterations.isEmpty && alterations.all (!accepts ·)

/-- The requirements one predicate fixes over the steps a trigger admits, given the predicate at
the trigger -- already closed over the step before, for a transition claim -- and the steps the
trigger admits. `[]` with nothing rejected is a predicate that claims nothing there.

A value is fixed when every accepted step carries it, the domain has another value to change it to,
and the predicate rejects every accepted step with it changed. The second condition is what keeps a
one-member domain from being "fixed" by every predicate: on a machine with one outcome, a claim
about the outcome claims nothing, and it is not read as if it did.

`fields` reads a state's fields for a claim that may fix one of them; a claim that does not passes
none. A field is fixed only where no whole state is and the state, outcome and facts alone do not
carry the predicate, which is what keeps every claim they carry enumerating as it did. -/
private def fixedRequirements [BEq State] [BEq Outcome] [BEq Fact]
    (states : List State) (outcomes : List Outcome)
    (keys : StepKeys State Outcome Fact)
    (results : List (Step State Outcome Fact))
    (accepts : Step State Outcome Fact → Bool)
    (label : String := "")
    (fields : State → List (String × String) := fun _ => []) :
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
  let carriedBy := fun (held : List (String × String)) (step : Step State Outcome Fact) =>
    state?.all (step.state == ·) && outcome?.all (step.outcome == ·) &&
      facts.all step.facts.contains &&
      held.all fun (field, spelling) => (fields step.state).lookup field == some spelling
  let wholeCarries := (state?.isSome || outcome?.isSome || !facts.isEmpty) &&
    results.all fun step => carriedBy [] step == accepts step
  let held := if state?.isSome || wholeCarries then [] else fixedFields states fields accepted accepts
  -- The requirements are a conjunction, and they must be the predicate over the table: a step the
  -- conjunction admits that the predicate rejects would let Search find a witness the author's
  -- claim does not accept, and a step it rejects that the predicate accepts would hide one.
  match results.find? fun step => carriedBy held step != accepts step with
  | some witness => throw fun trigger => .notCarried trigger (render keys witness)
  | none => pure ()
  pure <|
    (state?.toList.map fun state =>
      PropertyRequirement.stateClause (label ++ "state-" ++ keys.state state) (keys.state state)) ++
    (held.map fun (field, spelling) =>
      PropertyRequirement.stateFieldClause (label ++ "field-" ++ field ++ "-" ++ spelling) field
        spelling) ++
    (outcome?.toList.map fun outcome =>
      PropertyRequirement.outcomeClause (label ++ "outcome-" ++ keys.outcome outcome)
        (keys.outcome outcome)) ++
    (facts.map fun fact =>
      PropertyRequirement.factClause (label ++ "fact-" ++ keys.fact fact) (keys.fact fact))

/-- A same-step claim: the predicate over the steps one Action produces, wherever the machine
takes it. `actionKey` is the Action as its clauses name it, and `fields` reads a state's fields by
name, so the claim may fix one of them. -/
def enumerateSameStep [BEq State] [BEq Outcome] [BEq Fact]
    (states : List State) (outcomes : List Outcome)
    (keys : StepKeys State Outcome Fact)
    (fields : State → List (String × String))
    (actionKey : String)
    (results : List (Step State Outcome Fact))
    (holds : Step State Outcome Fact → Bool) : EnumeratedProperty :=
  let trigger := "`" ++ actionKey ++ "`"
  if results.isEmpty then { refusal := some (.noStep trigger) } else
  match fixedRequirements states outcomes keys results holds (fields := fields) with
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
    let trigger := "prior state `" ++ keys.state prior ++ "`"
    -- The claim at this prior state is the predicate over the steps that arrive there. It must
    -- say the same of every step after whichever step came before: one it accepts after some
    -- arrivals and not others reads the step before beyond its state, and closing it over the
    -- arrivals would strengthen or weaken what the author wrote rather than carry it.
    match results.find? fun after =>
        befores.any (fun before => holds before after) != befores.all (fun before => holds before after) with
    | some witness => some (Except.error (.readsBefore trigger (render keys witness)))
    | none =>
    let accepts := fun (after : Step State Outcome Fact) =>
      befores.all fun before => holds before after
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
