import Temporal.Feature.Nexus.Lifecycle
import Temporal.Shared

/-! Shared declaration mechanics behind the ordinary Nexus operation walkthroughs. -/

namespace Temporal.Feature.Nexus.Operations

open Umpire
open Temporal.Feature.Nexus.Lifecycle

namespace Internal

def id (value : String) : DefinitionId := Temporal.Shared.definitionId value

end Internal

def source : SourceLocation :=
  Temporal.Shared.sourceLocation "Temporal/Feature/Nexus/Operations.lean"

def operationRole : ResourceRole := { id := operationRoleId, valueKind := .state }

namespace Internal

def operationIs
    (constraintId : DefinitionId)
    (state : ModelValue) : SetupConstraint := {
  id := constraintId
  relation := .equal
  left := .role operationRoleId
  right := .value state
}

def checkBehaviorDeclaration
    (declaration : BehaviorDeclaration) : Except BehaviorError CheckedBehavior :=
  checkBehavior (.ofTarget target) declaration

def queryDeclaration
    (queryId : DefinitionId)
    (property : CheckedProperty)
    (behavior : CheckedBehavior) : QueryDeclaration := {
  id := queryId
  source
  target := target.id
  form := .witness property
  behavior
  limits
  policy
}

end Internal

end Temporal.Feature.Nexus.Operations
