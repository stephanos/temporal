import Umpire.Target.Tests.Fixtures

/-! Checked authoring diagnostics and optional finite-planning capability tests. -/

namespace Umpire.TargetTests

open Umpire

def occurrenceId
    (line column endLine endColumn localOrdinal : Nat) : AuthoringOccurrenceId := {
  sourcePath := "Test/TargetAuthoring.lean"
  line
  column
  endLine
  endColumn
  localOrdinal
}

def occurrence
    (identity : DeclarationId)
    (role : AuthoringOccurrenceRole)
    (owner : DeclarationId)
    (line : Nat)
    (localOrdinal : Nat := 0) : AuthoringOccurrence := {
  id := occurrenceId line 2 line 20 localOrdinal
  declarationId := identity
  path := { role, owner }
}

def reusedIdentityProvider : CapabilityProvider TestLawStatement := {
  primaryProvider with
  contract := { primaryProvider.contract with id := primaryProvider.id }
}

def reusedIdentityTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with providers := [reusedIdentityProvider, secondaryProvider]
}

def reusedIdentityAuthoring : AuthoredTarget TestLawStatement Unit Bool Bool Bool Bool :=
  authoringOf reusedIdentityTarget (occurrences := [
    occurrence primaryProvider.id .providerDefinition testTarget.id 30,
    occurrence primaryProvider.id .capabilityRequirement primaryProvider.id 50,
    occurrence primaryProvider.id .declarationMetadata primaryProvider.id 10
  ])

def diagnosticSummary
    (result : Except AuthoringDiagnostic Target) :
    Option (DeclarationErrorKind × AuthoringOccurrenceRole × String × Nat) :=
  match result with
  | .ok _ => none
  | .error diagnostic => some
      (diagnostic.error.kind, diagnostic.path.role, diagnostic.offending.sourcePath,
        diagnostic.offending.line)

example : diagnosticSummary (checkTarget reusedIdentityAuthoring) =
    some (.wrongKind, .capabilityRequirement, "Test/TargetAuthoring.lean", 50) := by
  native_decide

def declarationsWithKind
    (identity : DeclarationId)
    (kind : DeclarationKind) : List DeclarationMetadata :=
  testDeclarations.map fun declaration =>
    if declaration.id == identity then { declaration with kind } else declaration

def wrongProviderDefinitionTarget :
    TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with declarations := declarationsWithKind primaryProvider.id .connector
}

def wrongConnectorDefinitionTarget :
    TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with declarations := declarationsWithKind ownershipConnector.id .provider
}

def wrongProviderDefinitionAuthoring :
    AuthoredTarget TestLawStatement Unit Bool Bool Bool Bool :=
  authoringOf wrongProviderDefinitionTarget (occurrences := [
    occurrence primaryProvider.id .providerDefinition testTarget.id 60
  ])

def wrongConnectorDefinitionAuthoring :
    AuthoredTarget TestLawStatement Unit Bool Bool Bool Bool :=
  authoringOf wrongConnectorDefinitionTarget (occurrences := [
    occurrence ownershipConnector.id .connectorDefinition testTarget.id 70
  ])

example : [
    diagnosticSummary (checkTarget wrongProviderDefinitionAuthoring),
    diagnosticSummary (checkTarget wrongConnectorDefinitionAuthoring)
  ] = [
    some (.wrongKind, .providerDefinition, "Test/TargetAuthoring.lean", 60),
    some (.wrongKind, .connectorDefinition, "Test/TargetAuthoring.lean", 70)
  ] := by
  native_decide

def inactiveProviderId : DeclarationId := id "test.provider.inactive"
def alphaRelationId : DeclarationId := id "test.relation.alpha"
def omegaRelationId : DeclarationId := id "test.relation.omega"

def repeatedProviderReferenceConnector : CapabilityConnector TestLawStatement := {
  ownershipConnector with
  reconciliations := [
    {
      declaration := omegaRelationId
      kind := .relation
      providers := [inactiveProviderId]
      semanticDigest := "test-omega-reconciliation/v1"
    },
    {
      declaration := alphaRelationId
      kind := .relation
      providers := [inactiveProviderId]
      semanticDigest := "test-alpha-reconciliation/v1"
    }
  ]
}

def repeatedProviderReferenceTarget :
    TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  declarations := [
    metadata inactiveProviderId.value .provider,
    metadata alphaRelationId.value .relation,
    metadata omegaRelationId.value .relation
  ] ++ testDeclarations
  connectors := [repeatedProviderReferenceConnector]
}

def reconciliationProviderOccurrence
    (reconciliation : DeclarationId)
    (line : Nat) : AuthoringOccurrence := {
  id := occurrenceId line 2 line 20 0
  declarationId := inactiveProviderId
  path := {
    role := .providerReference
    owner := ownershipConnector.id
    context := .reconciliation reconciliation
  }
}

def repeatedProviderReferenceAuthoring :
    AuthoredTarget TestLawStatement Unit Bool Bool Bool Bool :=
  authoringOf repeatedProviderReferenceTarget (occurrences := [
    reconciliationProviderOccurrence omegaRelationId 10,
    reconciliationProviderOccurrence alphaRelationId 90
  ])

