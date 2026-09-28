import Umpire.Command.Authoring
import Umpire.Command.Compose

/-!
# What a composed table agrees with

`compose` emits the table its walk found as a literal, and a literal says nothing about how it was
found. This module is the check the command decides in the kernel for each composition and what a
`true` proves: `composedTableAgrees` reads the members' tables and the composed literal as catalog
positions, and `ComposedAgreement.ofChecked` turns it into soundness and completeness of the
literal with respect to the composition of the members' tables over every composed state the
starts reach.

**Positions, not keys.** Every value is read as its position in its own catalog: a member's state,
action, outcome and fact as a `Nat`, a composed state as one position per member, a composed action
as its participants' `(member, action)` positions, and a composed outcome or fact as its
`(member, position)`. The kernel decides the check by unfolding it, and a `Nat` comparison is one
accelerated step where a key comparison is a walk over characters. Nothing is sorted inside the
check either: results are compared as sets, and the states a start reaches are found by a
breadth-first walk with the catalog's size as its fuel, both structural recursions the kernel
unfolds.

**What the composition function is.** `stepsFrom` is the semantics `Umpire.Command.Compose.walk`
implements: a composed action is enabled at a composed state where every participant's member has
a row from its own component; its results are every combination of one result per participant,
the first participant's outcome reported, each participant's component moved, and every
participant's facts in member order. The theorems are about that function and the literal, so a
walk that found a row the members do not authorize, missed a row they do, or kept a state no start
reaches is refused by the kernel rather than trusted.
-/

namespace Umpire.Command.Compose

open Umpire

/-! ### Tables by position -/

/-- One result of a member's row: the outcome's, the state's and each fact's catalog position. -/
structure IndexedStep where
  outcome : Nat
  state : Nat
  facts : List Nat
  deriving DecidableEq, Repr

/-- One member's table by catalog positions: the states it starts in, and for each state, at its
position, the actions it has a row for and their results. -/
structure IndexedMember where
  starts : List Nat
  rows : List (List (Nat × List IndexedStep))
  deriving DecidableEq, Repr

/-- One composed result: the reporting member and its outcome's position, each member's state
position, and each fact as its member and position. -/
structure ComposedIndexedStep where
  outcome : Nat × Nat
  state : List Nat
  facts : List (Nat × Nat)
  deriving DecidableEq, Repr

/-- One composed row: the source state's member positions, the action's participants as (member,
action position), the first reporting, and every result. -/
structure ComposedIndexedRow where
  source : List Nat
  participants : List (Nat × Nat)
  results : List ComposedIndexedStep
  deriving DecidableEq, Repr

/-- A composed literal by positions: its start states, state catalog, action catalog and rows. -/
structure IndexedLiteral where
  starts : List (List Nat)
  states : List (List Nat)
  actions : List (List (Nat × Nat))
  rows : List ComposedIndexedRow
  deriving DecidableEq, Repr

/-- A value's position in its catalog, or the catalog's length for one the catalog does not hold,
so that a value outside the catalog matches no row. -/
def position [BEq α] (catalog : FiniteCatalog α) (value : α) : Nat :=
  (catalog.findIdx? (·.value == value)).getD catalog.length

