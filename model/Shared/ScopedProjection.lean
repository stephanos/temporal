/-!
Bounded causal projection over a closed finite transition table. Identities, evidence payloads,
and model values are parameters; this module knows neither an authoring language nor a protocol.
A released step carries membership in the supplied table. Consumers supply only value rendering
and result-state accessors, never a transition oracle. An append commits all releases or none.
-/
namespace Shared.ScopedProjection

variable {Id State Action Result Field : Type}

structure Identity (Id : Type) where
  scope : List (Id × String)
  source : Id
  ordinal : Nat
  deriving BEq, DecidableEq, Repr

structure EvidenceField (Id Value : Type) where
  id : Id
  value : Option Value
  deriving BEq, DecidableEq, Repr

structure Event (Id Field : Type) where
  identity : Identity Id
  operation : String
  kind : Id
  parents : List (Identity Id) := []
  runSequences : List Nat
  fields : List Field := []
  deriving BEq, DecidableEq, Repr

structure Limits where
  events : Nat
  buffered : Nat
  keys : Nat
  support : Nat
  work : Nat
  eventSize : Nat
  deriving BEq, DecidableEq, Repr

inductive Meaning (Action Result : Type) where
  | irrelevant
  | submission (action : Action)
  | confirmed (submission : Option Action) (steps : List (Action × Result))
  deriving BEq, DecidableEq, Repr

structure Rule (Id Action Result : Type) where
  kind : Id
  fieldCount : Nat
  meaning : Meaning Action Result
  deriving BEq, DecidableEq, Repr

/-- Closed executable data; table order retains the checked producer's finite enumeration. -/
structure Plan (Id State Action Result : Type) where
  initial : State
  rules : List (Rule Id Action Result)
  transitions : List (State × Action × Result)
  limits : Limits
  deriving BEq, DecidableEq

inductive Error (Id : Type) where
  | unsupportedVersion | invalidDeclaration | unsupportedDisposition | invalidInitialState
  | wrongScope | unknownSource
  | identityConflict (identity : Identity Id)
  | wrongOperation (identity parent : Identity Id)
  | unsupportedEvidence (kind : Id)
  | unauthorizedField (field : Id)
  | missingSubmission (identity : Identity Id)
  | invalidTransition (identity : Identity Id)
  | causalCycle
  | incomparableOrder (identity previous : Identity Id)
  | eventsExhausted | bufferExhausted | keysExhausted | supportExhausted | workExhausted
  | eventSizeExhausted | invalidEvidenceSupport
  | incomplete (pending : List (Identity Id))
  | nonterminal (operations : List String)
  | closed
  deriving BEq, DecidableEq, Repr

/-- Closed field policies share the evidence admission boundary across authoring and wire consumers. -/
def validateEvidence {Id Value : Type} [BEq Id] (kind : Value → Nat)
    (size : Event Id (EvidenceField Id Value) → Nat) (limits : Limits)
    (sources : List Id) (policies : List (Id × List (Id × Nat × Nat)))
    (scope : List (Id × String)) (event : Event Id (EvidenceField Id Value)) : Except (Error Id) Unit := do
  if size event > limits.eventSize then throw .eventSizeExhausted
  if event.identity.scope != scope || event.parents.any (fun parent => parent.scope != scope) then
    throw .wrongScope
  if !sources.contains event.identity.source ||
      event.parents.any (fun parent => !sources.contains parent.source) then throw .unknownSource
  if event.operation.isEmpty then throw .wrongScope
  if event.runSequences.isEmpty || event.runSequences.contains 0 ||
      event.runSequences.eraseDups != event.runSequences then throw .invalidEvidenceSupport
  if event.identity.ordinal >= limits.events ||
      event.parents.any (fun parent => parent.ordinal >= limits.events) then throw .eventsExhausted
  let some (_, fields) := policies.find? (fun policy => policy.1 == event.kind)
    | throw (.unsupportedEvidence event.kind)
  if event.parents.eraseDups != event.parents then throw .invalidDeclaration
  if (event.fields.map (·.id)).eraseDups != event.fields.map (·.id) then throw .invalidDeclaration
  for field in event.fields do
    let some (_, expected, disposition) := fields.find? (fun entry => entry.1 == field.id)
      | throw (.unauthorizedField field.id)
    match disposition, field.value with
    | 1, some value => if kind value != expected then throw (.unauthorizedField field.id)
    | 2, none => pure ()
    | _, _ => throw (.unauthorizedField field.id)
  for (id, _, disposition) in fields do
    if disposition != 3 && !(event.fields.any fun value => value.id == id) then
      throw (.unauthorizedField id)