def nestedDiagnosticSummary
    (result : Except AuthoringDiagnostic Target) :
    Option (AuthoringOccurrenceContext × Nat) :=
  match result with
  | .ok _ => none
  | .error diagnostic => some (diagnostic.path.context, diagnostic.offending.line)

example : nestedDiagnosticSummary (checkTarget repeatedProviderReferenceAuthoring) =
    some (.reconciliation alphaRelationId, 90) := by
  native_decide

def duplicateMetadataTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  declarations := metadata testTarget.id.value .target :: testDeclarations
}

def reorderedDuplicateMetadataTarget :
    TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  duplicateMetadataTarget with declarations := duplicateMetadataTarget.declarations.reverse
}

def duplicateOccurrences : List AuthoringOccurrence := [
  occurrence testTarget.id .declarationMetadata testTarget.id 40,
  occurrence testTarget.id .declarationMetadata testTarget.id 10
]

def duplicateAuthoring : AuthoredTarget TestLawStatement Unit Bool Bool Bool Bool :=
  authoringOf duplicateMetadataTarget (occurrences := duplicateOccurrences)

def reorderedDuplicateAuthoring : AuthoredTarget TestLawStatement Unit Bool Bool Bool Bool :=
  authoringOf reorderedDuplicateMetadataTarget (occurrences := duplicateOccurrences.reverse)

def duplicateSummary
    (result : Except AuthoringDiagnostic Target) : Option (Nat × Nat) :=
  match result with
  | .ok _ => none
  | .error diagnostic =>
      diagnostic.original.map fun original => (original.line, diagnostic.offending.line)

example : [
    duplicateSummary (checkTarget duplicateAuthoring),
    duplicateSummary (checkTarget reorderedDuplicateAuthoring)
  ] = [some (10, 40), some (10, 40)] := by
  native_decide

def finitePlanning : FinitePlanningCapability testKernel.authoritativeStep := {
  actions := [false, true]
  roleDomainDigest := "test-role-domain/v1"
  actionDomainDigest := "test-action-domain/v1"
  actionSound := by
    intro action _
    exact ⟨false, transition false action, rfl⟩
  actionComplete := by
    intro _ action _ _
    cases action <;> simp
}

def finitePlanningAuthoring : AuthoredTarget TestLawStatement Unit Bool Bool Bool Bool :=
  authoringOf testTarget (.available testKernel rfl finitePlanning)

def planningSummary
    (result : Except AuthoringDiagnostic
      (CheckedTarget TestLawStatement Unit Bool Bool Bool Bool)) :
    Option (Option (List Bool × String × String)) :=
  match result with
  | .error _ => none
  | .ok checked =>
      some <| match checked.planning with
        | .unavailable => none
        | .available capability => some
            (capability.actions, capability.roleDomainDigest, capability.actionDomainDigest)

example : planningSummary (checkTarget (authoringOf testTarget)) = some none := by
  native_decide

example : planningSummary (checkTarget finitePlanningAuthoring) =
    some (some ([false, true], "test-role-domain/v1", "test-action-domain/v1")) := by
  native_decide

def checkedSemanticSummary
    (result : Except Error (CheckedTarget TestLawStatement Unit Bool Bool Bool Bool)) :
    Option (String × String) :=
  match result with
  | .error _ => none
  | .ok checked => some (checked.canonicalMetadata, checked.semanticDigest)

def movedLayoutAuthoring : AuthoredTarget TestLawStatement Unit Bool Bool Bool Bool :=
  authoringOf testTarget (occurrences := [
    occurrence testTarget.id .targetDeclaration testTarget.id 400
  ])

example : [
    checkedSemanticSummary (composeTarget testTarget),
    checkedSemanticSummary (checkTarget movedLayoutAuthoring),
    checkedSemanticSummary (checkTarget finitePlanningAuthoring)
  ].all (· == checkedSemanticSummary (composeTarget testTarget)) = true := by
  native_decide

#check Umpire.captureAuthoringOccurrence
#check Umpire.elaborateTarget

elab "rejectedTarget%" : term => do
  let reference ← Lean.getRef
  let captured ← captureAuthoringOccurrence reference primaryProvider.id {
    role := .capabilityRequirement
    owner := primaryProvider.id
  } 0
  let _ ← elaborateTarget reusedIdentityAuthoring [captured]
  Lean.Elab.Term.elabTerm (← `(true)) none

/--
error: target authoring failed: {"error":{"kind":"wrong-kind"
-/
#guard_msgs (error, substring := true) in
#check rejectedTarget%

elab "rejectedDuplicateTarget%" original:ident offending:ident : term => do
  let path : AuthoringOccurrencePath := {
    role := .declarationMetadata
    owner := testTarget.id
  }
  let original ← captureAuthoringOccurrence original testTarget.id path 0
  let offending ← captureAuthoringOccurrence offending testTarget.id path 1
  let _ ← elaborateTarget duplicateAuthoring [offending, original]
  Lean.Elab.Term.elabTerm (← `(true)) none

/--
error: target authoring failed: {"error":{"kind":"duplicate-identity","declarationId":"test.target.composed","sourcePath":"Umpire/TargetTests.lean","offendingValue":"test.target.composed","relatedIdentities":["test.target.composed"]},"original":{"sourcePath":
-/
#guard_msgs (error, substring := true) in
#check rejectedDuplicateTarget% originalOccurrence offendingOccurrence

end Umpire.TargetTests
