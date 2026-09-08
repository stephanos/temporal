import Temporal.Feature.Nexus.Lifecycle
import Temporal.Shared
import Umpire.Property.Authoring
import Umpire.Query.Authoring

/-! Shared declaration mechanics behind the ordinary Nexus operation walkthroughs. -/

namespace Temporal.Feature.Nexus.Operations

open Umpire
open Temporal.Feature.Nexus.Lifecycle

namespace Internal

def id (value : String) : DefinitionId := Temporal.Shared.definitionId value

def family : DefinitionFamily :=
  Temporal.Shared.definitionFamily "nexus.basic-lifecycle"

/-- The ordinary operation bound names each existing Query stage and converts once by record
assembly. Source construction likewise delegates once to `Temporal.Shared.sourceLocation`; neither
path scans declarations or invokes a checker. -/
def queryLimitSpec : QueryLimitSpec := {
  transitions := 1
  selectedActions := 1
  candidateEvaluations := 8
}

/-- Preserve the established observation-clause identities while reusing the shared transition
result constructor. The shared constructor names the fact clause with a `fact` suffix, whereas
these declarations predate it and publish `observation` in their canonical metadata. -/
def operationTransitionResultClauses
    (propertyKey : String)
    (action state outcome fact : ModelValue) : List PropertyClause :=
  (transitionResultClauses family propertyKey action state outcome fact).map fun clause =>
    match clause with
    | .inputOutput _ input output =>
        .inputOutput (family.id "property" (propertyKey ++ ".observation")) input output
    | clause => clause

end Internal

def source : SourceLocation :=
  Temporal.Shared.sourceLocation "Temporal/Feature/Nexus/Operations.lean"

def operationRole : ResourceRole := { id := operationRoleId, valueKind := .state }

namespace Internal

def queryDeclaration
    (queryId : DefinitionId)
    (property : CheckedProperty)
    (behavior : CheckedBehavior) : QueryDeclaration := {
  id := queryId
  source
  target := target.id
  form := .witness property
  behavior
  limits := queryLimitSpec.toQueryLimits
  policy
}

def querySpec
    (key : String)
    (property : CheckedProperty)
    (behavior : CheckedBehavior) : QuerySpec := {
  family
  key
  source
  target := target.id
  form := .witness property
  behavior
  limits := queryLimitSpec
  policy
}

end Internal

end Temporal.Feature.Nexus.Operations