inductive Progress (Step : Type) where
  | pending (unused : Unit)
  | stutter
  | emitted (steps : List Step)

/-- Table membership is the only authority this generic interpreter can establish. -/
structure Step (plan : Plan Id State Action Result) where
  operation : String
  priorState : State
  action : Action
  result : Result
  directSupport : List (Identity Id)
  support : List (Identity Id)
  directRunSequences : List Nat
  runSequences : List Nat
  member : (priorState, action, result) ∈ plan.transitions

structure Run (plan : Plan Id State Action Result) (Field : Type) where
  scope : List (Id × String)
  accepted : List (Event Id Field) := []
  processed : List (Identity Id) := []
  support : List (Identity Id × List (Identity Id)) := []
  states : List (String × State) := []
  steps : List (Step plan) := []
  work : Nat := 0
  closed : Bool := false

variable [DecidableEq Id] [DecidableEq State] [DecidableEq Action] [DecidableEq Result]
variable [DecidableEq Field]
variable {plan : Plan Id State Action Result}

private def eqb [DecidableEq α] (a b : α) : Bool := decide (a = b)
private def has [DecidableEq α] (xs : List α) (x : α) : Bool := decide (x ∈ xs)
private def dedup [DecidableEq α] (xs : List α) : List α :=
  xs.foldl (fun result x => if has result x then result else result ++ [x]) []

/-- Pending identities do not denote stutters or semantic steps. -/
def Run.pending (run : Run plan Field) : List (Identity Id) :=
  run.accepted.filterMap fun event => if has run.processed event.identity then none else some event.identity

private def sourceBefore (a b : Identity Id) : Bool :=
  eqb a.source b.source && a.ordinal < b.ordinal
private def orderingEdge (before : Identity Id) (after : Event Id Field) : Bool :=
  has after.parents before || sourceBefore before after.identity

private def reaches (events : List (Event Id Field)) (before after : Identity Id) : Bool := _root_.Id.run do
  if eqb before after then return true
  let nodes := events.toArray
  let mut visited := nodes.map fun event => eqb event.identity before
  let mut frontier := [before]
  for _ in [:nodes.size] do
    let current :: rest := frontier | return false
    frontier := rest
    for index in [:nodes.size] do
      if !(visited[index]?).getD false then
        if let some candidate := nodes[index]? then
          if orderingEdge current candidate then
            if eqb candidate.identity after then return true
            visited := visited.set! index true
            frontier := frontier ++ [candidate.identity]
  return false

private def ready (events : List (Event Id Field)) (processed : List (Identity Id))
    (event : Event Id Field) : Bool :=
  event.parents.all (has processed) &&
    (event.identity.ordinal == 0 || events.any fun candidate =>
      eqb candidate.identity.source event.identity.source &&
      candidate.identity.ordinal + 1 == event.identity.ordinal && has processed candidate.identity)

private def acyclic (events : List (Event Id Field)) : Bool := _root_.Id.run do
  let mut removed : List (Identity Id) := []
  for _ in [:events.length] do
    for event in events do
      if !has removed event.identity && events.all (fun predecessor =>
          !orderingEdge predecessor.identity event || has removed predecessor.identity) then
        removed := event.identity :: removed
    if removed.length == events.length then return true
  return removed.length == events.length

private def validateGraph (events : List (Event Id Field)) : Except (Error Id) Unit := do
  for event in events do
    for parent in event.parents do
      if let some predecessor := events.find? (fun candidate => eqb candidate.identity parent) then
        if predecessor.operation != event.operation then throw (.wrongOperation event.identity parent)
  if !acyclic events then throw .causalCycle

