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

**Read once, decided over literals.** Reading a table by position is a scan of the catalog per
value, and a check over the flat literal would scan every row for each state it visits. So
`compose` reads the members' tables, the candidates and the composed literal to position literals
once, at elaboration, groups the literal by state, and the kernel decides five things apart: that
each reading equals its literal, that the grouping flattens to the literal, and that
`composedTableAgrees` holds of the grouping. `ComposedAgreement.ofLiterals` joins them, so the
theorem is still about the tables and never about the literals alone. The readers destructure the
values they read, so that a position scan is over the value's own literal and the kernel's cache
serves every later occurrence of the same value.

**What the composition function is.** `stepsFrom` is the semantics `Umpire.Command.Compose.walk`
implements: a composed action is enabled at a composed state where every participant's member has
a row from its own component; its results are every combination of one result per participant,
the first participant's outcome reported, each participant's component moved, and every
participant's facts in member order. The actions it is asked about are the composition's
candidates, computed from the generated Action domain rather than read off the literal: every
member's own action a `sync:` line does not name, and every class of every synchronized action.
The theorems are about that function, those candidates and the literal, so a walk that found a row
the members do not authorize, missed a row they do, dropped an action some reachable state
enables, or kept a state no start reaches is refused by the kernel rather than trusted.
-/

namespace Umpire.Command.Compose

open Umpire

/-! ### Tables by position -/

/-- One result of a member's row: the outcome's, the state's and each fact's catalog position. -/
structure IndexedStep where
  outcome : Nat
  state : Nat
  facts : List Nat
  deriving DecidableEq, Repr, Lean.ToExpr

/-- One member's table by catalog positions: the states it starts in, and for each state, at its
position, the actions it has a row for and their results. -/
structure IndexedMember where
  starts : List Nat
  rows : List (List (Nat × List IndexedStep))
  deriving DecidableEq, Repr, Lean.ToExpr

/-- One composed result: the reporting member and its outcome's position, each member's state
position, and each fact as its member and position. -/
structure ComposedIndexedStep where
  outcome : Nat × Nat
  state : List Nat
  facts : List (Nat × Nat)
  deriving DecidableEq, Repr, Lean.ToExpr

/-- One composed row: the source state's member positions, the action's participants as (member,
action position), the first reporting, and every result. -/
structure ComposedIndexedRow where
  source : List Nat
  participants : List (Nat × Nat)
  results : List ComposedIndexedStep
  deriving DecidableEq, Repr, Lean.ToExpr

/-- A composed literal by positions: its start states, state catalog, action catalog and rows. -/
structure IndexedLiteral where
  starts : List (List Nat)
  states : List (List Nat)
  actions : List (List (Nat × Nat))
  rows : List ComposedIndexedRow
  deriving DecidableEq, Repr, Lean.ToExpr

/-- A value's position in its catalog, or the catalog's length for one the catalog does not hold,
so that a value outside the catalog matches no row. -/
def position [BEq α] (catalog : FiniteCatalog α) (value : α) : Nat :=
  (catalog.findIdx? (·.value == value)).getD catalog.length

/-- A declared Model's table by positions, its rows grouped by source state: one walk over the
transitions places each row, in table order, under its source's position. -/
def IndexedMember.ofModel [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    (model : DeclaredModel Setup State Action Outcome Fact) : IndexedMember :=
  let table := model.table
  { starts := model.initial.map (position table.states)
    rows := table.transitions.foldl (init := table.states.map fun _ => [])
      fun rows ⟨_, source, action, results⟩ =>
        rows.modify (position table.states source) fun own => own ++
          [(position table.actions action, results.map fun ⟨outcome, state, facts⟩ =>
            { outcome := position table.outcomes outcome
              state := position table.states state
              facts := facts.map (position table.facts) })] }

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
    states := model.table.states.map fun ⟨value, _⟩ => view.state value
    actions := model.table.actions.map fun ⟨value, _⟩ => view.action value
    rows := model.table.transitions.map fun ⟨_, source, action, results⟩ =>
      { source := view.state source
        participants := view.action action
        results := results.map fun ⟨outcome, state, facts⟩ =>
          { outcome := view.outcome outcome
            state := view.state state
            facts := facts.map view.fact } } }

