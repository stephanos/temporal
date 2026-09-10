import Std
import Shared.SemanticData

/-!
Independent bounded response obligations. A transition first creates its trigger obligation,
then offers its response to every obligation. Only unanswered obligations spend a transition;
zero remaining transitions expires after that response opportunity. Resolved obligations are
immutable. `consumeMany` is the common finite, incremental and offline fold.
-/

namespace Shared.CorrelatedObligation

/-- The semantic predicate results at one admitted operation transition. -/
structure Match where
  trigger : Bool
  response : Bool
  deriving BEq, DecidableEq, Repr

/-- A response obligation has an inclusive remaining window or an immutable answer. -/
inductive Obligation where
  | pending (remaining : Nat)
  | satisfied
  | violated
  deriving BEq, DecidableEq, Repr

/-- Offer the current response before expiring its inclusive deadline. -/
def Obligation.consume (response : Bool) : Obligation → Obligation
  | .pending remaining => if response then .satisfied else
      match remaining with
      | 0 => .violated
      | remaining + 1 => .pending remaining
  | .satisfied => .satisfied
  | .violated => .violated

/-- Consume subsequent semantic coordinates for one independently captured trigger. -/
def Obligation.consumeMany (obligation : Obligation) : List Bool → Obligation
  | [] => obligation
  | response :: rest => Obligation.consumeMany (obligation.consume response) rest

/-- Each trigger creates a distinct obligation even when its predecessor remains live. -/
def consume (bound : Nat) (obligations : List Obligation)
    (coordinate : Match) : List Obligation :=
  (obligations ++ if coordinate.trigger then [Obligation.pending bound] else []).map
    (Obligation.consume coordinate.response)

/-- Shared stream execution. Chunking does not create or close semantic coordinates. -/
def consumeMany (bound : Nat) (obligations : List Obligation)
    (coordinates : List Match) : List Obligation :=
  coordinates.foldl (consume bound) obligations

/-- Every chunk split gives exactly the same obligations, including empty chunks. -/
theorem consumeMany_append (bound : Nat) (obligations : List Obligation)
    (first second : List Match) :
    consumeMany bound obligations (first ++ second) =
      consumeMany bound (consumeMany bound obligations first) second := by
  simp [consumeMany, List.foldl_append]

/-- Earlier satisfaction and violation cannot be repaired or erased by later responses. -/
theorem Obligation.resolved_immutable (responses : List Bool) :
    Obligation.consumeMany .satisfied responses = .satisfied ∧
      Obligation.consumeMany .violated responses = .violated := by
  induction responses with
  | nil => simp [consumeMany]
  | cons response rest ih => simpa [consumeMany, consume] using ih

/-- One countdown is satisfied precisely when the independent bounded response window contains
true. This reference uses a list window, not the countdown transition implementation. -/
theorem Obligation.pending_satisfied (bound : Nat) (responses : List Bool) :
    Obligation.consumeMany (.pending bound) responses = .satisfied ↔
      (responses.take (bound + 1)).any id = true := by
  induction bound generalizing responses with
  | zero =>
      cases responses with
      | nil => simp [consumeMany]
      | cons response rest =>
          cases response <;> simp [consumeMany, consume, resolved_immutable]
  | succ bound ih =>
      cases responses with
      | nil => simp [consumeMany]
      | cons response rest =>
          cases response <;> simp [consumeMany, consume, ih, resolved_immutable]

/-- Existing obligations evolve independently of every trigger born in the remaining stream. -/
theorem consumeMany_distributes (bound : Nat) (obligations : List Obligation)
    (coordinates : List Match) :
    consumeMany bound obligations coordinates =
      obligations.map (fun obligation => obligation.consumeMany (coordinates.map (·.response))) ++
        consumeMany bound [] coordinates := by
  induction coordinates generalizing obligations with
  | nil => simp [consumeMany, Obligation.consumeMany]
  | cons coordinate rest ih =>
      change consumeMany bound (consume bound obligations coordinate) rest =
        obligations.map (fun obligation => obligation.consumeMany
          (coordinate.response :: rest.map (·.response))) ++
          consumeMany bound (consume bound [] coordinate) rest
      rw [ih (consume bound obligations coordinate), ih (consume bound [] coordinate)]
      simp [consume, List.map_append, List.map_map, Obligation.consumeMany, List.append_assoc]

