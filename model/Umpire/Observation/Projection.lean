import Umpire.Observation.Projection.Declaration

/-!
Checked incremental evidence projection. `check` binds closed mappings to one Target; `start`
allocates fresh Run state; `admit` stages a whole append, including buffered releases, before
returning its replacement state. An error returns no replacement state or emissions. `close`
requires causal closure and the Target's explicit terminal semantics.

Support contains exactly the event and its declared causal ancestors, not unrelated source-prefix
records. Source predecessors constrain scheduling without becoming causal support. The conservative
work reservation bounds the finite graph scans, support unions, and Target transition comparisons;
it is charged before staging, independently of the buffer, key, and retained-support ceilings.
-/

namespace Umpire.Observation.Projection

variable {Law : LawDefinition → Prop} {Setup State Action Outcome Fact : Type}
variable {target : CheckedTarget Law Setup State Action Outcome Fact}

/-- A checked projection is tied to its Target and an admitted initial state. -/
structure Checked (target : CheckedTarget Law Setup State Action Outcome Fact) where
  private mk ::
  private declaration : Declaration State Action Outcome Fact
  private initial : State
  private transitionCount : Nat
  canonicalBehavior : String
  behaviorFingerprint : BehaviorFingerprint

/-- A confirmed step carries kernel authority and exact direct/transitive source support. -/
structure Step (target : CheckedTarget Law Setup State Action Outcome Fact) where
  private mk ::
  scope : List (DefinitionId × String)
  operationField : DefinitionId
  operation : String
  priorState : State
  action : Action
  result : TransitionResult State Outcome Fact
  directSupport : List Identity
  support : List Identity
  directRunSequences : List Nat
  runSequences : List Nat
  authorized : target.kernel.authoritativeStep priorState action result

/-- Forget only evidence, retaining the Target-owned semantic step for checked consumers. -/
def Step.semantic (step : Step target) : ModelTraceStep State Action Outcome Fact :=
  .result step.action step.result

variable {plan : Checked target}

private def quote (value : String) : String := Lean.Json.compress (.str value)
private def array (values : List String) : String := "[" ++ String.intercalate "," values ++ "]"
private def idLe (a b : DefinitionId) : Bool := decide (a.value ≤ b.value)

private def meaningJson
    (state : State → String) (action : Action → String)
    (outcome : Outcome → String) (fact : Fact → String) :
    Meaning State Action Outcome Fact → String
  | .irrelevant => array [quote "irrelevant"]
  | .submission value => array [quote "submission", quote (action value)]
  | .confirmed required steps => array [quote "confirmed",
      match required with | none => "null" | some value => quote (action value),
      array (steps.map fun (selected, result) => array [quote (action selected),
        quote (state result.resultingState), quote (outcome result.modelOutcome),
        array (result.observations.map (quote ∘ fact))])]