/-- The composition's candidate actions, from every action of the generated domain read through
the view: a synchronized action is every candidate with more than one participant, and a member's
own action is a candidate unless a synchronized one names it, since that member's action is then
one step with the others' rather than a step of its own. A `sync:` line with one participant reads
as that participant's own action, which is the same candidate either way. -/
def candidatesOf (actions : List (List (Nat × Nat))) : List (List (Nat × Nat)) :=
  actions.filter fun candidate =>
    match candidate with
    | [participant] => !actions.any fun other => other.length > 1 && other.contains participant
    | _ => true

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

/-! ### The literal grouped by state

The check indexes the literal where a flat one would be scanned. `GroupedLiteral` holds the same
table with each row under its source state's catalog position and each start and result state as a
catalog position, so a state's rows, a start and a result's state are each one walk of a position
away rather than a search over every row or state, and `GroupedLiteral.flatten` spells it back as
the flat literal the theorem is about. `compose` groups the flat literal it read and the kernel
decides that the grouping flattens to it, so the grouping is checked rather than trusted. -/

/-- One result of a grouped row: the reporting member and its outcome's position, the catalog
position of the composed state it steps to, and each fact as its member and position. -/
structure GroupedStep where
  outcome : Nat × Nat
  state : Nat
  facts : List (Nat × Nat)
  deriving DecidableEq, Repr, Lean.ToExpr

/-- One row of a grouped literal, its source the group it is in: the action's participants and
every result. -/
structure GroupedRow where
  participants : List (Nat × Nat)
  results : List GroupedStep
  deriving DecidableEq, Repr, Lean.ToExpr

/-- A composed literal by catalog position: the starts as positions, each catalog state's member
positions, the action catalog, and at each state's position the rows from that state. -/
structure GroupedLiteral where
  starts : List Nat
  states : List (List Nat)
  actions : List (List (Nat × Nat))
  rows : List (List GroupedRow)
  deriving DecidableEq, Repr, Lean.ToExpr

/-- The member positions at a catalog position; beyond the catalog, a state of no member's width,
which no start holds and the check lets no row step to. -/
def GroupedLiteral.stateAt (grouped : GroupedLiteral) (at? : Nat) : List Nat :=
  grouped.states.getD at? []

/-- The rows at a catalog position; none beyond the catalog. -/
def GroupedLiteral.rowsAt (grouped : GroupedLiteral) (at? : Nat) : List GroupedRow :=
  grouped.rows.getD at? []

/-- A grouped result as a composed one, its state resolved through the catalog. -/
def GroupedStep.flatten (grouped : GroupedLiteral) (step : GroupedStep) : ComposedIndexedStep :=
  { outcome := step.outcome, state := grouped.stateAt step.state, facts := step.facts }

/-- The flat literal a grouped one spells, every position resolved through its state catalog. -/
def GroupedLiteral.flatten (grouped : GroupedLiteral) : IndexedLiteral :=
  { starts := grouped.starts.map grouped.stateAt
    states := grouped.states
    actions := grouped.actions
    rows := (List.range grouped.states.length).flatMap fun at? =>
      (grouped.rowsAt at?).map fun row =>
        { source := grouped.stateAt at?
          participants := row.participants
          results := row.results.map (GroupedStep.flatten grouped) } }

/-- A flat literal grouped by state: each start and result state as its position in the state
catalog, or the catalog's length for one outside it, and each row under its source's position.
What `compose` hands the check; that its `flatten` is the flat literal is decided, not assumed. -/
def GroupedLiteral.ofFlat (literal : IndexedLiteral) : GroupedLiteral :=
  let positionOf := fun (state : List Nat) =>
    (literal.states.findIdx? (· == state)).getD literal.states.length
  { starts := literal.starts.map positionOf
    states := literal.states
    actions := literal.actions
    rows := literal.states.map fun state =>
      (literal.rows.filter (·.source == state)).map fun row =>
        { participants := row.participants
          results := row.results.map fun result =>
            { outcome := result.outcome, state := positionOf result.state, facts := result.facts } } }

