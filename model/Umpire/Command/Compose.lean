import Std.Data.HashMap
import Std.Data.HashSet
import Umpire.Command.Authoring
import Umpire.Command.Finite
import Umpire.Command.Predicate

/-!
# Composing machines into one Model

A `compose` command builds one Model from several declared machines. This module is the part of it
that decides nothing about Lean syntax: it reads each member's checked table as keys, turns the
`sync:` lines into the composed Action catalog, and walks the composed states reachable from the
composed start states. The command emits what the walk returns as a literal table.

Keys are what the walk works on, because the member tables already say everything by key and a
key is what the composed catalogs are spelled in: a member's own action is `<field>_<key>`, a
synchronized action is its `sync:` name with the classed participant's class, and a composed state
joins its members' state keys with `_` in the order the author's state structure writes its fields.
`FiniteCatalog.validKey` admits no `.`, and `instances:` already spells its product keys this way.

**Semantics.** A member's own action steps that member and leaves the others where they are. A
synchronized action is enabled only where every participant has a row from its current state, and
its results are every combination of the participants' results: the first-named participant's
outcome, each participant's state, and every participant's facts in member order. A participant
whose action carries no input takes part in every class of a classed one.

**Order.** The walk's discovery order depends on the order members and `sync:` lines are written,
so nothing it returns keeps that order: the state and action catalogs, the start states and every
row's results are sorted by the key Search admits them by, the lowered value's order key, and that
key contains the catalog key itself, so no two members of one catalog tie. A stable sort would
otherwise carry the `members:` order into the Behavior Fingerprint through the ties.
-/

namespace Umpire.Command.Compose

open Umpire

/-- One result of a member's row, by the keys its catalogs give it. -/
structure KeyStep where
  outcome : String
  state : String
  facts : List String
  deriving BEq, Repr, Inhabited

/-- One row of a member's table, by keys. -/
structure KeyRow where
  source : String
  action : String
  results : List KeyStep
  deriving BEq, Repr, Inhabited

private def keyIn [BEq α] (catalog : FiniteCatalog α) (value : α) : String :=
  ((catalog.find? (·.value == value)).map (·.key)).getD ""

