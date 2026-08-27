import Umpire.Target.Tests.Fixtures

/-! Whole-value compatibility for the checked Target boundary, including kernel outputs. -/

namespace Umpire.TargetTests.Compatibility

open Umpire
open Umpire.TargetTests

private structure ProviderValue where
  id : DefinitionId
  source : SourceLocation
  contract : CapabilityContract
  meanings : List MeaningProvision
  witnessedLaws : List LawRequirement
  deriving BEq, DecidableEq

private structure ConnectorValue where
  id : DefinitionId
  source : SourceLocation
  version : Nat
  semanticDigest : String
  reconciliations : List Reconciliation
  requiredLaws : List LawRequirement
  witnessedLaws : List LawRequirement
  deriving BEq, DecidableEq

private structure CheckedTargetValue where
  id : DefinitionId
  source : SourceLocation
  definitions : List DefinitionMetadata
  requiredCapabilities : List DefinitionId
  providers : List ProviderValue
  connectors : List ConnectorValue
  resolvedSetups : List Unit
  kernelMetadata : KernelMetadata
  initialStates : List Bool
  stepResults : List (List (TransitionResult Bool Bool Bool))
  deriving BEq, DecidableEq

private def providerValue (provider : CapabilityProvider TestLawStatement) : ProviderValue := {
  id := provider.id
  source := provider.source
  contract := provider.contract
  meanings := provider.meanings
  witnessedLaws := provider.lawWitnesses.map LawWitness.requirement
}

private def connectorValue (connector : CapabilityConnector TestLawStatement) : ConnectorValue := {
  id := connector.id
  source := connector.source
  version := connector.version
  semanticDigest := connector.semanticDigest
  reconciliations := connector.reconciliations
  requiredLaws := connector.requiredLaws
  witnessedLaws := connector.lawWitnesses.map LawWitness.requirement
}

private def checkedTargetValue
    (target : CheckedTarget TestLawStatement Unit Bool Bool Bool Bool) : CheckedTargetValue := {
  id := target.id
  source := target.source
  definitions := target.definitions
  requiredCapabilities := target.requiredCapabilities
  providers := target.providers.map providerValue
  connectors := target.connectors.map connectorValue
  resolvedSetups := target.resolvedSetups
  kernelMetadata := target.kernel.metadata
  initialStates := target.kernel.initialStates ()
  stepResults := [
    target.kernel.steps false false,
    target.kernel.steps false true,
    target.kernel.steps true false,
    target.kernel.steps true true
  ]
}

private def stableSource (path : String) : SourceLocation := {
  path
  line := 1
  column := 1
  provenance := "lean-test"
}

private def stableMetadata
    (value : String)
    (kind : DefinitionKind)
    (digest : String := "contract-v1") : DefinitionMetadata := {
  id := DefinitionId.of value
  kind
  source := stableSource "Umpire/TargetTests.lean"
  contractDigest := digest
}

private def stableProviderLaw : LawRequirement := {
  id := DefinitionId.of "umpire.law.provider-sound"
  semanticDigest := "provider-sound/v1"
}

private def stableConnectorLaw : LawRequirement := {
  id := DefinitionId.of "umpire.law.connector-sound"
  semanticDigest := "connector-sound/v1"
}

private def expectedCheckedTargetValue : CheckedTargetValue := {
  id := DefinitionId.of "test.target.composed"
  source := stableSource "Test/CompositeSemantic.lean"
  definitions := [
    stableMetadata "test.action.request" .action,
    stableMetadata "test.capability.primary" .capability,
    stableMetadata "test.capability.secondary" .capability,
    stableMetadata "test.connector.shared" .connector,
    stableMetadata "test.kernel.transition" .kernel,
    stableMetadata "test.observation.completed" .observation,
    stableMetadata "test.provider.primary" .provider,
    stableMetadata "test.provider.secondary" .provider,
    stableMetadata "test.relation.shared" .relation,
    stableMetadata "test.target.composed" .target,
    stableMetadata "umpire.law.connector-sound" .law stableConnectorLaw.semanticDigest,
    stableMetadata "umpire.law.provider-sound" .law stableProviderLaw.semanticDigest
  ]
  requiredCapabilities := [
    DefinitionId.of "test.capability.primary",
    DefinitionId.of "test.capability.secondary"
  ]
  providers := [{
    id := DefinitionId.of "test.provider.primary"
    source := stableSource "Test/PrimarySemantic.lean"
    contract := {
      id := DefinitionId.of "test.capability.primary"
      semanticDigest := "test-primary-capability/v1"
      requiredLaws := [stableProviderLaw]
    }
    meanings := [{
      definitionId := DefinitionId.of "test.relation.shared"
      kind := .relation
      semanticDigest := "test-primary-shared/v1"
    }]
    witnessedLaws := [stableProviderLaw]
  }, {
    id := DefinitionId.of "test.provider.secondary"
    source := stableSource "Test/SecondarySemantic.lean"
    contract := {
      id := DefinitionId.of "test.capability.secondary"
      semanticDigest := "test-secondary-capability/v1"
      requiredLaws := [stableProviderLaw]
    }
    meanings := [{
      definitionId := DefinitionId.of "test.relation.shared"
      kind := .relation
      semanticDigest := "test-secondary-shared/v1"
    }]
    witnessedLaws := [stableProviderLaw]
  }]
  connectors := [{
    id := DefinitionId.of "test.connector.shared"
    source := stableSource "Test/CompositeSemantic.lean"
    version := 1
    semanticDigest := "test-shared-connector/v1"
    reconciliations := [{
      definitionId := DefinitionId.of "test.relation.shared"
      kind := .relation
      providers := [
        DefinitionId.of "test.provider.primary",
        DefinitionId.of "test.provider.secondary"
      ]
      semanticDigest := "test-shared-connector/reconciled-v1"
    }]
    requiredLaws := [stableConnectorLaw]
    witnessedLaws := [stableConnectorLaw]
  }]
  resolvedSetups := [()]
  kernelMetadata := {
    id := DefinitionId.of "test.kernel.transition"
    contractDigest := "test-kernel/v1"
    source := stableSource "Umpire/TargetTests.lean"
  }
  initialStates := [false]
  stepResults := [
    [{ modelOutcome := false, resultingState := false, observations := [false] }],
    [{ modelOutcome := true, resultingState := true, observations := [false] }],
    [{ modelOutcome := false, resultingState := false, observations := [true] }],
    [{ modelOutcome := true, resultingState := true, observations := [true] }]
  ]
}

example : (composeTarget testTarget).toOption.map checkedTargetValue =
    some expectedCheckedTargetValue := by
  native_decide

end Umpire.TargetTests.Compatibility
