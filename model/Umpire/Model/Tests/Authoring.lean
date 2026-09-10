import Umpire.Model.Tests.Fixtures

/-! Checked authoring diagnostics and optional finite-planning capability tests. -/

namespace Umpire.ModelTests

open Umpire

def occurrenceId
    (line column endLine endColumn localOrdinal : Nat) : SourceSpan := {
  sourcePath := "Test/ModelAuthoring.lean"
  line
  column
  endLine
  endColumn
  localOrdinal
}

def occurrence
    (definitionId : DefinitionId)
    (role : SourceRefRole)
    (owner : DefinitionId)
    (line : Nat)
    (localOrdinal : Nat := 0) : SourceRef := {
  id := occurrenceId line 2 line 20 localOrdinal
  definitionId
  path := { role, owner }
}

def reusedDefinitionIdProvider : Provider TestLawStatement := {
  primaryProvider with
  contract := { primaryProvider.contract with id := primaryProvider.id }
}

def reusedDefinitionIdTarget : ModelSpec TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with providers := [reusedDefinitionIdProvider, secondaryProvider]
}

def reusedDefinitionIdAuthoring : DraftModel TestLawStatement Unit Bool Bool Bool Bool :=
  authoringOf reusedDefinitionIdTarget (occurrences := [
    occurrence primaryProvider.id .providerDefinition testTarget.id 30,
    occurrence primaryProvider.id .capabilityRequirement primaryProvider.id 50,
    occurrence primaryProvider.id .definitionMetadata primaryProvider.id 10
  ])

def diagnosticSummary
    (result : Except LocatedError Target) :
    Option (DefinitionErrorKind × SourceRefRole × String × Nat) :=
  match result with
  | .ok _ => none
  | .error diagnostic => some
      (diagnostic.error.kind, diagnostic.path.role, diagnostic.offending.sourcePath,
        diagnostic.offending.line)

example : diagnosticSummary (checkModel reusedDefinitionIdAuthoring) =
    some (.wrongKind, .capabilityRequirement, "Test/ModelAuthoring.lean", 50) := by
  native_decide

def definitionsWithKind
    (definitionId : DefinitionId)
    (kind : DefinitionKind) : List DefinitionMetadata :=
  testDefinitions.map fun definition =>
    if definition.id == definitionId then { definition with kind } else definition

def wrongProviderDefinitionTarget :
    ModelSpec TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with definitions := definitionsWithKind primaryProvider.id .connector
}

def wrongConnectorDefinitionTarget :
    ModelSpec TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with definitions := definitionsWithKind ownershipConnector.id .provider
}

def wrongProviderDefinitionAuthoring :
    DraftModel TestLawStatement Unit Bool Bool Bool Bool :=
  authoringOf wrongProviderDefinitionTarget (occurrences := [
    occurrence primaryProvider.id .providerDefinition testTarget.id 60
  ])

def wrongConnectorDefinitionAuthoring :
    DraftModel TestLawStatement Unit Bool Bool Bool Bool :=
  authoringOf wrongConnectorDefinitionTarget (occurrences := [
    occurrence ownershipConnector.id .connectorDefinition testTarget.id 70
  ])

example : [
    diagnosticSummary (checkModel wrongProviderDefinitionAuthoring),
    diagnosticSummary (checkModel wrongConnectorDefinitionAuthoring)
  ] = [
    some (.wrongKind, .providerDefinition, "Test/ModelAuthoring.lean", 60),
    some (.wrongKind, .connectorDefinition, "Test/ModelAuthoring.lean", 70)
  ] := by
  native_decide

def inactiveProviderId : DefinitionId := id "test.provider.inactive"
def alphaRelationId : DefinitionId := id "test.relation.alpha"
def omegaRelationId : DefinitionId := id "test.relation.omega"

def repeatedProviderReferenceConnector : Connector TestLawStatement := {
  ownershipConnector with
  reconciliations := [
    {
      definitionId := omegaRelationId
      kind := .relation
      providers := [inactiveProviderId]
      behaviorVersion := "test-omega-reconciliation/v1"
    },
    {
      definitionId := alphaRelationId
      kind := .relation
      providers := [inactiveProviderId]
      behaviorVersion := "test-alpha-reconciliation/v1"
    }
  ]
}

def repeatedProviderReferenceTarget :
    ModelSpec TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  definitions := [
    metadata inactiveProviderId.value .provider,
    metadata alphaRelationId.value .relation,
    metadata omegaRelationId.value .relation
  ] ++ testDefinitions
  connectors := [repeatedProviderReferenceConnector]
}

def reconciliationProviderOccurrence
    (reconciliation : DefinitionId)
    (line : Nat) : SourceRef := {
  id := occurrenceId line 2 line 20 0
  definitionId := inactiveProviderId
  path := {
    role := .providerReference
    owner := ownershipConnector.id
    context := .reconciliation reconciliation
  }
}

def repeatedProviderReferenceAuthoring :
    DraftModel TestLawStatement Unit Bool Bool Bool Bool :=
  authoringOf repeatedProviderReferenceTarget (occurrences := [
    reconciliationProviderOccurrence omegaRelationId 10,
    reconciliationProviderOccurrence alphaRelationId 90
  ])

