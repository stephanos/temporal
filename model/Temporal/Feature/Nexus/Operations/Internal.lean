import Temporal.Feature.Nexus.Lifecycle
import Temporal.Shared
import Umpire.Property.Elab
import Umpire.Query.Elab

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
def queryLimits : Limits := Limits.bounded 1 1 8

/-- Preserve the established observation-clause identities while reusing the shared transition
result constructor. The shared constructor names the fact clause with a `fact` suffix, whereas
these declarations predate it and publish `observation` in their canonical metadata. -/
def operationStepClauses
    (propertyKey : String)
    (action state outcome fact : ModelValue) : List PropertyClause :=
  (stepClauses family propertyKey action state outcome fact).map fun clause =>
    match clause with
    | .inputOutput _ input output =>
        .inputOutput (family.id "property" (propertyKey ++ ".observation")) input output
    | clause => clause

end Internal

def source : SourceLocation :=
  Temporal.Shared.sourceLocation "Temporal/Feature/Nexus/Operations.lean"

def operationRole : Scenario.Role := { id := operationRoleId, valueKind := .state }

namespace Internal

def authoredQuery
    (queryId : DefinitionId)
    (property : CheckedProperty)
    (behavior : CheckedScenario) : Query := {
  id := queryId
  source
  target := target.id
  form := .find property
  behavior
  limits := queryLimits
  policy
}

def keyedQueryDeclaration
    (key : String)
    (property : CheckedProperty)
    (behavior : CheckedScenario) : Query :=
  authoredQuery (family.id "query" key) property behavior

end Internal

end Temporal.Feature.Nexus.Operations
