import DslExperiment.Property

/-! Finite trace queries preserve scenario history instead of merging equal model states.
Semantic depth and computational work budget are independent. An exhausted budget cannot prove
scenario emptiness or a universal property. Exact regressions and broader scenarios share a model. -/

namespace DslExperiment

/-- A small finite scenario algebra; `ordered` permits intervening events. -/
inductive Scenario where
  | any
  | exact (steps : List Step)
  | ordered (first second : Step)
  | resolved (operation : Fin 2)
  | either (left right : Scenario)
  deriving Repr, DecidableEq, BEq

/-- Trace constraints select model behavior; they never create transitions. -/
def Scenario.accepts : Scenario → List Step → Bool
  | .any, _ => true
  | .exact expected, steps => steps == expected
  | .ordered first second, steps =>
      match steps.dropWhile (· != first) with
      | [] => false
      | _ :: rest => rest.contains second
  | .resolved operation, steps => steps.any fun step =>
      step.operation == operation && (step.event == .canceled || step.event == .completed)
  | .either left right, steps => left.accepts steps || right.accepts steps

/-- Witness discovery and universal checking are distinct requests. -/
inductive Question where
  | witness | verify
  deriving Repr, DecidableEq, BEq

/-- Answers carry scope/completeness separately through `QueryReport`. -/
inductive Answer where
  | witness | verified | counterexample | noWitness | unsatisfiable | unexercised | unknown
  deriving Repr, DecidableEq, BEq

/-- Closing every selected finite prefix is an explicit semantic choice, never a work-limit default. -/
inductive EndpointPolicy where
  | closedFinitePrefixes | runtimePrefixes | terminalWorlds
  deriving Repr, DecidableEq, BEq

/-- `none` means a work limit prevented establishing absence. -/
structure QueryReport where
  endpoints : EndpointPolicy
  maxDepth : Nat
  satisfiable : Option Bool
  exercised : Option Bool
  complete : Bool
  evaluated : Nat
  answer : Answer
  selected : Option (List Step)
  deriving Repr, DecidableEq, BEq

/-- Every visited prefix remains distinct, including paths reaching the same world. -/
structure Search where
  traces : List (List Step) := []
  complete : Bool := false
  evaluated : Nat := 0
  deriving Repr, DecidableEq, BEq

private def visit (maxDepth : Nat) : Nat → List (World × List Step) → Search → Search
  | _, [], result => { result with complete := true }
  | 0, _ :: _, result => result
  | fuel + 1, (world, trace) :: rest, result =>
      let next := if trace.length < maxDepth then
          (successors world).map fun (step, nextWorld) => (nextWorld, trace ++ [step])
        else []
      visit maxDepth fuel (next ++ rest)
        { result with traces := result.traces ++ [trace], evaluated := result.evaluated + 1 }

/-- Enumerate bounded prefixes with a separate hard limit on prefix visits. -/
def explore (maxDepth budget : Nat) : Search := visit maxDepth budget [(initial, [])] {}

private def terminal (phase : Phase) : Bool := phase == .canceled || phase == .succeeded

private def eligible (policy : EndpointPolicy) (trace : List Step) : Bool :=
  match policy with
  | .closedFinitePrefixes | .runtimePrefixes => true
  | .terminalWorlds => (replay trace).any fun world => terminal world.1 && terminal world.2

/-- Query bounded finite prefixes; the work budget limits visited prefixes, including the empty one. -/
def query (clause : ResponseClause) (scenario : Scenario) (question : Question)
    (endpoints : EndpointPolicy) (maxDepth budget : Nat)
    (requireTrigger : Bool := true) : QueryReport := Id.run do
  let search := explore maxDepth budget
  let allowed := search.traces.filter fun trace => scenario.accepts trace && eligible endpoints trace
  let closure := if endpoints == .runtimePrefixes then Closure.runtimePrefix else .closedModel
  let unresolved := allowed.any fun trace => evaluate clause closure trace == .inconclusive
  let exercised := allowed.any fun trace => trace.any clause.trigger.matches
  let existsOrUnknown := fun found => if found then some true
    else if search.complete then some false else none
  let selected := allowed.find? fun trace =>
    let verdict := evaluate clause closure trace
    match question with
    | .witness => verdict == .satisfied && (!requireTrigger || trace.any clause.trigger.matches)
    | .verify => verdict == .violated
  let answer := if selected.isSome then
      if question == .witness then Answer.witness else .counterexample
    else if !search.complete then .unknown
    else if allowed.isEmpty then .unsatisfiable
    else if requireTrigger && !exercised then .unexercised
    else if unresolved then .unknown
    else if question == .witness then .noWitness
    else .verified
  return {
    endpoints
    maxDepth
    satisfiable := existsOrUnknown (!allowed.isEmpty)
    exercised := existsOrUnknown exercised
    complete := search.complete
    evaluated := search.evaluated
    answer
    selected
  }

end DslExperiment
