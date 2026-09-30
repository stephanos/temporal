import Umpire.Model.Tests.Authoring
import Umpire.Model.Tests.KernelSoundness
import Umpire.Model.Tests.Validation

/-! Source-located mutation coverage for the complete Target-owned error boundary. -/

namespace Umpire.ModelTests

open Umpire

def invalidDefinitionIdTarget : ModelSpec TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  definitions := metadata "action" .action :: testDefinitions
}

def mismatchedLawProvider : Provider TestLawStatement := {
  primaryProvider with
  contract := {
    primaryProvider.contract with
    requiredLaws := [{ providerLaw with body := "provider-sound/stale" }]
  }
}

def mismatchedLawTarget : ModelSpec TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with providers := [mismatchedLawProvider, secondaryProvider]
}

def uncoveredCapabilityTarget : ModelSpec TestLawStatement Unit Bool Bool Bool Bool := {
  testTarget with
  requiredCapabilities := [secondaryProvider.contract.id]
  providers := [primaryProvider]
  connectors := []
}

def authoredMutation
    (declaration : ModelSpec TestLawStatement Unit Bool Bool Bool Bool)
    (definitionId : DefinitionId)
    (role : SourceRefRole)
    (owner : DefinitionId)
    (line : Nat)
    (context : SourceRefContext := .direct) :
    DraftModel TestLawStatement Unit Bool Bool Bool Bool :=
  authoringOf declaration (occurrences := [{
    id := occurrenceId line 2 line 20 0
    definitionId
    path := { role, owner, context }
  }])

def duplicateMutation : DraftModel TestLawStatement Unit Bool Bool Bool Bool :=
  authoringOf duplicateDefinitionIdTarget (occurrences := [
    occurrence testTarget.id .definitionMetadata testTarget.id 32,
    occurrence testTarget.id .definitionMetadata testTarget.id 31
  ])

def locatedMutationSummary
    (result : Except LocatedError
      (CheckedModel TestLawStatement Unit Bool Bool Bool Bool)) :
    Option (DefinitionErrorKind × SourceRefRole × String × Nat × Nat) :=
  match result with
  | .ok _ => none
  | .error diagnostic => some
      (diagnostic.error.kind, diagnostic.path.role, diagnostic.offending.sourcePath,
        diagnostic.offending.line, diagnostic.offending.column)

def targetMutationResults :
    List (Option (DefinitionErrorKind × SourceRefRole × String × Nat × Nat)) := [
  locatedMutationSummary <| checkModel <|
    authoredMutation emptyDefinitionIdTarget (id "") .definitionMetadata (id "") 10,
  locatedMutationSummary <| checkModel <|
    authoredMutation invalidDefinitionIdTarget (id "action") .definitionMetadata (id "action") 20,
  locatedMutationSummary (checkModel duplicateMutation),
  locatedMutationSummary <| checkModel <|
    authoredMutation unknownDefinitionIdTarget (id "test.capability.missing")
      .capabilityRequirement testTarget.id 40,
  locatedMutationSummary <| checkModel <|
    authoredMutation wrongKindTarget (id "test.action.request")
      .capabilityRequirement testTarget.id 50,
  locatedMutationSummary <| checkModel <|
    authoredMutation missingLawTarget providerLaw.id .lawRequirement primaryProvider.id 60,
  locatedMutationSummary <| checkModel <|
    authoredMutation staleWitnessTarget providerLaw.id .lawProof primaryProvider.id 70,
  locatedMutationSummary <| checkModel <|
    authoredMutation mismatchedLawTarget providerLaw.id .lawRequirement primaryProvider.id 80,
  locatedMutationSummary <| checkModel <|
    authoredMutation uncoveredCapabilityTarget secondaryProvider.contract.id
      .capabilityRequirement testTarget.id 90,
  locatedMutationSummary <| checkModel <|
    authoredMutation conflictingTarget (id "test.relation.shared") .meaning primaryProvider.id 100,
  locatedMutationSummary <| checkModel <|
    authoredMutation ambiguousConnectorTarget (id "test.relation.shared") .reconciliation
      secondOwnershipConnector.id 110,
  locatedMutationSummary <| checkModel <|
    authoredMutation incompleteKernelTarget testKernel.metadata.id .machine testTarget.id 120
]

example : targetMutationResults = [
    some (.emptyDefinitionId, .definitionMetadata, "Test/ModelAuthoring.lean", 10, 2),
    some (.invalidDefinitionId, .definitionMetadata, "Test/ModelAuthoring.lean", 20, 2),
    some (.duplicateDefinitionId, .definitionMetadata, "Test/ModelAuthoring.lean", 32, 2),
    some (.unknownDefinitionId, .capabilityRequirement, "Test/ModelAuthoring.lean", 40, 2),
    some (.wrongKind, .capabilityRequirement, "Test/ModelAuthoring.lean", 50, 2),
    some (.missingLaw, .lawRequirement, "Test/ModelAuthoring.lean", 60, 2),
    some (.unexpectedLaw, .lawProof, "Test/ModelAuthoring.lean", 70, 2),
    some (.lawContractMismatch, .lawRequirement, "Test/ModelAuthoring.lean", 80, 2),
    some (.missingProvider, .capabilityRequirement, "Test/ModelAuthoring.lean", 90, 2),
    some (.conflictingProviders, .meaning, "Test/ModelAuthoring.lean", 100, 2),
    some (.ambiguousConnector, .reconciliation, "Test/ModelAuthoring.lean", 110, 2),
    some (.incompleteMachine, .machine, "Test/ModelAuthoring.lean", 120, 2)
  ] := by
  native_decide

end Umpire.ModelTests
