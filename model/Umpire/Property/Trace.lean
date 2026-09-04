import Umpire.Property.Language

/-! Property-specific Model Trace coordinate adaptation and capability-limited projection. -/

namespace Umpire

/-- Look up a strict Model Trace coordinate only when it is compatible with this Property field.
Initial state is prior state only for a nonempty trace, and a resulting state is prior state only
when another step follows it. -/
def PropertyTraceField.valueAt?
    (field : PropertyTraceField)
    (trace : ModelTrace ModelValue ModelValue ModelValue ModelValue)
    (coordinate : ModelCoordinate) : Option ModelValue := do
  let value ← trace.valueAt? coordinate
  let compatible : Bool := match field with
    | .state | .selectedAction | .modelOutcome | .observation =>
        coordinate.definitionKind == field.definitionKind
    | .priorState => match coordinate with
        | .initialState => !trace.steps.isEmpty
        | .resultingState step => decide (step < trace.steps.length)
        | _ => false
    | .resultingState => match coordinate with
        | .resultingState _ => true
        | _ => false
    | .relation => coordinate.definitionKind == .observation
  if compatible then some value else none

structure PropertyTraceStep where
  priorState : Option ModelValue
  selectedAction : Option ModelValue
  modelOutcome : Option ModelValue
  resultingState : Option ModelValue
  observations : List ModelValue
  logicalTime : Option Nat
  deriving BEq, DecidableEq, Repr

/-- The evaluator's input contains only values admitted by the checked capability requirements. -/
structure PropertyTraceView where
  initialState : Option ModelValue
  steps : List PropertyTraceStep
  deriving BEq, DecidableEq, Repr

private def PropertyCapabilityView.allows
    (access : PropertyCapabilityView)
    (value : ModelValue) : Bool :=
  access.meanings.any fun meaning => meaning.definitionId == value.definitionId

private def PropertyCapabilityView.admit
    (access : PropertyCapabilityView)
    (value : ModelValue) : Option ModelValue :=
  if access.allows value then some value else none

private def logicalTimeOf
    (source : Option DefinitionId)
    (observations : List ModelValue)
    (previous : Option Nat) : Option Nat :=
  match source with
  | none => none
  | some id =>
      match observations.find? fun observation => observation.definitionId == id with
      | some observation =>
          match observation.value.toNat? with
          | some current =>
              if previous.any fun prior => current < prior then none else some current
          | none => none
      | none => previous

private def buildTraceSteps
    (access : PropertyCapabilityView)
    (priorState : Option ModelValue)
    (previousTime : Option Nat) :
    List (ModelTraceStep ModelValue ModelValue ModelValue ModelValue) →
      List PropertyTraceStep
  | [] => []
  | step :: rest =>
      let observations := step.observations.filter fun observation => access.allows observation
      let logicalTime := logicalTimeOf access.logicalTimeSource observations previousTime
      let resultingState := access.admit step.resultingState
      {
        priorState
        selectedAction := access.admit step.selectedAction
        modelOutcome := access.admit step.modelOutcome
        resultingState
        observations
        logicalTime
      } :: buildTraceSteps access resultingState logicalTime rest

def CheckedProperty.traceView
    (property : CheckedProperty)
    (trace : ModelTrace ModelValue ModelValue ModelValue ModelValue) :
    PropertyTraceView :=
  let initialState := property.access.admit trace.initialState
  {
    initialState
    steps := buildTraceSteps property.access initialState none trace.steps
  }

end Umpire