/-- A declared Model's rows, read through its own catalogs. -/
def keyRows [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    (model : DeclaredModel Setup State Action Outcome Fact) : List KeyRow :=
  model.table.transitions.map fun row => {
    source := keyIn model.table.states row.source
    action := keyIn model.table.actions row.action
    results := row.results.map fun step => {
      outcome := keyIn model.table.outcomes step.outcome
      state := keyIn model.table.states step.state
      facts := step.facts.map (keyIn model.table.facts) } }

/-- One member of a composition, by keys: the field that holds it, its machine's action keys, the
names of its timers, the states it starts in, and its rows. -/
structure Member where
  field : String
  /-- The machine's state keys, which the composed state key joins with `_`. -/
  states : List String := []
  actions : List String
  /-- Each action's input domains, in order, by the action's name. A timer takes none. -/
  domains : List (String × List String) := []
  timers : List String := []
  starts : List String
  rows : List KeyRow
  deriving Repr, Inhabited

/-- One `sync:` line: its name and each participant as (member field, action name). -/
structure SyncLine where
  name : String
  participants : List (String × String)
  deriving Repr, Inhabited

/-- The action a key names: its first segment, which is the constructor of a classed action and the
whole key of a bare one. -/
def actionName (key : String) : String := ((key.splitOn "-").head?).getD key

/-- The key of a composed state: its members' state keys, in the state structure's field order. -/
def stateKey (components : List String) : String := "_".intercalate components

/-- The key of one member's own action or value in the composed catalogs. -/
def memberKey (field key : String) : String := field ++ "_" ++ key

/-- The catalog key an author's reference names: a dotted `field.action` is that member's own
action, a bare name is a `sync:` name, and a class written after it is joined the way a machine
keys its classed actions. -/
def referenceKey (components : List String) (classKeys : List String) : String :=
  "-".intercalate ("_".intercalate components :: classKeys)

/-- The owner a composition's Definition IDs hang off, under the enclosing namespace's family. A
machine's own owner is its bare name, so the prefix keeps a composition and a machine of one name
apart. -/
def owner (name : String) : String := "compose-" ++ name

/-- The most results one synchronized step may have. -/
def productBound : Nat := 16

/-- One action of the composed catalog, before the walk says whether it is ever enabled. -/
structure Candidate where
  key : String
  /-- The `sync:` line it comes from, by position; `none` for one member's own action. -/
  sync : Option Nat
  /-- Each participant as (member position, the member's action key). The first reports the
  outcome. -/
  participants : List (Nat × String)
  deriving BEq, Repr, Inhabited

/-- Why a composition was refused, with what the message needs to name the line. -/
inductive ComposeError where
  | unknownParticipant (line : Nat) (field action : String)
  | repeatedParticipant (line : Nat) (field : String)
  | timerSynchronized (line : Nat) (field timer : String)
  | inputMismatch (line : Nat) (first second : String)
  | unsynchronizedShared (action : String) (fields : List String)
  | duplicateAction (line : Nat) (key : String)
  | underscoredState (field key : String)
  | productTooLarge (line : Nat) (key : String) (count bound : Nat)
  | boundExceeded (states actions bound : Nat)
  deriving BEq, Repr, Inhabited

def ComposeError.message : ComposeError → String
  | .unknownParticipant _ field action =>
      s!"'{field}.{action}' names no action of a member; a `sync:` participant is a member field \
and an action that member's machine steps on"
  | .repeatedParticipant _ field =>
      s!"member '{field}' takes part in this `sync:` line twice; each member takes part in a \
synchronized action once"
  | .timerSynchronized _ field timer =>
      s!"'{field}.{timer}' is a timer; a timer is `system` behaviour one member fires on its own, \
so no `sync:` line names it"
  | .inputMismatch _ first second =>
      s!"'{first}' and '{second}' range over different inputs; a synchronized action takes one \
class per step, so its classed participants take the same input domains"
  | .unsynchronizedShared action fields =>
      s!"members {", ".intercalate fields} each own an action named '{action}' and no `sync:` \
line names every one of them; two members' actions of one name are synchronized explicitly"
  | .duplicateAction _ key =>
      s!"'{key}' is already the key of another action of the composition; a `sync:` name keys its \
action, so it is spelled apart from every member's `<field>_<action>` key"
  | .underscoredState field key =>
      s!"member '{field}' has a state keyed '{key}', which contains '_'; a composed state joins its \
members' keys with `_`, so a member's state key carries none"
  | .productTooLarge _ key count bound =>
      s!"'{key}' multiplies its participants' results out to {count} steps; a synchronized step \
has at most {bound} results"
  | .boundExceeded states actions bound =>
      s!"the composition reaches at least {states} states over {actions} action classes, \
{states * actions} evaluations, and the bound is {bound}; compose smaller machines"

/-- The composed Action catalog before the walk: every member's own actions that no `sync:` line
names, and one action per `sync:` line and class. Members are in the state structure's field
order. -/
def candidates (members : List Member) (syncs : List SyncLine) :
    Except ComposeError (List Candidate) := do
  -- A composed state key is read back apart only if no member's key contains the separator.
  for member in members do
    if let some key := member.states.find? (·.contains '_') then
      throw (.underscoredState member.field key)
  let position := fun (field : String) => members.findIdx? (·.field == field)
  let mut synced : List (Nat × String) := []
  let mut fromSyncs : List Candidate := []
  for (line, index) in syncs.zipIdx do
    let mut resolved : List (Nat × String × List String) := []
    for (field, action) in line.participants do
      let some slot := position field | throw (.unknownParticipant index field action)
      let member := members[slot]!
      if member.timers.contains action then throw (.timerSynchronized index field action)
      let keys := member.actions.filter (actionName · == action)
      if keys.isEmpty then throw (.unknownParticipant index field action)
      if resolved.any (·.1 == slot) then throw (.repeatedParticipant index field)
      let classes := if keys == [action] then [] else keys.map fun key =>
        (key.drop action.length).toString
      resolved := resolved ++ [(slot, action, classes)]
      synced := (slot, action) :: synced
    let classed := resolved.filter fun (_, _, classes) => !classes.isEmpty
    let spell := fun (slot : Nat) (action : String) => members[slot]!.field ++ "." ++ action
    -- Classed participants take one class per step, so they range over the same domains: the
    -- same declarations, not merely constructors spelled alike.
    let inputsOf := fun (slot : Nat) (action : String) (classes : List String) =>
      (((members[slot]!.domains.find? (·.1 == action)).map (·.2)).getD [], classes)
    if let some (firstAt, firstAction, firstClasses) := classed.head? then
      if let some (slot, action, _) := classed.find? fun (slot, action, classes) =>
          inputsOf slot action classes != inputsOf firstAt firstAction firstClasses then
        throw (.inputMismatch index (spell firstAt firstAction) (spell slot action))
    let classes := ((classed.head?).map (·.2.2)).getD [""]
    fromSyncs := fromSyncs ++ classes.map fun suffix => {
      key := line.name ++ suffix
      sync := some index
      participants := resolved.map fun (slot, action, own) =>
        (slot, if own.isEmpty then action else action ++ suffix) }
  -- Two members' actions of one name are the same step only when a line says so. A timer is keyed
  -- per member and never synchronizes, so two members' timers of one name are two timers.
  let owned := members.zipIdx.flatMap fun (member, slot) =>
    ((member.actions.map actionName).eraseDups.filter (!member.timers.contains ·)).map (·, slot)
  for name in (owned.map (·.1)).eraseDups do
    let owners := owned.filterMap fun (owner, slot) => if owner == name then some slot else none
    if owners.length > 1 && owners.any (fun slot => !synced.contains (slot, name)) then
      throw (.unsynchronizedShared name (owners.map fun slot => members[slot]!.field))
  let own := members.zipIdx.flatMap fun (member, slot) =>
    (member.actions.filter fun key => !synced.contains (slot, actionName key)).map fun key =>
      { key := memberKey member.field key, sync := none, participants := [(slot, key)] }
  let all := own ++ fromSyncs
  if let some duplicate := fromSyncs.find? fun candidate =>
      (all.filter (·.key == candidate.key)).length > 1 then
    throw (.duplicateAction (duplicate.sync.getD 0) duplicate.key)
  pure all

/-- A same-step claim written with a bare classed action is a claim about every one of its classes:
the per-class enumerations, each group's clause labels prefixed by its class key so two classes that
fix one value are two clauses. The first refusal is the claim's. -/
def acrossClasses (enumerated : List (String × EnumeratedProperty)) : EnumeratedProperty :=
  let relabel := fun (key : String) (requirement : PropertyRequirement) =>
    match requirement with
    | .stateClause label spelling => .stateClause (key ++ "-" ++ label) spelling
    | .outcomeClause label spelling => .outcomeClause (key ++ "-" ++ label) spelling
    | .factClause label spelling => .factClause (key ++ "-" ++ label) spelling
  match enumerated.findSome? (·.2.refusal) with
  | some refusal => { refusal := some refusal }
  | none => { groups := enumerated.flatMap fun (key, property) =>
      property.groups.map fun group =>
        { group with requirements := group.requirements.map (relabel key) } }

/-- One composed result: the reported outcome's key, every member's state key, and the facts'
keys. -/
structure ComposedStep where
  outcome : String
  state : List String
  facts : List String
  deriving BEq, Repr, Inhabited

/-- One composed row: the source state's members, the action's key, and every result. -/
structure ComposedRow where
  source : List String
  action : String
  results : List ComposedStep
  deriving BEq, Repr, Inhabited

/-- What the walk returns, every list in the order the command emits it. -/
structure Walked where
  states : List (List String)
  actions : List Candidate
  /-- The candidates no reachable state enables, which the catalog leaves out. -/
  dropped : List Candidate
  initial : List (List String)
  rows : List ComposedRow
  deriving Repr, Inhabited

private def orderKey (origin : Origin) (owner kind key : String) : String :=
  modelValueOrderKey (ModelValue.named (origin.ownedId kind owner key) key)

private def resultOrderKey (origin : Origin) (owner : String) (step : ComposedStep) : String :=
  let named := fun (kind key : String) => ModelValue.named (origin.ownedId kind owner key) key
  stepOrderKey ⟨named "outcome" step.outcome, named "state" (stateKey step.state),
    step.facts.map (named "fact")⟩

private def sortedBy (key : α → String) (items : List α) : List α :=
  items.mergeSort fun left right => key left ≤ key right

/-- Every combination of one result per participant, first participant varying slowest. -/
private def combinations : List (Nat × List KeyStep) → List (List (Nat × KeyStep))
  | [] => [[]]
  | (slot, results) :: rest =>
      results.flatMap fun result => (combinations rest).map ((slot, result) :: ·)

/-- The steps one candidate takes from one composed state, or `none` where a participant has no
row. -/
private def stepsFrom (members : Array Member)
    (tables : Array (Std.HashMap (String × String) (List KeyStep)))
    (source : List String) (candidate : Candidate) :
    Except ComposeError (Option (List ComposedStep)) := do
  let parts := candidate.participants.filterMap fun (slot, key) =>
    (tables[slot]!.get? (source[slot]!, key)).map (slot, ·)
  if parts.length != candidate.participants.length then return none
  let count := parts.foldl (fun product (_, results) => product * results.length) 1
  if let some line := candidate.sync then
    if count > productBound then throw (.productTooLarge line candidate.key count productBound)
  pure <| some <| (combinations parts).map fun combination =>
    let (reporter, reported) := combination.head!
    let field := fun (slot : Nat) => members[slot]!.field
    { outcome := memberKey (field reporter) reported.outcome
      state := combination.foldl (fun state (slot, result) => state.set slot result.state) source
      facts := (combination.mergeSort fun left right => left.1 ≤ right.1).flatMap
        fun (slot, result) => result.facts.map (memberKey (field slot)) }

/-- Walk the composed states reachable from the members' start states, breadth first, and return
the catalogs and rows in the order the command emits them.

The bound is the walk's own size, reachable states times candidate actions, checked as each new
state is found, so a composition too large to elaborate is refused with its counts rather than
walked to the end. -/
def walk (origin : Origin) (owner : String) (members : List Member) (candidates : List Candidate)
    (bound : Nat := enumerationBound) : Except ComposeError Walked := do
  let memberArray := members.toArray
  let tables := memberArray.map fun member =>
    member.rows.foldl (init := ({} : Std.HashMap (String × String) (List KeyStep)))
      fun table row => table.insert (row.source, row.action) row.results
  let starts := (members.foldr (init := [[]]) fun member rest =>
    member.starts.flatMap fun start => rest.map (start :: ·)).eraseDups
  let mut seen : Std.HashSet (List String) := {}
  let mut order : Array (List String) := #[]
  for start in starts do
    seen := seen.insert start
    order := order.push start
  if order.size * candidates.length > bound then
    throw (.boundExceeded order.size candidates.length bound)
  let mut found : Std.HashMap (String × String) (List ComposedStep) := {}
  let mut enabled : Std.HashSet String := {}
  let mut cursor := 0
  -- Each pass visits one state, and a state is visited once, so the bound on the states is a bound
  -- on the passes.
  for _ in [0:bound + 1] do
    let some source := order[cursor]? | break
    cursor := cursor + 1
    for candidate in candidates do
      let some steps ← stepsFrom memberArray tables source candidate | continue
      found := found.insert (stateKey source, candidate.key)
        (sortedBy (resultOrderKey origin owner) steps)
      enabled := enabled.insert candidate.key
      for step in steps do
        unless seen.contains step.state do
          seen := seen.insert step.state
          order := order.push step.state
          if order.size * candidates.length > bound then
            throw (.boundExceeded order.size candidates.length bound)
  let states := sortedBy (fun state => orderKey origin owner "state" (stateKey state)) order.toList
  let actions := sortedBy (fun candidate => orderKey origin owner "action" candidate.key)
    (candidates.filter (enabled.contains ·.key))
  pure {
    states
    actions
    dropped := candidates.filter (!enabled.contains ·.key)
    initial := sortedBy (fun state => orderKey origin owner "state" (stateKey state)) starts
    rows := states.flatMap fun source => actions.filterMap fun candidate =>
      (found.get? (stateKey source, candidate.key)).map fun results =>
        { source, action := candidate.key, results } }

end Umpire.Command.Compose