/-! ### The decided check -/

/-- Whether two result lists hold the same results. -/
def sameResults (left right : List ComposedIndexedStep) : Bool :=
  left.all (fun result => decide (result ∈ right)) &&
    right.all (fun result => decide (result ∈ left))

/-- Whether a row at a source holds what the composition function says there. -/
def rowAgrees (members : List IndexedMember) (grouped : GroupedLiteral) (source : List Nat)
    (row : GroupedRow) : Bool :=
  match stepsFrom members source row.participants with
  | some results => sameResults results (row.results.map (GroupedStep.flatten grouped))
  | none => false

/-- Whether a composed state holds one start state per member. -/
def startOfMembers : List IndexedMember → List Nat → Bool
  | [], [] => true
  | member :: members, component :: rest =>
      decide (component ∈ member.starts) && startOfMembers members rest
  | _, _ => false

/-- Whether every candidate the composition function enables at a source has a row among the
source's own. A row found first spares the composition function's step. -/
def sourceComplete (members : List IndexedMember) (candidates : List (List (Nat × Nat)))
    (own : List GroupedRow) (source : List Nat) : Bool :=
  candidates.all fun participants =>
    own.any (fun row => decide (row.participants = participants)) ||
      (stepsFrom members source participants).isNone

/-- The positions the rows at a position step to. -/
def successors (grouped : GroupedLiteral) (at? : Nat) : List Nat :=
  (grouped.rowsAt at?).flatMap fun row => row.results.map (·.state)

/-- Add every found position that `seen` does not hold to both `seen` and `new`. -/
def absorb : List Nat → List Nat × List Nat → List Nat × List Nat
  | [], found => found
  | at? :: rest, (seen, new) =>
      absorb rest (if decide (at? ∈ seen) then (seen, new) else (at? :: seen, at? :: new))

/-- The positions `fuel` breadth-first rounds reach from `frontier` beyond `seen`. -/
def reach (grouped : GroupedLiteral) : Nat → List Nat → List Nat → List Nat
  | 0, _, seen => seen
  | fuel + 1, frontier, seen =>
      let found := absorb (frontier.flatMap (successors grouped)) (seen, [])
      reach grouped fuel found.2 found.1

/-- Whether a grouped literal is the composition of its members over the candidates and the states
its starts reach: every start is a catalog position holding one start per member; every catalog
action is a candidate; at every catalog position, every row is by a catalog action, holds what the
composition function says there and steps to catalog positions, and every candidate the composition
function enables there has a row; and every catalog position is reached from a start. -/
def composedTableAgrees (members : List IndexedMember) (candidates : List (List (Nat × Nat)))
    (grouped : GroupedLiteral) : Bool :=
  let width := grouped.states.length
  grouped.starts.all (fun start =>
    decide (start < width) && startOfMembers members (grouped.stateAt start)) &&
  grouped.actions.all (fun action => decide (action ∈ candidates)) &&
  (List.range width).all (fun at? =>
    (grouped.rowsAt at?).all (fun row =>
      decide (row.participants ∈ grouped.actions) &&
        rowAgrees members grouped (grouped.stateAt at?) row &&
        row.results.all fun result => decide (result.state < width)) &&
      sourceComplete members candidates (grouped.rowsAt at?) (grouped.stateAt at?)) &&
  (List.range width).all fun at? =>
    decide (at? ∈ reach grouped (width + 1) grouped.starts grouped.starts)

/-! ### What a `true` proves -/

/-- A composed state the composition function reaches from a start through the candidates. -/
inductive Reachable (members : List IndexedMember) (candidates : List (List (Nat × Nat)))
    (starts : List (List Nat)) : List Nat → Prop
  | start {state : List Nat} (member : state ∈ starts) : Reachable members candidates starts state
  | step {source : List Nat} {participants : List (Nat × Nat)}
      {results : List ComposedIndexedStep} {result : ComposedIndexedStep}
      (reached : Reachable members candidates starts source) (action : participants ∈ candidates)
      (composed : stepsFrom members source participants = some results)
      (member : result ∈ results) : Reachable members candidates starts result.state

