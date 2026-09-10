import Umpire.Shared.DefinitionGraph
import Umpire.Model.Check
import Umpire.Id

/-! Checked, portable constraints over pure Model Traces: the authored Scenario record. -/

namespace Umpire

/-! Checked, portable constraints over pure Model Traces. -/

inductive ScenarioErrorKind where
  | emptyDefinitionId
  | invalidDefinitionId
  | duplicateDefinitionId
  | unknownReference
  | wrongReferenceKind
  | invalidBinding
  | contradictoryOccurrenceBounds
  | contradictoryConstraint
  | forbiddenRequired
  | duplicateOrdering
  | selfOrdering
  | cyclicOrdering
  | occurrenceLimitExceeded
  | incompleteExactTrace
  deriving BEq, DecidableEq, Ord, Repr

def ScenarioErrorKind.name : ScenarioErrorKind → String
  | .emptyDefinitionId => "empty-definition-id"
  | .invalidDefinitionId => "invalid-definition-id"
  | .duplicateDefinitionId => "duplicate-definition-id"
  | .unknownReference => "unknown-reference"
  | .wrongReferenceKind => "wrong-reference-kind"
  | .invalidBinding => "invalid-binding"
  | .contradictoryOccurrenceBounds => "contradictory-occurrence-bounds"
  | .contradictoryConstraint => "contradictory-constraint"
  | .forbiddenRequired => "forbidden-required"
  | .duplicateOrdering => "duplicate-ordering"
  | .selfOrdering => "self-ordering"
  | .cyclicOrdering => "cyclic-ordering"
  | .occurrenceLimitExceeded => "occurrence-limit-exceeded"
  | .incompleteExactTrace => "incomplete-exact-trace"

structure ScenarioError where
  kind : ScenarioErrorKind
  definitionId : DefinitionId
  sourcePath : String
  offendingValue : String
  relatedDefinitionIds : List DefinitionId
  deriving BEq, DecidableEq, Repr

/-- A symbolic setup role retains the kind of Model Value that may bind it. -/
structure Scenario.Role where
  id : DefinitionId
  valueKind : DefinitionKind
  deriving BEq, DecidableEq, Repr

inductive SetupOperand where
  | role (id : DefinitionId)
  | value (value : ModelValue)
  deriving BEq, DecidableEq, Repr

inductive SetupRelation where
  | equal
  | different
  deriving BEq, DecidableEq, Ord, Repr

def SetupRelation.name : SetupRelation → String
  | .equal => "equal"
  | .different => "different"

structure SetupConstraint where
  id : DefinitionId
  relation : SetupRelation
  left : SetupOperand
  right : SetupOperand
  deriving BEq, DecidableEq, Repr

/-- Require one symbolic setup role to equal one concrete Model Value. -/
def SetupConstraint.roleEquals
    (id : DefinitionId)
    (role : DefinitionId)
    (value : ModelValue) : SetupConstraint := {
  id
  relation := .equal
  left := .role role
  right := .value value
}

/-- A required action occurrence has a stable Definition ID independent of its action Definition ID. -/
structure Scenario.Step where
  id : DefinitionId
  action : DefinitionId
  deriving BEq, DecidableEq, Repr

structure Scenario.Count where
  action : DefinitionId
  minimum : Nat := 0
  maximum : Option Nat := none
  deriving BEq, DecidableEq, Repr

namespace Scenario.Count

def exactly (action : DefinitionId) (count : Nat) : Scenario.Count :=
  { action, minimum := count, maximum := some count }

def atLeast (action : DefinitionId) (count : Nat) : Scenario.Count :=
  { action, minimum := count }

def atMost (action : DefinitionId) (count : Nat) : Scenario.Count :=
  { action, maximum := some count }

end Scenario.Count

structure Scenario.Order where
  before : DefinitionId
  after : DefinitionId
  deriving BEq, DecidableEq, Repr

