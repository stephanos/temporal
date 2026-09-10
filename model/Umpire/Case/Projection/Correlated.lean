import Umpire.Case.Projection.Coverage
import Umpire.Property.Correlated

/-!
Atomic composition of checked evidence projection with the Property-owned correlated kernel.
Only newly emitted, Target-authorized steps tick obligations. Pending evidence is retained by the
projector; it cannot establish satisfaction. Projection or Property rejection returns no replacement
Run and cannot erase an answer already supported by the previous immutable Run.

A Run also carries the checked modeled-fields to declared-Observations coverage map. Each admitted
event's covered declared fields are rebuilt into admitted projections at their modeled coordinates,
so a clause's keyed captures retain exactly the values the declared Observations supplied. A rebuilt
projection witnesses the declared Observation's value at those coordinates, and nothing more: a
request operand denotes the selected Action's own arguments, which no projected scalar
reconstructs, so a clause reading one is rejected rather than admitted into a Run whose operands
could never bind.
-/

namespace Umpire.Case.Projection.Correlated

open Property.Correlated

variable {Law : Law → Prop} {Setup : Type}
variable {target : CheckedModel Law Setup ModelValue ModelValue ModelValue ModelValue}

/-- Failures retain which checked boundary rejected the append. -/
inductive Error where
  | projection (error : Projection.Error)
  | property (error : Property.Correlated.Error)
  | coverage (error : Projection.CoverageError)
  deriving BEq, DecidableEq, Repr

/-- What the coverage map must supply before an evidence-driven Run may admit a clause. Every
modeled operand a clause reads must be covered by a declared Observation this projection emits, and
that coverage must be rebuildable: a request operand denotes the selected Action's own arguments,
which no projected scalar reconstructs, so a clause reading one is rejected here rather than
admitted into a Run whose operands could never bind. -/
private def checkCoverage {plan : Projection.Checked target}
    (coverage : Projection.Coverage plan) (property : CheckedProperty) : Except Error Unit := do
  for clause in property.correlatedRules do
    let unsupported := fun reason =>
      Error.property (.unsupported clause.declaration.id reason)
    let require := fun (path : PropertyFieldPath) => do
      match coverage.entryOf? { path with capture := none } with
      | none => throw (unsupported "field coordinates have no declared Observation")
      | some entry =>
        if !entry.rebuildable then
          throw (unsupported "request operand is not rebuildable from projected evidence")
    for declaration in clause.declaration.captures do
      require declaration.path
    if let some correlation := clause.correlation then
      for operand in correlation.expression.fieldOperands do
        if let .field path _ := operand then
          require path

/-- The consumer's binding comes from the admitted projector, never a second authored scope.
A clause that declares keyed captures or a correlation is admitted only when the requested coverage
supplies every field it reads; the default empty coverage therefore rejects every such clause. -/
def compile (plan : Projection.Checked target) (property : CheckedProperty)
    (limits : Property.Correlated.Limits)
    (coverage : Projection.Coverage plan := Projection.Coverage.empty plan) :
    Except Error (Compiled target) := do
  checkCoverage coverage property
  (Property.Correlated.compile target property plan.scopeFields plan.operationField limits).mapError
    Error.property

/-- Both run-local states advance or reject together. -/
structure Run (plan : Projection.Checked target) (compiled : Compiled target) where
  private mk ::
  private evidence : Projection.Run plan
  private semantic : Property.Correlated.Run compiled
  private coverage : Projection.Coverage plan
  private closed : Bool := false

/-- Allocate fresh projection and obligation state under the same immutable execution bindings. -/
def start [DecidableEq Setup] (plan : Projection.Checked target) (compiled : Compiled target)
    (setup : Setup) (scope : List (DefinitionId × String))
    (coverage : Projection.Coverage plan := Projection.Coverage.empty plan) :
    Except Error (Run plan compiled) := do
  if compiled.scopeFields != plan.scopeFields || compiled.operationField != plan.operationField then
    throw (.property .wrongScope)
  -- The Run reads its own coverage, so a Run started under a coverage that no longer supplies the
  -- compiled clauses is rejected here rather than admitting a capture that could never bind.
  checkCoverage coverage compiled.property
  let evidence ← (plan.start scope).mapError Error.projection
  let semantic ← (compiled.start setup plan.initialState scope).mapError Error.property
  pure ⟨evidence, semantic, coverage, false⟩

/-- Forget evidence support only after the projector has supplied Target authority. -/
def semanticStep (step : Projection.Step target) : Transition := {
  scope := step.scope
  operationField := step.operationField
  operation := step.operation
  priorState := step.priorState
  action := step.action
  result := step.result
}

/-- Each adapted step retains the projector's exact authoritative transition. -/
theorem semanticStep_authorized (step : Projection.Step target) :
    target.machine.authoritativeStep (semanticStep step).priorState (semanticStep step).action
      (semanticStep step).result := step.authorized

/-- Consume only the append's new emissions, together with the covered projections this event's
declared fields supply. Re-reading accepted evidence cannot tick a self-loop, and a declared field
whose value the modeled coordinates cannot denote rejects the whole append. -/
def Run.admit {plan : Projection.Checked target} {compiled : Compiled target}
    (run : Run plan compiled) (event : Projection.Event) : Except Error (Run plan compiled) := do
  if run.closed then throw (.property .closed)
  let (evidence, progress) ← (run.evidence.admit event).mapError Error.projection
  let values ← (run.coverage.evidence event.fields).mapError Error.coverage
  let semantic ← (run.semantic.consumeEvidence
    (progress.emissions.map fun step => (semanticStep step, values))).mapError Error.property
  pure ⟨evidence, semantic, run.coverage, false⟩

/-- Offline replay and incremental evidence admission share the same atomic append. -/
def Run.admitMany {plan : Projection.Checked target} {compiled : Compiled target}
    (run : Run plan compiled) (events : List Projection.Event) : Except Error (Run plan compiled) :=
  events.foldlM Run.admit run

/-- Partial causal evidence cannot close a selected finite trace. Deadline violations already
proved by admitted steps survive this incomplete-evidence interpretation. -/
def Run.answers {plan : Projection.Checked target} {compiled : Compiled target}
    (run : Run plan compiled) : List (DefinitionId × PropertyEndpointAnswer) :=
  run.semantic.answers (!run.evidence.pending.isEmpty)

/-- Stop evidence admission without inventing a Target transition or requiring runtime terminality. -/
def Run.close {plan : Projection.Checked target} {compiled : Compiled target}
    (run : Run plan compiled) : Run plan compiled :=
  { run with semantic := run.semantic.close, closed := true }

/-- Chunk equality includes splits inside unresolved causal evidence and the first rejection. -/
theorem Run.admitMany_append {plan : Projection.Checked target} {compiled : Compiled target}
    (run : Run plan compiled) (first second : List Projection.Event) :
    run.admitMany (first ++ second) =
      (run.admitMany first >>= fun next => next.admitMany second) := by
  simp [admitMany, List.foldlM_append]

end Umpire.Case.Projection.Correlated