/-- Independent finite-trace reference: each triggered suffix must contain a response in its
inclusive window. No monitor state or countdown is consulted. -/
def closedReference (bound : Nat) : List Match → Bool
  | [] => true
  | coordinate :: rest =>
      (!coordinate.trigger || ((coordinate :: rest).take (bound + 1)).any (·.response)) &&
        closedReference bound rest

/-- All spawned countdowns agree with independent bounded suffix windows on deliberately closed
traces. Repeated triggers are quantified separately rather than coalesced. -/
theorem closed_agrees (bound : Nat) (coordinates : List Match) :
    (consumeMany bound [] coordinates).all (fun obligation => decide (obligation = .satisfied)) = closedReference bound coordinates := by
  induction coordinates with
  | nil => rfl
  | cons coordinate rest ih =>
      simp only [consumeMany, List.foldl_cons]
      rw [← consumeMany, consumeMany_distributes]
      simp only [List.all_append, ih]
      cases trigger : coordinate.trigger with
      | false => simp [consume, closedReference, trigger]
      | true =>
          simp only [consume, trigger, ite_true, List.nil_append, List.map_cons,
            List.map_nil, List.all_cons, List.all_nil, Bool.and_true]
          have agreement := Obligation.pending_satisfied bound
            ((coordinate :: rest).map (·.response))
          have same :
              decide (((Obligation.pending bound).consume coordinate.response |>.consumeMany
                (rest.map (·.response))) = .satisfied) =
              ((coordinate :: rest).take (bound + 1)).any (·.response) := by
            apply Bool.eq_iff_iff.mpr
            simpa [Obligation.consumeMany, ← List.map_take] using agreement
          simp only [same, closedReference, trigger, Bool.not_true, Bool.false_or]

open Shared.SemanticData

structure Predicate where
  field : Nat
  reference : Name
  equalsText : Option String
  deriving BEq, DecidableEq

structure Clause where
  id : String
  bound : Nat
  final : Bool
  trigger : Predicate
  response : Predicate
  deriving BEq, DecidableEq

/-- Predicate interpretation examines only the declared named transition values. -/
def Predicate.holds (predicate : Predicate) (action : Atom) (result : Shared.SemanticData.Result Atom Atom Atom) : Bool :=
  let values := match predicate.field with
    | 1 => [action]
    | 2 => [result.outcome]
    | 3 => [result.state]
    | 4 => result.facts
    | _ => []
  values.any fun value => value.definitionId == predicate.reference &&
    predicate.equalsText.all (· == value.value)

/-- One admitted table row determines each clause's trigger and response coordinate. -/
def Clause.coordinate (clause : Clause) (action : Atom) (result : Shared.SemanticData.Result Atom Atom Atom) :
    Match :=
  ⟨clause.trigger.holds action result, clause.response.holds action result⟩


abbrev Transition := Atom × Atom × Shared.SemanticData.Result Atom Atom Atom

variable {table : List Transition} {clauses : List Clause}

/-- One transition whose membership was checked before any monitor state mutation. -/
structure Admitted (table : List Transition) where
  value : Transition
  member : value ∈ table

