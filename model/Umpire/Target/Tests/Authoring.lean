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

def reusedIdentityAuthoring : AuthoredTarget TestLawStatement Unit Bool Bool Bool Bool := {
  declaration := reusedIdentityTarget
  occurrences := [
    occurrence primaryProvider.id .providerDefinition testTarget.id 30,
    occurrence primaryProvider.id .capabilityRequirement primaryProvider.id 50,
    occurrence primaryProvider.id .declarationMetadata primaryProvider.id 10
  ]
}

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

def duplicateAuthoring : AuthoredTarget TestLawStatement Unit Bool Bool Bool Bool := {
  declaration := duplicateMetadataTarget
  occurrences := duplicateOccurrences
}

def reorderedDuplicateAuthoring : AuthoredTarget TestLawStatement Unit Bool Bool Bool Bool := {
  declaration := reorderedDuplicateMetadataTarget
  occurrences := duplicateOccurrences.reverse
}

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

def finitePlanning : FinitePlanningCapability testKernel := {
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

def finitePlanningAuthoring : AuthoredTarget TestLawStatement Unit Bool Bool Bool Bool := {
  declaration := testTarget
  planning := .available testKernel rfl finitePlanning
}

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

example : planningSummary (checkTarget ({ declaration := testTarget } :
    AuthoredTarget TestLawStatement Unit Bool Bool Bool Bool)) = some none := by
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

def movedLayoutAuthoring : AuthoredTarget TestLawStatement Unit Bool Bool Bool Bool := {
  declaration := testTarget
  occurrences := [occurrence testTarget.id .targetDeclaration testTarget.id 400]
}

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
error: target authoring failed: {"kind":"wrong-kind"
-/
#guard_msgs (error, substring := true) in
#check rejectedTarget%

end Umpire.TargetTests