/-- Validate declarations before allocating Run state; no semantic result is inferred from submission. -/
def check [DecidableEq Setup] [DecidableEq State] [DecidableEq Action]
    [DecidableEq Outcome] [DecidableEq Fact]
    (target : CheckedTarget Law Setup State Action Outcome Fact)
    (declaration : Declaration State Action Outcome Fact) (setup : Setup) (initial : State) :
    Except Error (Checked target) := do
  if declaration.version != 1 then throw .unsupportedVersion
  let identities := [declaration.id, declaration.operationField] ++ declaration.scopeFields ++
    declaration.sources ++ declaration.rules.flatMap (fun rule =>
      rule.kind :: rule.fields.map (fun field => field.1.id))
  if identities.any (fun identity => !identity.validate.isOk) then throw .invalidDeclaration
  if declaration.scopeFields.isEmpty || declaration.sources.isEmpty ||
      declaration.scopeFields.contains declaration.operationField ||
      declaration.rules.isEmpty || declaration.scopeFields.eraseDups != declaration.scopeFields ||
      declaration.sources.eraseDups != declaration.sources ||
      (declaration.rules.map Rule.kind).eraseDups != declaration.rules.map Rule.kind then
    throw .invalidDeclaration
  if !(setup ∈ target.resolvedSetups) || !(initial ∈ target.kernel.initialStates setup) then
    throw .invalidInitialState
  let .complete domain := target.kernel.behaviorDomain | throw .invalidDeclaration
  for rule in declaration.rules do
    if (rule.fields.map fun field => field.1.id).eraseDups != rule.fields.map (fun field => field.1.id)
      then throw .invalidDeclaration
    for (_, disposition) in rule.fields do
      if let .hash _ := disposition then throw .unsupportedDisposition
    match rule.meaning with
    | .irrelevant => pure ()
    | .submission action =>
        if !(action ∈ domain.actions) then throw .invalidDeclaration
    | .confirmed required steps =>
        if steps.isEmpty then throw .invalidDeclaration
        if required.any (fun action => !(action ∈ domain.actions)) then throw .invalidDeclaration
        for (action, result) in steps do
          if !(domain.states.any fun state =>
              (target.kernel.steps state action).any (fun candidate => decide (candidate = result))) then
            throw .invalidDeclaration
        if required.any (fun action => !(declaration.rules.any fun candidate =>
            match candidate.meaning with
            | .submission submitted => decide (submitted = action)
            | _ => false)) then throw .invalidDeclaration
  let declaration := { declaration with
    scopeFields := declaration.scopeFields.mergeSort idLe
    sources := declaration.sources.mergeSort idLe
    rules := (declaration.rules.map fun rule => { rule with
      fields := rule.fields.mergeSort (fun a b => idLe a.1.id b.1.id) }).mergeSort
        (fun a b => idLe a.kind b.kind) }
  let rules := declaration.rules.map fun rule => array [quote rule.kind.value,
    array (rule.fields.map fun (field, disposition) => array [quote field.id.value,
      quote field.valueType.name, quote disposition.name]),
    meaningJson domain.encodeState domain.encodeAction domain.encodeOutcome domain.encodeObservation
      rule.meaning]
  let limits := declaration.limits
  let canonical := array [quote "checked-projection/v1", quote declaration.id.value,
    quote target.behaviorFingerprint.render, quote (domain.encodeSetup setup),
    quote (domain.encodeState initial),
    array (declaration.scopeFields.map (quote ∘ DefinitionId.value)),
    quote declaration.operationField.value,
    array (declaration.sources.map (quote ∘ DefinitionId.value)), array rules,
    array ([limits.events, limits.buffered, limits.keys, limits.support, limits.work,
      limits.eventSize].map toString)]
  pure ⟨declaration, initial, target.behaviorDescription.transitions.length, canonical,
    behaviorFingerprintOf canonical⟩

private structure Payload (target : CheckedTarget Law Setup State Action Outcome Fact) where
  scope : List (DefinitionId × String)
  accepted : List Event := []
  processed : List Identity := []
  support : List (Identity × List Identity) := []
  states : List (String × State) := []
  steps : List (Step target) := []
  work : Nat := 0
  closed : Bool := false

/-- Opaque immutable Run state; only successful admission can replace it. -/
structure Run (plan : Checked target) where
  private mk ::
  private payload : Payload target

/-- Allocate independent buffers, counters, keys, and semantic state for one Run binding. -/
def Checked.start (plan : Checked target) (scope : List (DefinitionId × String)) :
    Except Error (Run plan) := do
  let scope := scope.mergeSort fun a b => idLe a.1 b.1
  if scope.map Prod.fst != plan.declaration.scopeFields || scope.any (fun binding => binding.2.isEmpty)
    then throw .wrongScope
  pure ⟨{ scope }⟩

/-- Immutable admitted evidence, including pending records, available only at the Observation boundary. -/
def Run.accepted (run : Run plan) : List Event := run.payload.accepted