private def release (stateOf : Result → State) (run : Run plan Field) (event : Event Id Field) :
    Except (Error Id) (Run plan Field) := do
  let some rule := plan.rules.find? (fun rule => eqb rule.kind event.kind)
    | throw (.unsupportedEvidence event.kind)
  let ancestors := event.parents.flatMap fun parent =>
    (run.support.find? fun entry => eqb entry.1 parent).map Prod.snd |>.getD []
  let support := dedup (ancestors ++ [event.identity])
  let direct := dedup (event.identity :: event.parents)
  let sequences := fun identities => dedup
    ((identities.flatMap fun identity =>
      (run.accepted.find? fun candidate => eqb candidate.identity identity).map Event.runSequences
        |>.getD []).mergeSort)
  let mut run := { run with
    processed := run.processed ++ [event.identity]
    support := run.support ++ [(event.identity, support)] }
  match rule.meaning with
  | .irrelevant | .submission _ => pure ()
  | .confirmed required outputs =>
      if let some previous := run.steps.reverse.find? (fun step => step.operation == event.operation) then
        if let some identity := previous.directSupport.head? then
          if !reaches run.accepted identity event.identity then
            throw (.incomparableOrder event.identity identity)
      if required.any (fun action => !(event.parents.any fun parent =>
          (run.accepted.find? fun candidate => eqb candidate.identity parent).any fun candidate =>
            (plan.rules.find? fun rule => eqb rule.kind candidate.kind).any fun parentRule =>
              match parentRule.meaning with
              | .submission submitted => eqb submitted action
              | _ => false)) then throw (.missingSubmission event.identity)
      for (action, result) in outputs do
        let prior := (run.states.find? fun entry => entry.1 == event.operation).map Prod.snd
          |>.getD plan.initial
        if h : (prior, action, result) ∈ plan.transitions then
          let step : Step plan := ⟨event.operation, prior, action, result, direct, support,
            sequences direct, sequences support, h⟩
          run := { run with
            states := (run.states.filter fun entry => entry.1 != event.operation) ++
              [(event.operation, stateOf result)]
            steps := run.steps ++ [step] }
        else throw (.invalidTransition event.identity)
  let retained := run.support.foldl (fun n entry => n + entry.2.length) 0 +
    run.steps.foldl (fun n step => n + step.directSupport.length + step.support.length +
      step.directRunSequences.length + step.runSequences.length) 0
  if retained > plan.limits.support then throw .supportExhausted
  return run

/-- The conservative reservation accounts for graph scans, payload comparisons and table lookups. -/
def reservation (eventSize : Event Id Field → Nat) (plan : Plan Id State Action Result)
    (events : List (Event Id Field)) : Nat :=
  let n := events.length + 1
  let outputs := plan.rules.foldl (fun n rule => n + rule.fieldCount +
    (match rule.meaning with | .confirmed _ steps => steps.length | _ => 1)) 0
  (events.foldl (fun size event => size + eventSize event) 0 + 1) *
    n * n * n * (outputs + plan.transitions.length + 1)

/-- Validated evidence admission is atomic, including all buffered releases and table checks. -/
def Run.admit (idText : Id → String) (stateOf : Result → State)
    (eventSize : Event Id Field → Nat) (run : Run plan Field) (event : Event Id Field) :
    Except (Error Id) (Run plan Field) := do
  if run.closed then throw .closed
  if let some previous := run.accepted.find? (fun previous => eqb previous.identity event.identity) then
    if previous != event then throw (.identityConflict event.identity)
    let cost := reservation eventSize plan run.accepted
    if run.work + cost > plan.limits.work then throw .workExhausted
    return { run with work := run.work + cost }
  let events := run.accepted ++ [event]
  if events.length > plan.limits.events then throw .eventsExhausted
  if (dedup (events.map Event.operation)).length > plan.limits.keys then throw .keysExhausted
  let cost := reservation eventSize plan events
  if run.work + cost > plan.limits.work then throw .workExhausted
  validateGraph events
  let mut staged := { run with accepted := events, work := run.work + cost }
  let ordered := events.mergeSort fun a b =>
    decide (idText a.identity.source < idText b.identity.source) ||
      (eqb a.identity.source b.identity.source && a.identity.ordinal ≤ b.identity.ordinal)
  for _ in [:events.length] do
    for candidate in ordered do
      if !has staged.processed candidate.identity && ready events staged.processed candidate then
        staged ← release stateOf staged candidate
    if staged.processed.length == events.length then break
  if staged.pending.length > plan.limits.buffered then throw .bufferExhausted
  return staged

end Shared.ScopedProjection