/-- Optional fields keep malformed promoted witnesses representable until checking. -/
structure AuthoredExactTraceStep where
  selectedAction : Option ModelValue
  outcome : Option ModelValue
  resultingState : Option ModelValue
  observations : Option (List ModelValue)
  deriving BEq, DecidableEq, Repr

structure AuthoredExactTrace where
  setup : List RoleBinding
  initialState : Option ModelValue
  steps : List AuthoredExactTraceStep
  deriving BEq, DecidableEq, Repr

/-- A complete pure trace together with the symbolic setup bindings that selected it. -/
structure Scenario.Trace where
  setup : List RoleBinding
  trace : ModelTrace ModelValue ModelValue ModelValue ModelValue
  deriving BEq, DecidableEq, Repr

/-- Build a complete Scenario trace containing one model-owned transition result. -/
def Scenario.Trace.singleStep
    (setup : List RoleBinding)
    (initialState : ModelValue)
    (selectedAction : ModelValue)
    (result : Umpire.Step ModelValue ModelValue ModelValue) : Scenario.Trace := {
  setup
  trace := {
    initialState
    steps := [ModelTraceStep.result selectedAction result]
  }
}

structure Scenario where
  id : DefinitionId
  source : SourceLocation
  version : Nat := 1
  requires : List DefinitionId := []
  roles : List Scenario.Role := []
  setup : List SetupConstraint := []
  allowedActions : List DefinitionId := []
  requiredOccurrences : List Scenario.Step := []
  forbiddenActions : List DefinitionId := []
  occurrenceBounds : List Scenario.Count := []
  ordering : List Scenario.Order := []
  sequences : List (List DefinitionId) := []
  adjacencies : List (List DefinitionId) := []
  actionsExactly : Option (List DefinitionId) := none
  traceExactly : Option AuthoredExactTrace := none
  documentation : String := ""
  deriving BEq, DecidableEq, Repr

/-- Declare exactly one occurrence of one Action while leaving its result to the Target. -/
def Scenario.exactlyOneAction
    (id : DefinitionId)
    (source : SourceLocation)
    (occurrence : Scenario.Step)
    (requires : List DefinitionId := [])
    (roles : List Scenario.Role := [])
    (setup : List SetupConstraint := [])
    (documentation : String := "") : Scenario := {
  id
  source
  requires
  roles
  setup
  allowedActions := [occurrence.action]
  requiredOccurrences := [occurrence]
  occurrenceBounds := [Scenario.Count.exactly occurrence.action 1]
  actionsExactly := some [occurrence.action]
  documentation
}

structure ScenarioCheckContext where
  definitions : List DefinitionMetadata
  deriving BEq, DecidableEq, Repr

def ScenarioCheckContext.ofTarget
    (target : CheckedModel LawStatement Setup State Action Outcome Observation) :
    ScenarioCheckContext := {
  definitions := target.definitions
}

inductive ScenarioStatus where
  | unclassified
  | unsatisfiable
  deriving BEq, DecidableEq, Ord, Repr

def ScenarioStatus.name : ScenarioStatus → String
  | .unclassified => "unclassified"
  | .unsatisfiable => "unsatisfiable"

structure CheckedScenario where
  id : DefinitionId
  source : SourceLocation
  version : Nat
  requires : List DefinitionId
  roles : List Scenario.Role
  setup : List SetupConstraint
  allowedActions : List DefinitionId
  requiredOccurrences : List Scenario.Step
  forbiddenActions : List DefinitionId
  occurrenceBounds : List Scenario.Count
  ordering : List Scenario.Order
  sequences : List (List DefinitionId)
  adjacencies : List (List DefinitionId)
  actionsExactly : Option (List DefinitionId)
  traceExactly : Option Scenario.Trace
  spaceStatus : ScenarioStatus
  documentation : String
  canonicalMetadata : String
  behaviorFingerprint : BehaviorFingerprint
  deriving BEq, DecidableEq, Repr