/-- A declared Model's table by positions, its rows grouped by source state. -/
def IndexedMember.ofModel [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    (model : DeclaredModel Setup State Action Outcome Fact) : IndexedMember :=
  let table := model.table
  { starts := model.initial.map (position table.states)
    rows := table.states.map fun entry =>
      (table.transitions.filter (·.source == entry.value)).map fun row =>
        (position table.actions row.action, row.results.map fun step =>
          { outcome := position table.outcomes step.outcome
            state := position table.states step.state
            facts := step.facts.map (position table.facts) }) }

/-- How a composed Model's values read as member positions. `compose` generates one per
composition from the members it resolved: a state's fields as their positions in the members'
state catalogs, a member's own action as that member and the action's position, a synchronized
action as each participant's, and an outcome or fact as its member and position. -/
structure IndexedView (State Action Outcome Fact : Type) where
  state : State → List Nat
  action : Action → List (Nat × Nat)
  outcome : Outcome → Nat × Nat
  fact : Fact → Nat × Nat

/-- A composed Model's literal table by positions, read through its view. -/
def IndexedLiteral.ofModel [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    (model : DeclaredModel Setup State Action Outcome Fact)
    (view : IndexedView State Action Outcome Fact) : IndexedLiteral :=
  { starts := model.initial.map view.state
    states := model.table.states.map (view.state ·.value)
    actions := model.table.actions.map (view.action ·.value)
    rows := model.table.transitions.map fun row =>
      { source := view.state row.source
        participants := view.action row.action
        results := row.results.map fun step =>
          { outcome := view.outcome step.outcome
            state := view.state step.state
            facts := step.facts.map view.fact } } }

/-! ### The composition function -/

/-- The results a member's table has for one action at one state, or `none` where it has no
row. -/
def IndexedMember.results (member : IndexedMember) (state action : Nat) :
    Option (List IndexedStep) :=
  (member.rows.getD state []).lookup action

/-- Each participant's results, or `none` where one has no row from its member's component. -/
def resolve (members : List IndexedMember) (source : List Nat) :
    List (Nat × Nat) → Option (List (Nat × List IndexedStep))
  | [] => some []
  | (slot, action) :: rest => do
      let member ← members[slot]?
      let component ← source[slot]?
      let results ← member.results component action
      let others ← resolve members source rest
      pure ((slot, results) :: others)

/-- Every combination of one result per participant, the first participant varying slowest. -/
def combinations : List (Nat × List IndexedStep) → List (List (Nat × IndexedStep))
  | [] => [[]]
  | (slot, results) :: rest =>
      results.flatMap fun result => (combinations rest).map ((slot, result) :: ·)

/-- One composed result of a combination: the first participant reports the outcome, every
participant's component moves, and the facts are the participants' in member order, `count`
members in all. -/
def composedStep (source : List Nat) (count : Nat) :
    List (Nat × IndexedStep) → Option ComposedIndexedStep
  | [] => none
  | (reporter, reported) :: rest => some {
      outcome := (reporter, reported.outcome)
      state := ((reporter, reported) :: rest).foldl
        (fun state (slot, result) => state.set slot result.state) source
      facts := (List.range count).flatMap fun slot =>
        match ((reporter, reported) :: rest).lookup slot with
        | some result => result.facts.map ((slot, ·))
        | none => [] }

/-- The composition function: the composed steps an action's participants take from a composed
state, or `none` where a participant has no row there. -/
def stepsFrom (members : List IndexedMember) (source : List Nat)
    (participants : List (Nat × Nat)) : Option (List ComposedIndexedStep) :=
  (resolve members source participants).map fun parts =>
    (combinations parts).filterMap (composedStep source members.length)

/-! ### The decided check -/

/-- Whether two result lists hold the same results. -/
def sameResults (left right : List ComposedIndexedStep) : Bool :=
  left.all (fun result => decide (result ∈ right)) &&
    right.all (fun result => decide (result ∈ left))

/-- Whether a row holds what the composition function says at its source. -/
def rowAgrees (members : List IndexedMember) (row : ComposedIndexedRow) : Bool :=
  match stepsFrom members row.source row.participants with
  | some results => sameResults results row.results
  | none => false

/-- Whether a composed state holds one start state per member. -/
def startOfMembers : List IndexedMember → List Nat → Bool
  | [], [] => true
  | member :: members, component :: rest =>
      decide (component ∈ member.starts) && startOfMembers members rest
  | _, _ => false

/-- The rows from one source. -/
def rowsFrom (rows : List ComposedIndexedRow) (source : List Nat) : List ComposedIndexedRow :=
  rows.filter fun row => decide (row.source = source)

/-- Whether every action the composition function enables at a source has a row there. -/
def sourceComplete (members : List IndexedMember) (actions : List (List (Nat × Nat)))
    (rows : List ComposedIndexedRow) (source : List Nat) : Bool :=
  let own := rowsFrom rows source
  actions.all fun participants =>
    (stepsFrom members source participants).isNone ||
      own.any fun row => decide (row.participants = participants)

/-- The states the rows step to from one source. -/
def successors (rows : List ComposedIndexedRow) (source : List Nat) : List (List Nat) :=
  (rowsFrom rows source).flatMap fun row => row.results.map (·.state)

/-- Add every found state that `seen` does not hold to both `seen` and `new`. -/
def absorb : List (List Nat) → List (List Nat) × List (List Nat) →
    List (List Nat) × List (List Nat)
  | [], found => found
  | state :: rest, (seen, new) =>
      absorb rest (if decide (state ∈ seen) then (seen, new) else (state :: seen, state :: new))

/-- The states `fuel` breadth-first rounds reach from `frontier` beyond `seen`. -/
def reach (rows : List ComposedIndexedRow) :
    Nat → List (List Nat) → List (List Nat) → List (List Nat)
  | 0, _, seen => seen
  | fuel + 1, frontier, seen =>
      let found := absorb (frontier.flatMap (successors rows)) (seen, [])
      reach rows fuel found.2 found.1

/-- Whether a composed literal is the composition of its members over the states its starts
reach: every start holds one start per member and is a catalog state; every row is at a catalog
state, by a catalog action, holds what the composition function says there, and steps to catalog
states; every catalog state has a row for each catalog action the composition function enables
there; and every catalog state is reached from a start. -/
def composedTableAgrees (members : List IndexedMember) (literal : IndexedLiteral) : Bool :=
  literal.starts.all (fun start =>
    startOfMembers members start && decide (start ∈ literal.states)) &&
  literal.rows.all (fun row =>
    decide (row.source ∈ literal.states) && decide (row.participants ∈ literal.actions) &&
      rowAgrees members row &&
      row.results.all fun result => decide (result.state ∈ literal.states)) &&
  literal.states.all (sourceComplete members literal.actions literal.rows) &&
  literal.states.all fun state =>
    decide (state ∈ reach literal.rows (literal.states.length + 1) literal.starts literal.starts)

/-! ### What a `true` proves -/

/-- A composed state the composition function reaches from a start through the catalog's
actions. -/
inductive Reachable (members : List IndexedMember) (actions : List (List (Nat × Nat)))
    (starts : List (List Nat)) : List Nat → Prop
  | start {state : List Nat} (member : state ∈ starts) : Reachable members actions starts state
  | step {source : List Nat} {participants : List (Nat × Nat)}
      {results : List ComposedIndexedStep} {result : ComposedIndexedStep}
      (reached : Reachable members actions starts source) (action : participants ∈ actions)
      (composed : stepsFrom members source participants = some results)
      (member : result ∈ results) : Reachable members actions starts result.state

/-- What the decided check says: the literal's starts are the members', its rows are the
composition function's at reached states, and the composition function's rows at reached states
are its. -/
structure ComposedAgreement (members : List IndexedMember) (literal : IndexedLiteral) : Prop where
  /-- Every start holds one start state per member. -/
  starts : ∀ start ∈ literal.starts, start.length = members.length ∧
    ∀ (slot : Nat) (member : IndexedMember) (component : Nat),
      members[slot]? = some member → start[slot]? = some component → component ∈ member.starts
  /-- Every row is at a reached state, by a catalog action, and holds the results the composition
  function gives there. -/
  sound : ∀ row ∈ literal.rows,
    Reachable members literal.actions literal.starts row.source ∧
      row.participants ∈ literal.actions ∧
      ∃ results, stepsFrom members row.source row.participants = some results ∧
        ∀ result, result ∈ results ↔ result ∈ row.results
  /-- Every catalog action the composition function enables at a reached state has a row there
  with its results. -/
  complete : ∀ source, Reachable members literal.actions literal.starts source →
    ∀ participants ∈ literal.actions, ∀ results,
      stepsFrom members source participants = some results →
        ∃ row ∈ literal.rows, row.source = source ∧ row.participants = participants ∧
          ∀ result, result ∈ results ↔ result ∈ row.results

theorem sameResults_iff {left right : List ComposedIndexedStep} :
    sameResults left right = true ↔ ∀ result, result ∈ left ↔ result ∈ right := by
  simp only [sameResults, Bool.and_eq_true, List.all_eq_true, decide_eq_true_eq]
  constructor
  · rintro ⟨leftIn, rightIn⟩ result
    exact ⟨leftIn result, rightIn result⟩
  · intro same
    exact ⟨fun result mem => (same result).1 mem, fun result mem => (same result).2 mem⟩

theorem rowAgrees_iff {members : List IndexedMember} {row : ComposedIndexedRow} :
    rowAgrees members row = true ↔
      ∃ results, stepsFrom members row.source row.participants = some results ∧
        ∀ result, result ∈ results ↔ result ∈ row.results := by
  unfold rowAgrees
  cases stepsFrom members row.source row.participants with
  | none => simp
  | some results => simp [sameResults_iff]

theorem startOfMembers_iff : ∀ {members : List IndexedMember} {start : List Nat},
    startOfMembers members start = true ↔ start.length = members.length ∧
      ∀ (slot : Nat) (member : IndexedMember) (component : Nat),
        members[slot]? = some member → start[slot]? = some component → component ∈ member.starts
  | [], [] => by simp [startOfMembers]
  | [], _ :: _ => by simp [startOfMembers]
  | _ :: _, [] => by simp [startOfMembers]
  | member :: members, component :: rest => by
      have tail := @startOfMembers_iff members rest
      simp only [startOfMembers, Bool.and_eq_true, decide_eq_true_eq, tail, List.length_cons,
        Nat.add_right_cancel_iff]
      constructor
      · rintro ⟨here, length, all⟩
        refine ⟨length, fun slot member' component' memberAt componentAt => ?_⟩
        cases slot with
        | zero =>
            simp only [List.getElem?_cons_zero, Option.some.injEq] at memberAt componentAt
            subst memberAt componentAt
            exact here
        | succ slot =>
            simp only [List.getElem?_cons_succ] at memberAt componentAt
            exact all slot member' component' memberAt componentAt
      · rintro ⟨length, all⟩
        exact ⟨all 0 member component (by simp) (by simp), length,
          fun slot member' component' memberAt componentAt =>
            all (slot + 1) member' component' (by simpa) (by simpa)⟩

theorem sourceComplete_spec {members : List IndexedMember} {actions : List (List (Nat × Nat))}
    {rows : List ComposedIndexedRow} {source : List Nat}
    (complete : sourceComplete members actions rows source = true) :
    ∀ participants ∈ actions, ∀ results, stepsFrom members source participants = some results →
      ∃ row ∈ rows, row.source = source ∧ row.participants = participants := by
  simp only [sourceComplete, rowsFrom, List.all_eq_true, Bool.or_eq_true, List.any_eq_true,
    List.mem_filter, decide_eq_true_eq] at complete
  intro participants mem results composed
  rcases complete participants mem with none | ⟨row, ⟨rowMem, sourceEq⟩, participantsEq⟩
  · simp [composed] at none
  · exact ⟨row, rowMem, sourceEq, participantsEq⟩

theorem mem_absorb {R : List Nat → Prop} :
    ∀ (found seen new : List (List Nat)), (∀ state ∈ found, R state) →
      (∀ state ∈ seen, R state) → (∀ state ∈ new, R state) →
      (∀ state ∈ (absorb found (seen, new)).1, R state) ∧
        ∀ state ∈ (absorb found (seen, new)).2, R state
  | [], _, _, _, seenR, newR => ⟨seenR, newR⟩
  | state :: rest, seen, new, foundR, seenR, newR => by
      have restR : ∀ state ∈ rest, R state := fun state mem =>
        foundR state (List.mem_cons_of_mem _ mem)
      have stateR : R state := foundR state List.mem_cons_self
      simp only [absorb]
      split
      · exact mem_absorb rest seen new restR seenR newR
      · refine mem_absorb rest _ _ restR ?_ ?_ <;> intro other mem <;>
          rcases List.mem_cons.1 mem with rfl | mem
        · exact stateR
        · exact seenR other mem
        · exact stateR
        · exact newR other mem

theorem mem_reach {rows : List ComposedIndexedRow} {R : List Nat → Prop}
    (closed : ∀ source, R source → ∀ state ∈ successors rows source, R state) :
    ∀ (fuel : Nat) (frontier seen : List (List Nat)), (∀ state ∈ frontier, R state) →
      (∀ state ∈ seen, R state) → ∀ state ∈ reach rows fuel frontier seen, R state
  | 0, _, _, _, seenR => seenR
  | fuel + 1, frontier, seen, frontierR, seenR => by
      have foundR : ∀ state ∈ frontier.flatMap (successors rows), R state := by
        intro state mem
        obtain ⟨source, sourceMem, mem⟩ := List.mem_flatMap.1 mem
        exact closed source (frontierR source sourceMem) state mem
      obtain ⟨seenR', newR⟩ := mem_absorb (R := R) _ seen [] foundR seenR (by simp)
      simp only [reach]
      exact mem_reach closed fuel _ _ newR seenR'

/-- The decided check is the agreement: what `composedTableAgrees` evaluates to `true` on holds. -/
theorem ComposedAgreement.ofChecked {members : List IndexedMember} {literal : IndexedLiteral}
    (checked : composedTableAgrees members literal = true) : ComposedAgreement members literal := by
  simp only [composedTableAgrees, Bool.and_eq_true, List.all_eq_true, decide_eq_true_eq] at checked
  obtain ⟨⟨⟨starts, rows⟩, complete⟩, reached⟩ := checked
  have rowSays : ∀ row ∈ literal.rows, row.source ∈ literal.states ∧
      row.participants ∈ literal.actions ∧
      (∃ results, stepsFrom members row.source row.participants = some results ∧
        ∀ result, result ∈ results ↔ result ∈ row.results) ∧
      ∀ result ∈ row.results, result.state ∈ literal.states := by
    intro row mem
    obtain ⟨⟨⟨source, action⟩, agrees⟩, closed⟩ := rows row mem
    exact ⟨source, action, rowAgrees_iff.1 agrees, closed⟩
  have edge : ∀ source, Reachable members literal.actions literal.starts source →
      ∀ state ∈ successors literal.rows source,
        Reachable members literal.actions literal.starts state := by
    intro source reachedSource state mem
    simp only [successors, rowsFrom, List.mem_flatMap, List.mem_filter, List.mem_map,
      decide_eq_true_eq] at mem
    obtain ⟨row, ⟨rowMem, sourceEq⟩, result, resultMem, stateEq⟩ := mem
    obtain ⟨_, action, ⟨results, composed, same⟩, _⟩ := rowSays row rowMem
    subst stateEq
    rw [← sourceEq] at reachedSource
    exact Reachable.step reachedSource action composed ((same result).2 resultMem)
  have statesReachable : ∀ state ∈ literal.states,
      Reachable members literal.actions literal.starts state := fun state mem =>
    mem_reach edge _ literal.starts literal.starts (fun _ mem => .start mem)
      (fun _ mem => .start mem) state (reached state mem)
  have inStates : ∀ state, Reachable members literal.actions literal.starts state →
      state ∈ literal.states := by
    intro state reachedState
    induction reachedState with
    | start mem => exact (starts _ mem).2
    | step _ action composed mem ih =>
        obtain ⟨row, rowMem, sourceEq, participantsEq⟩ :=
          sourceComplete_spec (complete _ ih) _ action _ composed
        obtain ⟨_, _, ⟨results', composed', same⟩, closed⟩ := rowSays row rowMem
        rw [sourceEq, participantsEq, composed] at composed'
        obtain rfl := Option.some.inj composed'
        exact closed _ ((same _).1 mem)
  refine ⟨fun start mem => startOfMembers_iff.1 (starts start mem).1, fun row mem => ?_,
    fun source reachedSource participants action results composed => ?_⟩
  · obtain ⟨sourceMem, action, agrees, _⟩ := rowSays row mem
    exact ⟨statesReachable row.source sourceMem, action, agrees⟩
  · obtain ⟨row, rowMem, sourceEq, participantsEq⟩ :=
      sourceComplete_spec (complete source (inStates source reachedSource)) participants action
        results composed
    obtain ⟨_, _, ⟨results', composed', same⟩, _⟩ := rowSays row rowMem
    rw [sourceEq, participantsEq, composed] at composed'
    obtain rfl := Option.some.inj composed'
    exact ⟨row, rowMem, sourceEq, participantsEq, same⟩

end Umpire.Command.Compose