/-- What the decided check says: the literal's starts are the members', its actions are
candidates, its rows are the composition function's at reached states, and the composition
function's rows at reached states are its. -/
structure ComposedAgreement (members : List IndexedMember) (candidates : List (List (Nat × Nat)))
    (literal : IndexedLiteral) : Prop where
  /-- Every start holds one start state per member. -/
  starts : ∀ start ∈ literal.starts, start.length = members.length ∧
    ∀ (slot : Nat) (member : IndexedMember) (component : Nat),
      members[slot]? = some member → start[slot]? = some component → component ∈ member.starts
  /-- Every catalog action is a candidate. -/
  catalog : ∀ action ∈ literal.actions, action ∈ candidates
  /-- Every row is at a reached state, by a catalog action, and holds the results the composition
  function gives there. -/
  sound : ∀ row ∈ literal.rows,
    Reachable members candidates literal.starts row.source ∧
      row.participants ∈ literal.actions ∧
      ∃ results, stepsFrom members row.source row.participants = some results ∧
        ∀ result, result ∈ results ↔ result ∈ row.results
  /-- Every candidate the composition function enables at a reached state has a row there with
  its results. -/
  complete : ∀ source, Reachable members candidates literal.starts source →
    ∀ participants ∈ candidates, ∀ results,
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

theorem rowAgrees_iff {members : List IndexedMember} {grouped : GroupedLiteral}
    {source : List Nat} {row : GroupedRow} :
    rowAgrees members grouped source row = true ↔
      ∃ results, stepsFrom members source row.participants = some results ∧
        ∀ result, result ∈ results ↔ result ∈ row.results.map (GroupedStep.flatten grouped) := by
  unfold rowAgrees
  cases stepsFrom members source row.participants with
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

theorem sourceComplete_spec {members : List IndexedMember} {candidates : List (List (Nat × Nat))}
    {own : List GroupedRow} {source : List Nat}
    (complete : sourceComplete members candidates own source = true) :
    ∀ participants ∈ candidates, ∀ results, stepsFrom members source participants = some results →
      ∃ row ∈ own, row.participants = participants := by
  simp only [sourceComplete, List.all_eq_true, Bool.or_eq_true, List.any_eq_true,
    decide_eq_true_eq] at complete
  intro participants mem results composed
  rcases complete participants mem with ⟨row, rowMem, participantsEq⟩ | none
  · exact ⟨row, rowMem, participantsEq⟩
  · simp [composed] at none

theorem mem_absorb {R : Nat → Prop} :
    ∀ (found seen new : List Nat), (∀ at? ∈ found, R at?) →
      (∀ at? ∈ seen, R at?) → (∀ at? ∈ new, R at?) →
      (∀ at? ∈ (absorb found (seen, new)).1, R at?) ∧
        ∀ at? ∈ (absorb found (seen, new)).2, R at?
  | [], _, _, _, seenR, newR => ⟨seenR, newR⟩
  | at? :: rest, seen, new, foundR, seenR, newR => by
      have restR : ∀ at? ∈ rest, R at? := fun at? mem =>
        foundR at? (List.mem_cons_of_mem _ mem)
      have atR : R at? := foundR at? List.mem_cons_self
      simp only [absorb]
      split
      · exact mem_absorb rest seen new restR seenR newR
      · refine mem_absorb rest _ _ restR ?_ ?_ <;> intro other mem <;>
          rcases List.mem_cons.1 mem with rfl | mem
        · exact atR
        · exact seenR other mem
        · exact atR
        · exact newR other mem

theorem mem_reach {grouped : GroupedLiteral} {R : Nat → Prop}
    (closed : ∀ at?, R at? → ∀ next ∈ successors grouped at?, R next) :
    ∀ (fuel : Nat) (frontier seen : List Nat), (∀ at? ∈ frontier, R at?) →
      (∀ at? ∈ seen, R at?) → ∀ at? ∈ reach grouped fuel frontier seen, R at?
  | 0, _, _, _, seenR => seenR
  | fuel + 1, frontier, seen, frontierR, seenR => by
      have foundR : ∀ at? ∈ frontier.flatMap (successors grouped), R at? := by
        intro at? mem
        obtain ⟨source, sourceMem, mem⟩ := List.mem_flatMap.1 mem
        exact closed source (frontierR source sourceMem) at? mem
      obtain ⟨seenR', newR⟩ := mem_absorb (R := R) _ seen [] foundR seenR (by simp)
      simp only [reach]
      exact mem_reach closed fuel _ _ newR seenR'