/-- All previously committed confirmed steps; rejected appends cannot revise them. -/
def Run.steps (run : Run plan) : List (Step target) := run.payload.steps

/-- The remaining causal/source-order buffer. -/
def Run.pending (run : Run plan) : List Identity :=
  run.payload.accepted.filterMap fun event =>
    if run.payload.processed.contains event.identity then none else some event.identity

/-- Retained Target state for an operation, or the admitted initial state before its first step. -/
def Run.state (run : Run plan) (operation : String) : State :=
  (run.payload.states.find? fun entry => entry.1 == operation).map Prod.snd |>.getD plan.initial

/-- Work committed by successful admissions, in conservative graph-comparison units. -/
def Run.work (run : Run plan) : Nat := run.payload.work

/-- Closure is immutable and rejects further evidence. -/
def Run.isClosed (run : Run plan) : Bool := run.payload.closed

private def identitySize (identity : Identity) : Nat :=
  identity.source.value.length + 1 + identity.scope.foldl (fun size (key, value) =>
    size + key.value.length + value.length) 0

private def eventSize (event : Event) : Nat :=
  identitySize event.identity + event.operation.length + event.kind.value.length +
    event.runSequences.foldl (fun size sequence => size + (toString sequence).length) 0 +
    event.parents.foldl (fun size parent => size + identitySize parent) 0 +
    event.fields.foldl (fun size field => size + field.id.value.length +
      (field.value.map (fun value => value.render.length)).getD 1) 0

private def validateEvent (plan : Checked target) (scope : List (DefinitionId × String))
    (event : Event) : Except Error Unit := do
  if eventSize event > plan.declaration.limits.eventSize then throw .eventSizeExhausted
  if event.identity.scope != scope || event.parents.any (fun parent => parent.scope != scope) then
    throw .wrongScope
  if !plan.declaration.sources.contains event.identity.source ||
      event.parents.any (fun parent => !plan.declaration.sources.contains parent.source) then
    throw .unknownSource
  if event.operation.isEmpty then throw .wrongScope
  if event.runSequences.isEmpty || event.runSequences.contains 0 ||
      event.runSequences.eraseDups != event.runSequences then throw .invalidEvidenceSupport
  if event.identity.ordinal >= plan.declaration.limits.events ||
      event.parents.any (fun parent => parent.ordinal >= plan.declaration.limits.events) then
    throw .eventsExhausted
  let some rule := plan.declaration.rules.find? (fun rule => rule.kind == event.kind)
    | throw (.unsupportedEvidence event.kind)
  if event.parents.eraseDups != event.parents then throw .invalidDeclaration
  if (event.fields.map Field.id).eraseDups != event.fields.map Field.id then
    throw .invalidDeclaration
  for field in event.fields do
    let some (declared, disposition) := rule.fields.find? (fun entry => entry.1.id == field.id)
      | throw (.unauthorizedField field.id)
    match disposition, field.value with
    | .retain, some value =>
        if value.valueType != declared.valueType then throw (.unauthorizedField field.id)
    | .redact, none => pure ()
    | _, _ => throw (.unauthorizedField field.id)
  for (field, disposition) in rule.fields do
    if disposition != .reject && !(event.fields.any fun value => value.id == field.id) then
      throw (.unauthorizedField field.id)

private def sourceBefore (a b : Identity) : Bool :=
  a.source == b.source && a.ordinal < b.ordinal

private def orderingEdge (before : Identity) (after : Event) : Bool :=
  after.parents.contains before || sourceBefore before after.identity

private def orderingReaches (events : List Event) (before after : Identity) : Bool := Id.run do
  if before == after then return true
  let nodes := events.toArray
  let mut visited := nodes.map fun event => event.identity == before
  let mut frontier := [before]
  for _ in [:nodes.size] do
    let current :: rest := frontier | return false
    frontier := rest
    for index in [:nodes.size] do
      if !(visited[index]?).getD false then
        if let some candidate := nodes[index]? then
          if orderingEdge current candidate then
            if candidate.identity == after then return true
            visited := visited.set! index true
            frontier := frontier ++ [candidate.identity]
  return false

