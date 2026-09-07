import DslExperiment.Model

/-! A deliberately small temporal fragment: each matching trigger creates an independent,
operation-scoped bounded response obligation. `evaluate` folds a passive machine; `reference`
checks suffixes independently. A closed model trace and an open runtime prefix have different
closure policies. Neither interpretation claims unbounded liveness. -/

namespace DslExperiment

/-- Only admitted steps of the trigger's operation advance this clock. -/
inductive Clock where
  | operationTransitions
  deriving Repr, DecidableEq, BEq

/-- Step predicates cannot contain driver effects or raw evidence expressions. -/
inductive StepPredicate where
  | event (value : Event)
  | anyOf (values : List Event)
  deriving Repr, DecidableEq, BEq

/-- Interpret the supported step predicate fragment. -/
def StepPredicate.matches (predicate : StepPredicate) (step : Step) : Bool :=
  match predicate with
  | .event value => step.event == value
  | .anyOf values => values.contains step.event

/-- A response may occur at the trigger coordinate or the inclusive deadline. -/
structure ResponseClause where
  trigger : StepPredicate
  response : StepPredicate
  bound : Nat
  clock : Clock := .operationTransitions
  deriving Repr, DecidableEq, BEq

/-- Pending obligations fail on closed model traces and remain unresolved on runtime prefixes. -/
inductive Closure where
  | closedModel | runtimePrefix
  deriving Repr, DecidableEq, BEq

/-- A violation is sticky; missing future evidence cannot establish satisfaction. -/
inductive Verdict where
  | satisfied | violated | inconclusive
  deriving Repr, DecidableEq, BEq

/-- Each trigger keeps its own age, even when several triggers share an operation key. -/
structure Obligation where
  operation : Fin 2
  age : Nat
  deriving Repr, DecidableEq, BEq

/-- The passive machine stores no commands or runtime payloads. -/
structure Monitor where
  pending : List Obligation := []
  violated : Bool := false
  triggers : Nat := 0
  deriving Repr, DecidableEq, BEq

/-- Consume one admitted semantic step. -/
def Monitor.consume (clause : ResponseClause) (monitor : Monitor) (step : Step) : Monitor := Id.run do
  let mut next := { monitor with pending := [] }
  for obligation in monitor.pending do
    if obligation.operation != step.operation then
      next := { next with pending := next.pending ++ [obligation] }
    else
      let age := obligation.age + 1
      if age ≤ clause.bound && clause.response.matches step then
        pure ()
      else if age ≥ clause.bound then
        next := { next with violated := true }
      else
        next := { next with pending := next.pending ++ [{ obligation with age }] }
  if clause.trigger.matches step then
    next := { next with triggers := next.triggers + 1 }
    if clause.response.matches step then
      pure ()
    else if clause.bound == 0 then
      next := { next with violated := true }
    else
      next := { next with pending := next.pending ++ [⟨step.operation, 0⟩] }
  return next

/-- Close a prefix without confusing a wall-time stop with a semantic deadline. -/
def Monitor.finish (monitor : Monitor) (closure : Closure) : Verdict :=
  if monitor.violated then .violated
  else if monitor.pending.isEmpty then .satisfied
  else if closure == .closedModel then .violated
  else .inconclusive

/-- Interpret a clause over semantic events. Callers owning a model trace must establish admission;
`evaluateAdmitted` does that check, while query and projection consumers already enforce it. -/
def evaluate (clause : ResponseClause) (closure : Closure) (steps : List Step) : Verdict :=
  (steps.foldl (Monitor.consume clause) {}).finish closure

/-- Public trace entry point rejects wrong-phase events before counting any semantic transitions. -/
def evaluateAdmitted (clause : ResponseClause) (closure : Closure) (steps : List Step) : Option Verdict :=
  (replay steps).map fun _ => evaluate clause closure steps

/-- Resuming a monitor from a prefix produces exactly the same state as one whole-trace fold. -/
theorem monitor_append (clause : ResponseClause) (earlier later : List Step) (monitor : Monitor) :
    (earlier ++ later).foldl (Monitor.consume clause) monitor =
      later.foldl (Monitor.consume clause) (earlier.foldl (Monitor.consume clause) monitor) := by
  induction earlier generalizing monitor with
  | nil => rfl
  | cons step rest ih => exact ih (monitor.consume clause step)

private def combine (left right : Verdict) : Verdict :=
  if left == .violated || right == .violated then .violated
  else if left == .inconclusive || right == .inconclusive then .inconclusive
  else .satisfied

private def atTrigger (clause : ResponseClause) (closure : Closure)
    (trigger : Step) (suffix : List Step) : Verdict :=
  let matching := suffix.filter fun step => step.operation == trigger.operation
  let window := (trigger :: matching).take (clause.bound + 1)
  if window.any clause.response.matches then .satisfied
  else if matching.length ≥ clause.bound || closure == .closedModel then .violated
  else .inconclusive

/-- Independent finite-trace semantics inspects each trigger's scoped suffix, without monitor state. -/
def reference (clause : ResponseClause) (closure : Closure) : List Step → Verdict
  | [] => .satisfied
  | step :: rest =>
      let current := if clause.trigger.matches step then atTrigger clause closure step rest
        else .satisfied
      combine current (reference clause closure rest)

/-- The developer-facing temporal spelling elaborates directly to the typed obligation. -/
syntax "whenever " term " eventually " term " within " num " operationTransitions" : term

macro_rules
  | `(whenever $trigger eventually $response within $bound:num operationTransitions) =>
      `(ResponseClause.mk $trigger $response $bound Clock.operationTransitions)

/-- A plain constructor and the temporal surface have definitionally identical meanings. -/
theorem temporal_surface_agrees (trigger response : StepPredicate) :
    (whenever trigger eventually response within 1 operationTransitions) =
      ResponseClause.mk trigger response 1 Clock.operationTransitions := rfl

#print axioms temporal_surface_agrees
#print axioms monitor_append

end DslExperiment