def nestedDiagnosticSummary
    (result : Except LocatedError Target) :
    Option (SourceRefContext × Nat) :=
  match result with
  | .ok _ => none
  | .error diagnostic => some (diagnostic.path.context, diagnostic.offending.line)

example : nestedDiagnosticSummary (checkModel repeatedProviderReferenceAuthoring) =
    some (.reconciliation alphaRelationId, 90) := by
  native_decide

def duplicateMetadataTarget : ModelSpec TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  definitions := metadata testTarget.id.value .target :: testDefinitions
}

def reorderedDuplicateMetadataTarget :
    ModelSpec TestLawStatement Unit Bool Bool Bool Bool := {
  duplicateMetadataTarget with definitions := duplicateMetadataTarget.definitions.reverse
}

def duplicateOccurrences : List SourceRef := [
  occurrence testTarget.id .definitionMetadata testTarget.id 40,
  occurrence testTarget.id .definitionMetadata testTarget.id 10
]

def duplicateAuthoring : DraftModel TestLawStatement Unit Bool Bool Bool Bool :=
  authoringOf duplicateMetadataTarget (occurrences := duplicateOccurrences)

def reorderedDuplicateAuthoring : DraftModel TestLawStatement Unit Bool Bool Bool Bool :=
  authoringOf reorderedDuplicateMetadataTarget (occurrences := duplicateOccurrences.reverse)

def duplicateSummary
    (result : Except LocatedError Target) : Option (Nat × Nat) :=
  match result with
  | .ok _ => none
  | .error diagnostic =>
      diagnostic.original.map fun original => (original.line, diagnostic.offending.line)

example : [
    duplicateSummary (checkModel duplicateAuthoring),
    duplicateSummary (checkModel reorderedDuplicateAuthoring)
  ] = [some (10, 40), some (10, 40)] := by
  native_decide

def finitePlanning : FinitePlanningCapability testKernel.authoritativeStep := {
  actions := [false, true]
  actionSound := by
    intro action _
    exact ⟨false, transition false action, rfl⟩
  actionComplete := by
    intro _ action _ _
    cases action <;> simp
}

def finitePlanningAuthoring : DraftModel TestLawStatement Unit Bool Bool Bool Bool :=
  authoringOf testTarget (.available testKernel rfl finitePlanning)

def planningSummary
    (result : Except LocatedError
      (CheckedModel TestLawStatement Unit Bool Bool Bool Bool)) :
    Option (Option (List Bool)) :=
  match result with
  | .error _ => none
  | .ok checked =>
      some <| match checked.planning with
        | .unavailable => none
        | .available capability => some capability.actions

example : planningSummary (checkModel (authoringOf testTarget)) = some none := by
  native_decide

example : planningSummary (checkModel finitePlanningAuthoring) =
    some (some [false, true]) := by
  native_decide

def checkedSemanticSummary
    (result : Except Error (CheckedModel TestLawStatement Unit Bool Bool Bool Bool)) :
    Option (String × String) :=
  match result with
  | .error _ => none
  | .ok checked => some (checked.canonicalMetadata, checked.behaviorFingerprint.render)

def movedLayoutAuthoring : DraftModel TestLawStatement Unit Bool Bool Bool Bool :=
  authoringOf testTarget (occurrences := [
    occurrence testTarget.id .modelSpec testTarget.id 400
  ])

example : [
    checkedSemanticSummary ((checkModel (DraftModel.make testTarget) |>.mapError LocatedError.error)),
    checkedSemanticSummary (checkModel movedLayoutAuthoring),
    checkedSemanticSummary (checkModel finitePlanningAuthoring)
  ].all (· == checkedSemanticSummary ((checkModel (DraftModel.make testTarget) |>.mapError LocatedError.error))) = true := by
  native_decide

#check Umpire.captureSourceRef
#check Umpire.elabModel

elab "rejectedTarget%" : term => do
  let reference ← Lean.getRef
  let captured ← captureSourceRef reference primaryProvider.id {
    role := .capabilityRequirement
    owner := primaryProvider.id
  } 0
  let _ ← elabModel reusedDefinitionIdAuthoring [captured]
  Lean.Elab.Term.elabTerm (← `(true)) none

/--
error: target authoring failed: {"error":{"kind":"wrong-kind"
-/
#guard_msgs (error, substring := true) in
#check rejectedTarget%

elab "rejectedDuplicateTarget%" original:ident offending:ident : term => do
  let path : SourceRefPath := {
    role := .definitionMetadata
    owner := testTarget.id
  }
  let original ← captureSourceRef original testTarget.id path 0
  let offending ← captureSourceRef offending testTarget.id path 1
  let _ ← elabModel duplicateAuthoring [offending, original]
  Lean.Elab.Term.elabTerm (← `(true)) none

/--
error: target authoring failed: {"error":{"kind":"duplicate-definition-id","definitionId":"test.target.composed","sourcePath":"Umpire/ModelTests.lean","offendingValue":"test.target.composed","relatedDefinitionIds":["test.target.composed"]},"original":{"sourcePath":
-/
#guard_msgs (error, substring := true) in
#check rejectedDuplicateTarget% originalOccurrence offendingOccurrence

end Umpire.ModelTests
