import Umpire.Target.Tests.Authoring
import Umpire.Target.Tests.KernelSoundness
import Umpire.Target.Tests.Validation

/-! Source-located mutation coverage for the complete Target-owned error boundary. -/

namespace Umpire.TargetTests

open Umpire

def invalidIdentityTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  declarations := metadata "action" .action :: testDeclarations
}

def mismatchedLawProvider : CapabilityProvider TestLawStatement := {
  primaryProvider with
  contract := {
    primaryProvider.contract with
    requiredLaws := [{ providerLaw with semanticDigest := "provider-sound/stale" }]
  }
}

def mismatchedLawTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with providers := [mismatchedLawProvider, secondaryProvider]
}

def uncoveredCapabilityTarget : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  requiredCapabilities := [secondaryProvider.contract.id]
  providers := [primaryProvider]
  connectors := []
}

def authoredMutation
    (declaration : TargetDeclaration TestLawStatement Unit Bool Bool Bool Bool)
    (identity : DeclarationId)
    (role : AuthoringOccurrenceRole)
    (owner : DeclarationId)
    (line : Nat)
    (context : AuthoringOccurrenceContext := .direct) :
    AuthoredTarget TestLawStatement Unit Bool Bool Bool Bool :=
  authoringOf declaration (occurrences := [{
    id := occurrenceId line 2 line 20 0
    declarationId := identity
    path := { role, owner, context }
  }])

def duplicateMutation : AuthoredTarget TestLawStatement Unit Bool Bool Bool Bool :=
  authoringOf duplicateIdentityTarget (occurrences := [
    occurrence testTarget.id .declarationMetadata testTarget.id 32,
    occurrence testTarget.id .declarationMetadata testTarget.id 31
  ])

def locatedMutationSummary
    (result : Except AuthoringDiagnostic
      (CheckedTarget TestLawStatement Unit Bool Bool Bool Bool)) :
    Option (DeclarationErrorKind × AuthoringOccurrenceRole × String × Nat × Nat) :=
  match result with
  | .ok _ => none
  | .error diagnostic => some
      (diagnostic.error.kind, diagnostic.path.role, diagnostic.offending.sourcePath,
        diagnostic.offending.line, diagnostic.offending.column)

def targetMutationResults :
    List (Option (DeclarationErrorKind × AuthoringOccurrenceRole × String × Nat × Nat)) := [
  locatedMutationSummary <| checkTarget <|
    authoredMutation emptyIdentityTarget (id "") .declarationMetadata (id "") 10,
  locatedMutationSummary <| checkTarget <|
    authoredMutation invalidIdentityTarget (id "action") .declarationMetadata (id "action") 20,
  locatedMutationSummary (checkTarget duplicateMutation),
  locatedMutationSummary <| checkTarget <|
    authoredMutation unknownIdentityTarget (id "test.capability.missing")
      .capabilityRequirement testTarget.id 40,
  locatedMutationSummary <| checkTarget <|
    authoredMutation wrongKindTarget (id "test.action.request")
      .capabilityRequirement testTarget.id 50,
  locatedMutationSummary <| checkTarget <|
    authoredMutation missingLawTarget providerLaw.id .lawRequirement primaryProvider.id 60,
  locatedMutationSummary <| checkTarget <|
    authoredMutation staleWitnessTarget providerLaw.id .lawWitness primaryProvider.id 70,
  locatedMutationSummary <| checkTarget <|
    authoredMutation mismatchedLawTarget providerLaw.id .lawRequirement primaryProvider.id 80,
  locatedMutationSummary <| checkTarget <|
    authoredMutation uncoveredCapabilityTarget secondaryProvider.contract.id
      .capabilityRequirement testTarget.id 90,
  locatedMutationSummary <| checkTarget <|
    authoredMutation conflictingTarget (id "test.relation.shared") .meaning primaryProvider.id 100,
  locatedMutationSummary <| checkTarget <|
    authoredMutation ambiguousConnectorTarget (id "test.relation.shared") .reconciliation
      secondOwnershipConnector.id 110,
  locatedMutationSummary <| checkTarget <|
    authoredMutation incompleteKernelTarget testKernel.metadata.id .kernel testTarget.id 120
]

example : targetMutationResults = [
    some (.emptyIdentity, .declarationMetadata, "Test/TargetAuthoring.lean", 10, 2),
    some (.invalidIdentity, .declarationMetadata, "Test/TargetAuthoring.lean", 20, 2),
    some (.duplicateIdentity, .declarationMetadata, "Test/TargetAuthoring.lean", 32, 2),
    some (.unknownIdentity, .capabilityRequirement, "Test/TargetAuthoring.lean", 40, 2),
    some (.wrongKind, .capabilityRequirement, "Test/TargetAuthoring.lean", 50, 2),
    some (.missingLaw, .lawRequirement, "Test/TargetAuthoring.lean", 60, 2),
    some (.unexpectedLaw, .lawWitness, "Test/TargetAuthoring.lean", 70, 2),
    some (.lawContractMismatch, .lawRequirement, "Test/TargetAuthoring.lean", 80, 2),
    some (.missingProvider, .capabilityRequirement, "Test/TargetAuthoring.lean", 90, 2),
    some (.conflictingProviders, .meaning, "Test/TargetAuthoring.lean", 100, 2),
    some (.ambiguousConnector, .reconciliation, "Test/TargetAuthoring.lean", 110, 2),
    some (.incompleteKernel, .kernel, "Test/TargetAuthoring.lean", 120, 2)
  ] := by
  native_decide

end Umpire.TargetTests