private def identityLe (a b : Identity) : Bool :=
  decide (a.source.value < b.source.value) ||
    (a.source == b.source && a.ordinal ≤ b.ordinal)

private def ready (events : List Event) (processed : List Identity) (event : Event) : Bool :=
  event.parents.all processed.contains &&
    (event.identity.ordinal == 0 || events.any fun candidate =>
      candidate.identity.source == event.identity.source &&
      candidate.identity.ordinal + 1 == event.identity.ordinal &&
      processed.contains candidate.identity)

private def acyclic (events : List Event) : Bool := Id.run do
  let mut removed : List Identity := []
  for _ in [:events.length] do
    for event in events do
      if !removed.contains event.identity && events.all (fun predecessor =>
          !orderingEdge predecessor.identity event || removed.contains predecessor.identity)
        then removed := event.identity :: removed
    if removed.length == events.length then return true
  return removed.length == events.length

private def validateGraph (events : List Event) : Except Error Unit := do
  for event in events do
    for parent in event.parents do
      if let some predecessor := events.find? (fun candidate => candidate.identity == parent) then
        if predecessor.operation != event.operation then
          throw (.wrongOperation event.identity parent)
  if !acyclic events then throw .causalCycle

private def release [DecidableEq State] [DecidableEq Action] [DecidableEq Outcome]
    [DecidableEq Fact] (plan : Checked target) (payload : Payload target) (event : Event) :
    Except Error (Payload target) := do
  let _ : BEq (TransitionResult State Outcome Fact) := ⟨fun a b => decide (a = b)⟩
  let _ : LawfulBEq (TransitionResult State Outcome Fact) := {
    eq_of_beq := of_decide_eq_true
    rfl := of_decide_eq_self_eq_true _ }
  let some rule := plan.declaration.rules.find? (fun rule => rule.kind == event.kind)
    | throw (.unsupportedEvidence event.kind)
  let ancestors := event.parents.flatMap fun parent =>
    (payload.support.find? fun entry => entry.1 == parent).map Prod.snd |>.getD []
  let support := (ancestors ++ [event.identity]).eraseDups
  let direct := (event.identity :: event.parents).eraseDups
  let sequences := fun identities =>
    ((identities.flatMap fun identity =>
      (payload.accepted.find? fun candidate => candidate.identity == identity).map Event.runSequences
        |>.getD []).mergeSort).eraseDups
  let mut payload := { payload with
    processed := payload.processed ++ [event.identity]
    support := payload.support ++ [(event.identity, support)] }
  match rule.meaning with
  | .irrelevant | .submission _ => pure ()
  | .confirmed required outputs =>
      if let some previous := payload.steps.reverse.find? (fun step => step.operation == event.operation)
        then
          if let some identity := previous.directSupport.head? then
            if !orderingReaches payload.accepted identity event.identity then
              throw (.incomparableOrder event.identity identity)
      if required.any (fun action => !(event.parents.any fun parent =>
          (payload.accepted.find? fun candidate => candidate.identity == parent).any fun candidate =>
            (plan.declaration.rules.find? fun rule => rule.kind == candidate.kind).any fun parentRule =>
              match parentRule.meaning with
              | .submission submitted => decide (submitted = action)
              | _ => false)) then throw (.missingSubmission event.identity)
      for (action, result) in outputs do
        let prior := (payload.states.find? fun entry => entry.1 == event.operation).map Prod.snd
          |>.getD plan.initial
        if h : result ∈ target.kernel.steps prior action then
          let step : Step target := ⟨payload.scope, plan.declaration.operationField, event.operation, prior, action, result,
            direct, support, sequences direct, sequences support,
            target.kernel.stepSound prior action result h⟩
          payload := { payload with
            states := (payload.states.filter fun entry => entry.1 != event.operation) ++
              [(event.operation, result.resultingState)]
            steps := payload.steps ++ [step] }
        else throw (.invalidTransition event.identity)
  let retained := payload.support.foldl (fun n entry => n + entry.2.length) 0 +
    payload.steps.foldl (fun n step => n + step.directSupport.length + step.support.length +
      step.directRunSequences.length + step.runSequences.length) 0
  if retained > plan.declaration.limits.support then throw .supportExhausted
  return payload

