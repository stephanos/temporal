import Umpire.Planning.Types

/-! Checked environment-independent references needed to hand an Artifact to Execution. -/

namespace Umpire

/-- Authored references that complete a selected model trace for downstream Execution binding. -/
structure ExecutionHandoffDeclaration where
  participantProgramDefinitionIds : List DefinitionId
  setupDefinitionIds : List DefinitionId
  orderingDefinitionIds : List DefinitionId
  terminationDefinitionIds : List DefinitionId
  cleanupDefinitionIds : List DefinitionId
  deriving BEq, DecidableEq, Repr

/-- Canonical execution references checked against the selected Artifact where model-owned. -/
structure ExecutionHandoff where
  private mk ::
  participantProgramDefinitionIds : List DefinitionId
  setupDefinitionIds : List DefinitionId
  orderingDefinitionIds : List DefinitionId
  terminationDefinitionIds : List DefinitionId
  cleanupDefinitionIds : List DefinitionId
  deriving BEq, DecidableEq, Repr

inductive ExecutionHandoffErrorKind where
  | invalidDefinitionId
  | missingReference
  | duplicateReference
  | unknownSetupReference
  | unknownOrderingReference
  | unknownTerminationReference
  | artifactIdentityDrift
  deriving BEq, DecidableEq, Ord, Repr

def ExecutionHandoffErrorKind.name : ExecutionHandoffErrorKind → String
  | .invalidDefinitionId => "invalid-definition-id"
  | .missingReference => "missing-reference"
  | .duplicateReference => "duplicate-reference"
  | .unknownSetupReference => "unknown-setup-reference"
  | .unknownOrderingReference => "unknown-ordering-reference"
  | .unknownTerminationReference => "unknown-termination-reference"
  | .artifactIdentityDrift => "artifact-identity-drift"

structure ExecutionHandoffError where
  kind : ExecutionHandoffErrorKind
  category : String
  definitionId : DefinitionId
  relatedDefinitionIds : List DefinitionId
  deriving BEq, DecidableEq, Repr

private def idLe (left right : DefinitionId) : Bool :=
  decide (left.value ≤ right.value)

private def canonicalIds (ids : List DefinitionId) : List DefinitionId :=
  ids.mergeSort idLe

private def firstDuplicate : List DefinitionId → Option DefinitionId
  | first :: second :: rest =>
      if first == second then some second else firstDuplicate (second :: rest)
  | _ => none

private def error
    (kind : ExecutionHandoffErrorKind)
    (category : String)
    (definitionId : DefinitionId)
    (relatedDefinitionIds : List DefinitionId := []) : ExecutionHandoffError := {
  kind
  category
  definitionId
  relatedDefinitionIds := canonicalIds relatedDefinitionIds |>.eraseDups
}

private def checkReferences
    (category : String)
    (ids : List DefinitionId) : Except ExecutionHandoffError (List DefinitionId) := do
  if ids.isEmpty then
    throw (error .missingReference category (DefinitionId.of ("umpire.execution." ++ category)))
  let canonical := canonicalIds ids
  for id in canonical do
    if !id.isNamespaced then
      throw (error .invalidDefinitionId category id)
  match firstDuplicate canonical with
  | some duplicate => throw (error .duplicateReference category duplicate [duplicate])
  | none => pure canonical

private def checkOwnedReferences
    (kind : ExecutionHandoffErrorKind)
    (category : String)
    (available requested : List DefinitionId) : Except ExecutionHandoffError Unit := do
  for id in requested do
    if !available.contains id then
      throw (error kind category id available)

/--
Check the closed handoff categories. Participant programs and cleanup obligations are downstream
bindings; setup, ordering, and termination must already be present in the selected model Artifact.
-/
def checkExecutionHandoff
    (availableSetupDefinitionIds : List DefinitionId)
    (availableOrderingDefinitionIds : List DefinitionId)
    (availableTerminationDefinitionIds : List DefinitionId)
    (declaration : ExecutionHandoffDeclaration) :
    Except ExecutionHandoffError ExecutionHandoff := do
  let participants ← checkReferences "participant-program"
    declaration.participantProgramDefinitionIds
  let setup ← checkReferences "setup" declaration.setupDefinitionIds
  let ordering ← checkReferences "ordering" declaration.orderingDefinitionIds
  let termination ← checkReferences "termination" declaration.terminationDefinitionIds
  let cleanup ← checkReferences "cleanup" declaration.cleanupDefinitionIds
  checkOwnedReferences .unknownSetupReference "setup" availableSetupDefinitionIds setup
  checkOwnedReferences .unknownOrderingReference "ordering" availableOrderingDefinitionIds ordering
  checkOwnedReferences .unknownTerminationReference "termination"
    availableTerminationDefinitionIds termination
  pure {
    participantProgramDefinitionIds := participants
    setupDefinitionIds := setup
    orderingDefinitionIds := ordering
    terminationDefinitionIds := termination
    cleanupDefinitionIds := cleanup
  }

def ExecutionHandoff.allDefinitionIds (handoff : ExecutionHandoff) : List DefinitionId :=
  canonicalIds (handoff.participantProgramDefinitionIds ++ handoff.setupDefinitionIds ++
    handoff.orderingDefinitionIds ++ handoff.terminationDefinitionIds ++
    handoff.cleanupDefinitionIds) |>.eraseDups

end Umpire