/-- The flat row a grouped row at a catalog position spells. -/
def GroupedRow.flatten (grouped : GroupedLiteral) (at? : Nat) (row : GroupedRow) :
    ComposedIndexedRow :=
  { source := grouped.stateAt at?
    participants := row.participants
    results := row.results.map (GroupedStep.flatten grouped) }

theorem GroupedLiteral.mem_flatten_rows {grouped : GroupedLiteral} {at? : Nat} {row : GroupedRow}
    (lt : at? < grouped.states.length) (mem : row ∈ grouped.rowsAt at?) :
    row.flatten grouped at? ∈ grouped.flatten.rows := by
  simp only [GroupedLiteral.flatten, List.mem_flatMap, List.mem_range, List.mem_map]
  exact ⟨at?, lt, row, mem, rfl⟩

theorem GroupedLiteral.of_mem_flatten_rows {grouped : GroupedLiteral} {row : ComposedIndexedRow}
    (mem : row ∈ grouped.flatten.rows) :
    ∃ at?, at? < grouped.states.length ∧ ∃ own ∈ grouped.rowsAt at?, row = own.flatten grouped at? := by
  simp only [GroupedLiteral.flatten, List.mem_flatMap, List.mem_range, List.mem_map] at mem
  obtain ⟨at?, lt, own, ownMem, rfl⟩ := mem
  exact ⟨at?, lt, own, ownMem, rfl⟩

