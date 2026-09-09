import Umpire.Observation.Projection
import Umpire.Property.Scoped

/-!
Atomic composition of checked evidence projection with the Property-owned scoped kernel.
Only newly emitted, Target-authorized steps tick obligations. Pending evidence is retained by the
projector; it cannot establish satisfaction. Projection or Property rejection returns no replacement
Run and cannot erase an answer already supported by the previous immutable Run.
-/

namespace Umpire.Observation.Scoped

open Property.Scoped

variable {Law : LawDefinition → Prop} {Setup : Type}
variable {target : CheckedTarget Law Setup ModelValue ModelValue ModelValue ModelValue}

/-- Failures retain which checked boundary rejected the append. -/
inductive Error where
  | projection (error : Projection.Error)
  | property (error : Property.Scoped.Error)
  deriving BEq, DecidableEq, Repr

/-- The consumer's binding comes from the admitted projector, never a second authored scope.
The evidence projector supplies no typed field values, so a clause that declares keyed captures is
rejected here rather than admitted into a run whose captures could never bind. -/
def compile (plan : Projection.Checked target) (property : CheckedProperty) (limits : Limits) :
    Except Error (Compiled target) := do
  if let some clause := property.scopedClauses.find? fun clause =>
      !clause.declaration.captures.isEmpty || clause.correlation.isSome then
    throw (.property (.unsupported clause.declaration.id "keyed field captures without evidence projection"))
  (Property.Scoped.compile target property plan.scopeFields plan.operationField limits).mapError
    Error.property

/-- Both run-local states advance or reject together. -/
structure Run (plan : Projection.Checked target) (compiled : Compiled target) where
  private mk ::
  private evidence : Projection.Run plan
  private semantic : Property.Scoped.Run compiled
  private closed : Bool := false

/-- Allocate fresh projection and obligation state under the same immutable execution bindings. -/
def start [DecidableEq Setup] (plan : Projection.Checked target) (compiled : Compiled target)
    (setup : Setup) (scope : List (DefinitionId × String)) : Except Error (Run plan compiled) := do
  if compiled.scopeFields != plan.scopeFields || compiled.operationField != plan.operationField then
    throw (.property .wrongScope)
  let evidence ← (plan.start scope).mapError Error.projection
  let semantic ← (compiled.start setup plan.initialState scope).mapError Error.property
  pure ⟨evidence, semantic, false⟩

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
    target.kernel.authoritativeStep (semanticStep step).priorState (semanticStep step).action
      (semanticStep step).result := step.authorized

/-- Consume only the append's new emissions. Re-reading accepted evidence cannot tick a self-loop. -/
def Run.admit {plan : Projection.Checked target} {compiled : Compiled target}
    (run : Run plan compiled) (event : Projection.Event) : Except Error (Run plan compiled) := do
  if run.closed then throw (.property .closed)
  let (evidence, progress) ← (run.evidence.admit event).mapError Error.projection
  let semantic ← (run.semantic.consumeMany (progress.emissions.map semanticStep)).mapError Error.property
  pure ⟨evidence, semantic, false⟩

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

end Umpire.Observation.Scoped