/-! Narrow ordinary-Lean constructors over the checked Scenario language. -/

/-- One keyed action position in an exact schedule. The key is local to the Scenario's family. -/
structure Scenario.Slot where
  key : String
  action : DefinitionId
  deriving BEq, DecidableEq, Repr

private def Scenario.adjacentOrders : List Scenario.Step → List Scenario.Order
  | first :: second :: rest =>
      { before := first.id, after := second.id } :: Scenario.adjacentOrders (second :: rest)
  | _ => []

/-- Replace the required occurrences and every schedule field they determine. -/
def Scenario.withSteps (scenario : Scenario) (steps : List Scenario.Step) : Scenario :=
  let actions := steps.map Scenario.Step.action
  { scenario with
    allowedActions := DefinitionId.canonicalSet actions
    requiredOccurrences := steps
    occurrenceBounds := (DefinitionId.canonicalSet actions).map fun action =>
      Scenario.Count.exactly action (actions.count action)
    ordering := Scenario.adjacentOrders steps
    actionsExactly := some actions }

/-- Author one Scenario that admits exactly the given keyed action schedule. -/
def Scenario.exactly
    (family : DefinitionFamily)
    (key : String)
    (source : SourceLocation)
    (occurrences : List Scenario.Slot)
    (requires : List DefinitionId := [])
    (roles : List Scenario.Role := [])
    (setup : List SetupConstraint := [])
    (documentation : String := "") : Scenario :=
  Scenario.withSteps
    { id := family.id "behavior" key, source, requires, roles, setup, documentation }
    (occurrences.map fun slot =>
      { id := family.id "occurrence" slot.key, action := slot.action })

/-- Existing trace constraints. Occurrence keys are local to the Scenario's definition family;
`before` and `inOrder` permit intervening allowed actions, while `adjacent` does not. -/
inductive Scenario.Constraint where
  | allow (actions : List DefinitionId)
  | forbid (actions : List DefinitionId)
  | require (key : String) (action : DefinitionId)
  | bound (bound : Scenario.Count)
  | before (first second : String)
  | inOrder (actions : List DefinitionId)
  | adjacent (actions : List DefinitionId)
  deriving BEq, DecidableEq, Repr

/-- Author one Scenario from typed trace constraints. Exact schedules and traces remain explicit;
checking these constraints never establishes that a Model can execute them. -/
def Scenario.constrained
    (family : DefinitionFamily)
    (key : String)
    (source : SourceLocation)
    (constraints : List Scenario.Constraint := [])
    (requires : List DefinitionId := [])
    (roles : List Scenario.Role := [])
    (setup : List SetupConstraint := [])
    (actionsExactly : Option (List DefinitionId) := none)
    (traceExactly : Option AuthoredExactTrace := none)
    (documentation : String := "") : Scenario :=
  constraints.foldl (fun scenario constraint =>
    match constraint with
    | .allow actions =>
        { scenario with allowedActions := scenario.allowedActions ++ actions }
    | .forbid actions =>
        { scenario with forbiddenActions := scenario.forbiddenActions ++ actions }
    | .require occurrenceKey action =>
        { scenario with requiredOccurrences := scenario.requiredOccurrences ++ [{
          id := family.id "occurrence" occurrenceKey
          action
        }] }
    | .bound bound =>
        { scenario with occurrenceBounds := scenario.occurrenceBounds ++ [bound] }
    | .before first second =>
        { scenario with ordering := scenario.ordering ++ [{
          before := family.id "occurrence" first
          after := family.id "occurrence" second
        }] }
    | .inOrder actions =>
        { scenario with sequences := scenario.sequences ++ [actions] }
    | .adjacent actions =>
        { scenario with adjacencies := scenario.adjacencies ++ [actions] }) {
    id := family.id "behavior" key
    source
    requires
    roles
    setup
    actionsExactly
    traceExactly
    documentation
  }

end Umpire
