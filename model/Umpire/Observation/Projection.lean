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
  private machine : Shared.ScopedProjection.Plan DefinitionId State Action
    (TransitionResult State Outcome Fact)
  private tableSound : ∀ row ∈ machine.transitions,
    target.kernel.authoritativeStep row.1 row.2.1 row.2.2
  private limitsMatch : machine.limits = declaration.limits
  canonicalBehavior : String
  behaviorFingerprint : BehaviorFingerprint

/-- Execution fields admitted by the projector, available to checked semantic consumers. -/
def Checked.scopeFields (plan : Checked target) : List DefinitionId := plan.declaration.scopeFields

/-- Immutable operation-key field admitted by the projector. -/
def Checked.operationField (plan : Checked target) : DefinitionId := plan.declaration.operationField

/-- Target-admitted initial state shared with checked semantic consumers. -/
def Checked.initialState (plan : Checked target) : State := plan.initial

/-- Closed executable table and mappings derived from the complete checked finite domain. -/
def Checked.executable (plan : Checked target) := plan.machine

/-- Original checked field policies and limits for lossless portable lowering. -/
def Checked.sourceDeclaration (plan : Checked target) := plan.declaration

/-- Every encoded-table row retains the checked Target's transition authority. -/
theorem Checked.table_authorized (plan : Checked target) (row)
    (member : row ∈ plan.executable.transitions) :
    target.kernel.authoritativeStep row.1 row.2.1 row.2.2 :=
  plan.tableSound row member

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
  let canonical := array [quote "checked-projection/v2", quote declaration.id.value,
    quote target.behaviorFingerprint.render, quote (domain.encodeSetup setup),
    quote (domain.encodeState initial),
    array (declaration.scopeFields.map (quote ∘ DefinitionId.value)),
    quote declaration.operationField.value,
    array (declaration.sources.map (quote ∘ DefinitionId.value)), array rules,
    array ([limits.events, limits.buffered, limits.keys, limits.support, limits.work,
      limits.eventSize].map toString)]
  let machine : Shared.ScopedProjection.Plan DefinitionId State Action
      (TransitionResult State Outcome Fact) := {
    initial
    rules := declaration.rules.map fun rule => ⟨rule.kind, rule.fields.length, rule.meaning⟩
    transitions := domain.states.flatMap fun prior => domain.actions.flatMap fun action =>
      (target.kernel.steps prior action).map fun result => (prior, action, result)
    limits := declaration.limits }
  have sound : ∀ row ∈ machine.transitions,
      target.kernel.authoritativeStep row.1 row.2.1 row.2.2 := by
    intro row member
    simp only [machine, List.mem_flatMap, List.mem_map] at member
    obtain ⟨prior, _, action, _, result, member, rfl⟩ := member
    exact target.kernel.stepSound prior action result member
  pure ⟨declaration, initial, machine, sound,
    rfl, canonical, behaviorFingerprintOf canonical⟩

/-- Opaque immutable Run state; only successful admission can replace it. -/
structure Run (plan : Checked target) where
  private mk ::
  private payload : Shared.ScopedProjection.Run plan.machine Field

/-- Allocate independent buffers, counters, keys, and semantic state for one Run binding. -/
def Checked.start (plan : Checked target) (scope : List (DefinitionId × String)) :
    Except Error (Run plan) := do
  let scope := scope.mergeSort fun a b => idLe a.1 b.1
  if scope.map Prod.fst != plan.declaration.scopeFields || scope.any (fun binding => binding.2.isEmpty)
    then throw .wrongScope
  pure ⟨{ scope }⟩

/-- Static validation and the projector reserve against the identical admitted limits. -/
theorem Checked.executable_limits (plan : Checked target) :
    plan.executable.limits = plan.sourceDeclaration.limits := plan.limitsMatch

/-- Immutable admitted evidence, including pending records, available only at the Observation boundary. -/
def Run.accepted (run : Run plan) : List Event := run.payload.accepted

private def checkedStep (scope : List (DefinitionId × String))
    (step : Shared.ScopedProjection.Step plan.machine) : Step target :=
  ⟨scope, plan.declaration.operationField, step.operation, step.priorState, step.action, step.result,
    step.directSupport, step.support, step.directRunSequences, step.runSequences,
    plan.tableSound _ step.member⟩

/-- All previously committed confirmed steps; rejected appends cannot revise them. -/
def Run.steps (run : Run plan) : List (Step target) :=
  run.payload.steps.map (checkedStep run.payload.scope)

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
      (field.value.map Shared.SemanticData.Scalar.size).getD 1) 0

/-- Closed admitted field policies, with authoring-only hash dispositions remaining unsupported. -/
def Checked.fieldPolicies (plan : Checked target) : List (DefinitionId × List (DefinitionId × Nat × Nat)) :=
  plan.declaration.rules.map fun rule => (rule.kind, rule.fields.map fun (field, disposition) =>
    (field.id, match field.valueType with | .text => 1 | .natural => 2 | .boolean => 3,
      match disposition with | .retain => 1 | .redact => 2 | .reject => 3 | .hash _ => 0))

/-- Validate through the same closed field-policy boundary as portable evidence. -/
def Checked.validateEvent (plan : Checked target) (scope : List (DefinitionId × String))
    (event : Event) : Except Error Unit :=
  Shared.ScopedProjection.validateEvidence
    (fun value => match value with | .text _ => 1 | .natural _ => 2 | .boolean _ => 3)
    eventSize plan.declaration.limits plan.declaration.sources plan.fieldPolicies scope event

/-- Atomically admit one event; rejection exposes neither staged state nor any newly released steps. -/
def Run.admit [DecidableEq State] [DecidableEq Action] [DecidableEq Outcome] [DecidableEq Fact]
    (run : Run plan) (event : Event) : Except Error (Run plan × Progress (Step target)) := do
  if run.isClosed then throw .closed
  plan.validateEvent run.payload.scope event
  let payload ← run.payload.admit DefinitionId.value (·.resultingState) eventSize event
  let emissions := (payload.steps.drop run.payload.steps.length).map (checkedStep payload.scope)
  let pending := payload.pending
  let progress := match emissions with
    | first :: rest => .emitted first rest
    | [] => if pending.contains event.identity then .pending pending else .stutter
  return (⟨payload⟩, progress)

/-- Close only a fully supported, nonempty projection whose operation states are Target-terminal. -/
def Run.close (run : Run plan) : Except Error (Run plan) := do
  if run.isClosed then return run
  if !run.pending.isEmpty then throw (.incomplete run.pending)
  let operations := (run.accepted.map (·.operation)).eraseDups
  let nonterminal := operations.filter fun operation => !target.isTerminal (run.state operation)
  if operations.isEmpty || !nonterminal.isEmpty then throw (.nonterminal nonterminal)
  let reservation := (run.accepted.foldl (fun size event => size + eventSize event) 0 + 1) *
    (operations.length + 1) * (run.accepted.length + 1)
  if run.work + reservation > plan.declaration.limits.work then throw .workExhausted
  return ⟨{ run.payload with closed := true, work := run.work + reservation }⟩

end Umpire.Observation.Projection