/-- A window's state is the actual fold over its operation's admitted transition history. -/
structure Window (clauses : List Clause) (history : List (Admitted table)) where
  clause : { clause // clause ∈ clauses }
  obligations : List Obligation
  consistent : obligations = consumeMany clause.val.bound []
    (history.map fun row => clause.val.coordinate row.value.2.1 row.value.2.2)

structure Operation (table : List Transition) (clauses : List Clause) where
  key : String
  state : Atom
  history : List (Admitted table)
  windows : List (Window clauses history)

structure MonitorLimits where
  transitions : Nat
  obligations : Nat
  work : Nat
  deriving BEq, DecidableEq

/-- The bounded operation-local state shared by source and portable evaluation. -/
structure Monitor (table : List Transition) (clauses : List Clause) where
  initial : Atom
  operations : List (Operation table clauses) := []
  transitions : Nat := 0
  obligations : Nat := 0
  work : Nat := 0

private def Operation.start (table : List Transition) (clauses : List Clause)
    (key : String) (initial : Atom) : Operation table clauses := {
  key, state := initial, history := []
  windows := clauses.attach.map fun clause => ⟨clause, [], rfl⟩ }

private def Window.consume {history : List (Admitted table)} (window : Window clauses history)
    (row : Admitted table) : Window clauses (history ++ [row]) := {
  clause := window.clause
  obligations := Shared.CorrelatedObligation.consume window.clause.val.bound window.obligations
    (window.clause.val.coordinate row.value.2.1 row.value.2.2)
  consistent := by
    rw [List.map_append, consumeMany_append, ← window.consistent]
    rfl }

/-- All windows at a semantic coordinate see their triggers before their inclusive response chance. -/
def Monitor.consume (limits : MonitorLimits) (monitor : Monitor table clauses)
    (key : String) (row : Admitted table) : Except String (Monitor table clauses) := do
  let operation := (monitor.operations.find? (·.key == key)).getD
    (Operation.start table clauses key monitor.initial)
  if key.isEmpty || operation.state != row.value.1 then throw "invalid transition continuity"
  if monitor.transitions >= limits.transitions then throw "transitions exhausted"
  let created := (operation.windows.filter fun window =>
    (window.clause.val.coordinate row.value.2.1 row.value.2.2).trigger).length
  if monitor.obligations + created > limits.obligations then throw "obligations exhausted"
  let maximumFacts := table.foldl (fun maximum row => max maximum row.2.2.facts.length) 0
  let candidates := table.filter fun candidate => candidate.1 == row.value.1 &&
    candidate.2.1 == row.value.2.1
  let cost := monitor.obligations + 16 * clauses.length * (monitor.transitions + 1) *
    (1 + maximumFacts) + monitor.operations.length + candidates.length
  if monitor.work + cost > limits.work then throw "work exhausted"
  let next : Operation table clauses := {
    key
    state := row.value.2.2.state
    history := operation.history ++ [row]
    windows := operation.windows.map (Window.consume · row) }
  let operations := if monitor.operations.any (·.key == key) then
    monitor.operations.map fun current => if current.key == key then next else current
    else monitor.operations ++ [next]
  pure { monitor with
    operations
    transitions := monitor.transitions + 1
    obligations := monitor.obligations + created
    work := monitor.work + cost }

inductive Verdict where
  | satisfied | violated | unresolved
  deriving BEq, DecidableEq, Repr

/-- Endpoint policy preserves established violations and never promotes missing evidence. -/
def answer (obligations : List Obligation) (closed incomplete final : Bool) : Verdict :=
  if obligations.contains .violated then .violated
  else if incomplete then .unresolved
  else if obligations.all (fun obligation => decide (obligation = .satisfied)) then .satisfied
  else if closed && final then .violated else .unresolved

/-- Closing is a semantic endpoint choice; incomplete evidence never supplies a deadline. -/
def Monitor.answers (monitor : Monitor table clauses) (closed incomplete : Bool) : List Verdict :=
  clauses.map fun clause =>
    let obligations := monitor.operations.flatMap fun operation => operation.windows.flatMap fun window =>
      if window.clause.val.id == clause.id then window.obligations else []
    answer obligations closed incomplete clause.final

end Shared.CorrelatedObligation