private def workReservation (plan : Checked target) (events : List Event) : Nat :=
  let n := events.length + 1
  let outputs := plan.declaration.rules.foldl (fun n rule => n + rule.fields.length +
    (match rule.meaning with
    | .confirmed _ steps => steps.length
    | _ => 1)) 0
  (events.foldl (fun size event => size + eventSize event) 0 + 1) *
    n * n * n * (outputs + plan.transitionCount + 1)

private def stage [DecidableEq State] [DecidableEq Action] [DecidableEq Outcome]
    [DecidableEq Fact] (plan : Checked target) (payload : Payload target) (event : Event) :
    Except Error (Payload target × Progress (Step target)) := do
  if payload.closed then throw .closed
  validateEvent plan payload.scope event
  if let some previous := payload.accepted.find? (fun previous => previous.identity == event.identity)
    then
      if previous != event then throw (.identityConflict event.identity)
      let reservation := workReservation plan payload.accepted
      if payload.work + reservation > plan.declaration.limits.work then throw .workExhausted
      return ({ payload with work := payload.work + reservation }, .stutter)
  let events := payload.accepted ++ [event]
  if events.length > plan.declaration.limits.events then throw .eventsExhausted
  if (events.map Event.operation).eraseDups.length > plan.declaration.limits.keys then
    throw .keysExhausted
  let reservation := workReservation plan events
  if payload.work + reservation > plan.declaration.limits.work then throw .workExhausted
  validateGraph events
  let mut staged := { payload with accepted := events, work := payload.work + reservation }
  let ordered := events.mergeSort fun a b => identityLe a.identity b.identity
  for _ in [:events.length] do
    for candidate in ordered do
      if !staged.processed.contains candidate.identity &&
          ready events staged.processed candidate then
        staged ← release plan staged candidate
    if staged.processed.length == events.length then break
  let pending := events.filterMap fun candidate =>
    if staged.processed.contains candidate.identity then none else some candidate.identity
  if pending.length > plan.declaration.limits.buffered then throw .bufferExhausted
  let emissions := staged.steps.drop payload.steps.length
  let progress := match emissions with
    | first :: rest => .emitted first rest
    | [] => if pending.contains event.identity then .pending pending else .stutter
  return (staged, progress)

/-- Atomically admit one event; rejection exposes neither staged state nor any newly released steps. -/
def Run.admit [DecidableEq State] [DecidableEq Action] [DecidableEq Outcome] [DecidableEq Fact]
    (run : Run plan) (event : Event) : Except Error (Run plan × Progress (Step target)) := do
  let (payload, progress) ← stage plan run.payload event
  return (⟨payload⟩, progress)

/-- Close only a fully supported, nonempty projection whose operation states are Target-terminal. -/
def Run.close (run : Run plan) : Except Error (Run plan) := do
  if run.isClosed then return run
  if !run.pending.isEmpty then throw (.incomplete run.pending)
  let operations := (run.accepted.map Event.operation).eraseDups
  let nonterminal := operations.filter fun operation => !target.isTerminal (run.state operation)
  if operations.isEmpty || !nonterminal.isEmpty then throw (.nonterminal nonterminal)
  let reservation := (run.accepted.foldl (fun size event => size + eventSize event) 0 + 1) *
    (operations.length + 1) * (run.accepted.length + 1)
  if run.work + reservation > plan.declaration.limits.work then throw .workExhausted
  return ⟨{ run.payload with closed := true, work := run.work + reservation }⟩

end Umpire.Observation.Projection