/-- The decided check is the agreement: what `composedTableAgrees` evaluates to `true` on holds of
the flat literal the grouped one spells. -/
theorem ComposedAgreement.ofChecked {members : List IndexedMember}
    {candidates : List (List (Nat × Nat))} {grouped : GroupedLiteral}
    (checked : composedTableAgrees members candidates grouped = true) :
    ComposedAgreement members candidates grouped.flatten := by
  simp only [composedTableAgrees, Bool.and_eq_true, List.all_eq_true, decide_eq_true_eq,
    List.mem_range] at checked
  obtain ⟨⟨⟨starts, catalog⟩, positions⟩, reached⟩ := checked
  have rowSays : ∀ at?, at? < grouped.states.length → ∀ row ∈ grouped.rowsAt at?,
      row.participants ∈ grouped.actions ∧
      (∃ results, stepsFrom members (grouped.stateAt at?) row.participants = some results ∧
        ∀ result, result ∈ results ↔ result ∈ row.results.map (GroupedStep.flatten grouped)) ∧
      ∀ result ∈ row.results, result.state < grouped.states.length := by
    intro at? lt row mem
    obtain ⟨⟨action, agrees⟩, closed⟩ := (positions at? lt).1 row mem
    exact ⟨action, rowAgrees_iff.1 agrees, closed⟩
  have complete : ∀ at?, at? < grouped.states.length →
      sourceComplete members candidates (grouped.rowsAt at?) (grouped.stateAt at?) = true :=
    fun at? lt => (positions at? lt).2
  -- A position is reached in the walk only through checked rows, which keep it in the catalog.
  have edge : ∀ at?, (at? < grouped.states.length ∧
        Reachable members candidates grouped.flatten.starts (grouped.stateAt at?)) →
      ∀ next ∈ successors grouped at?, next < grouped.states.length ∧
        Reachable members candidates grouped.flatten.starts (grouped.stateAt next) := by
    rintro at? ⟨lt, reachedAt⟩ next mem
    simp only [successors, List.mem_flatMap, List.mem_map] at mem
    obtain ⟨row, rowMem, result, resultMem, stateEq⟩ := mem
    obtain ⟨action, ⟨results, composed, same⟩, closed⟩ := rowSays at? lt row rowMem
    subst stateEq
    refine ⟨closed result resultMem, ?_⟩
    have flatMem : GroupedStep.flatten grouped result ∈ results :=
      (same _).2 (List.mem_map_of_mem resultMem)
    exact Reachable.step reachedAt (catalog _ action) composed flatMem
  have startsReach : ∀ start ∈ grouped.starts, start < grouped.states.length ∧
      Reachable members candidates grouped.flatten.starts (grouped.stateAt start) :=
    fun start mem => ⟨(starts start mem).1, .start (List.mem_map_of_mem mem)⟩
  have positionsReachable : ∀ at?, at? < grouped.states.length →
      Reachable members candidates grouped.flatten.starts (grouped.stateAt at?) := fun at? lt =>
    (mem_reach edge _ grouped.starts grouped.starts startsReach startsReach at? (reached at? lt)).2
  have inCatalog : ∀ state, Reachable members candidates grouped.flatten.starts state →
      ∃ at?, at? < grouped.states.length ∧ grouped.stateAt at? = state := by
    intro state reachedState
    induction reachedState with
    | start mem =>
        obtain ⟨start, startMem, rfl⟩ := List.mem_map.1 mem
        exact ⟨start, (starts start startMem).1, rfl⟩
    | step _ action composed mem ih =>
        obtain ⟨at?, lt, rfl⟩ := ih
        obtain ⟨row, rowMem, participantsEq⟩ :=
          sourceComplete_spec (complete at? lt) _ action _ composed
        obtain ⟨_, ⟨results', composed', same⟩, closed⟩ := rowSays at? lt row rowMem
        rw [participantsEq, composed] at composed'
        obtain rfl := Option.some.inj composed'
        obtain ⟨result, resultMem, rfl⟩ := List.mem_map.1 ((same _).1 mem)
        exact ⟨result.state, closed result resultMem, rfl⟩
  refine ⟨?_, catalog, ?_, ?_⟩
  · intro start mem
    obtain ⟨at?, atMem, rfl⟩ := List.mem_map.1 mem
    exact startOfMembers_iff.1 (starts at? atMem).2
  · intro row mem
    obtain ⟨at?, lt, own, ownMem, rfl⟩ := GroupedLiteral.of_mem_flatten_rows mem
    obtain ⟨action, agrees, _⟩ := rowSays at? lt own ownMem
    exact ⟨positionsReachable at? lt, action, agrees⟩
  · intro source reachedSource participants action results composed
    obtain ⟨at?, lt, rfl⟩ := inCatalog source reachedSource
    obtain ⟨row, rowMem, participantsEq⟩ :=
      sourceComplete_spec (complete at? lt) participants action results composed
    obtain ⟨_, ⟨results', composed', same⟩, _⟩ := rowSays at? lt row rowMem
    rw [participantsEq, composed] at composed'
    obtain rfl := Option.some.inj composed'
    exact ⟨row.flatten grouped at?, GroupedLiteral.mem_flatten_rows lt rowMem, rfl,
      participantsEq, same⟩

/-- The agreement from a check decided over literals: the members, the candidates and the literal
each equal the literal the check ran on, and the grouping flattens to it, so what the check says of
the grouping it says of them. `compose` decides the three readings, the flattening and the check as
five kernel decisions, each over the reading it is about, rather than one that rereads the tables
at every row. -/
theorem ComposedAgreement.ofLiterals {members membersLiteral : List IndexedMember}
    {candidates candidatesLiteral : List (List (Nat × Nat))}
    {literal literalLiteral : IndexedLiteral} {grouped : GroupedLiteral}
    (membersRead : members = membersLiteral) (candidatesRead : candidates = candidatesLiteral)
    (literalRead : literal = literalLiteral) (groupedRead : grouped.flatten = literalLiteral)
    (checked : composedTableAgrees membersLiteral candidatesLiteral grouped = true) :
    ComposedAgreement members candidates literal := by
  subst membersRead candidatesRead literalRead groupedRead
  exact ComposedAgreement.ofChecked checked

end Umpire.Command.Compose
