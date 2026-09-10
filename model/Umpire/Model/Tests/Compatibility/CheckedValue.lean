import Umpire.Model.Tests.Fixtures

/-! Whole-value compatibility for the checked Target boundary, including kernel outputs. -/

namespace Umpire.ModelTests.Compatibility

open Umpire
open Umpire.ModelTests

private structure ProviderValue where
  id : DefinitionId
  source : SourceLocation
  contract : Capability
  meanings : List Meaning
  witnessedLaws : List Law
  deriving BEq, DecidableEq

private structure ConnectorValue where
  id : DefinitionId
  source : SourceLocation
  version : Nat
  behaviorVersion : String
  reconciliations : List Reconciliation
  requiredLaws : List Law
  witnessedLaws : List Law
  deriving BEq, DecidableEq

private structure CheckedModelValue where
  id : DefinitionId
  source : SourceLocation
  definitions : List DefinitionMetadata
  requiredCapabilities : List DefinitionId
  providers : List ProviderValue
  connectors : List ConnectorValue
  resolvedSetups : List Unit
  machineMetadata : MachineMetadata
  initialStates : List Bool
  stepResults : List (List (Step Bool Bool Bool))
  deriving BEq, DecidableEq

private def providerValue (provider : Provider TestLawStatement) : ProviderValue := {
  id := provider.id
  source := provider.source
  contract := provider.contract
  meanings := provider.meanings
  witnessedLaws := provider.lawProofs.map LawProof.definition
}

private def connectorValue (connector : Connector TestLawStatement) : ConnectorValue := {
  id := connector.id
  source := connector.source
  version := connector.version
  behaviorVersion := connector.behaviorVersion
  reconciliations := connector.reconciliations
  requiredLaws := connector.requiredLaws
  witnessedLaws := connector.lawProofs.map LawProof.definition
}

private def checkedTargetValue
    (target : CheckedModel TestLawStatement Unit Bool Bool Bool Bool) : CheckedModelValue := {
  id := target.id
  source := target.source
  definitions := target.definitions
  requiredCapabilities := target.requiredCapabilities
  providers := target.providers.map providerValue
  connectors := target.connectors.map connectorValue
  resolvedSetups := target.resolvedSetups
  machineMetadata := target.machine.metadata
  initialStates := target.machine.initialStates ()
  stepResults := [
    target.machine.steps false false,
    target.machine.steps false true,
    target.machine.steps true false,
    target.machine.steps true true
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
  source := stableSource "Umpire/ModelTests.lean"
  behaviorVersion := digest
}

private def stableProviderLaw : Law := {
  id := DefinitionId.of "umpire.law.provider-sound"
  body := "provider-sound/v1"
}

private def stableConnectorLaw : Law := {
  id := DefinitionId.of "umpire.law.connector-sound"
  body := "connector-sound/v1"
}

private def expectedCheckedTargetValue : CheckedModelValue := {
  id := DefinitionId.of "test.target.composed"
  source := stableSource "Test/CompositeSemantic.lean"
  definitions := [
    stableMetadata "test.action.request" .action,
    stableMetadata "test.capability.primary" .capability,
    stableMetadata "test.capability.secondary" .capability,
    stableMetadata "test.connector.shared" .connector,
    stableMetadata "test.kernel.transition" .machine,
    stableMetadata "test.observation.completed" .fact,
    stableMetadata "test.provider.primary" .provider,
    stableMetadata "test.provider.secondary" .provider,
    stableMetadata "test.relation.shared" .relation,
    stableMetadata "test.target.composed" .target,
    stableMetadata "umpire.law.connector-sound" .law stableConnectorLaw.body,
    stableMetadata "umpire.law.provider-sound" .law stableProviderLaw.body
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
      behaviorVersion := "test-primary-capability/v1"
      requiredLaws := [stableProviderLaw]
    }
    meanings := [{
      definitionId := DefinitionId.of "test.relation.shared"
      kind := .relation
      behaviorVersion := "test-primary-shared/v1"
    }]
    witnessedLaws := [stableProviderLaw]
  }, {
    id := DefinitionId.of "test.provider.secondary"
    source := stableSource "Test/SecondarySemantic.lean"
    contract := {
      id := DefinitionId.of "test.capability.secondary"
      behaviorVersion := "test-secondary-capability/v1"
      requiredLaws := [stableProviderLaw]
    }
    meanings := [{
      definitionId := DefinitionId.of "test.relation.shared"
      kind := .relation
      behaviorVersion := "test-secondary-shared/v1"
    }]
    witnessedLaws := [stableProviderLaw]
  }]
  connectors := [{
    id := DefinitionId.of "test.connector.shared"
    source := stableSource "Test/CompositeSemantic.lean"
    version := 1
    behaviorVersion := "test-shared-connector/v1"
    reconciliations := [{
      definitionId := DefinitionId.of "test.relation.shared"
      kind := .relation
      providers := [
        DefinitionId.of "test.provider.primary",
        DefinitionId.of "test.provider.secondary"
      ]
      behaviorVersion := "test-shared-connector/reconciled-v1"
    }]
    requiredLaws := [stableConnectorLaw]
    witnessedLaws := [stableConnectorLaw]
  }]
  resolvedSetups := [()]
  machineMetadata := {
    id := DefinitionId.of "test.kernel.transition"
    source := stableSource "Umpire/ModelTests.lean"
  }
  initialStates := [false]
  stepResults := [
    [{ outcome := false, state := false, facts := [false] }],
    [{ outcome := true, state := true, facts := [false] }],
    [{ outcome := false, state := false, facts := [true] }],
    [{ outcome := true, state := true, facts := [true] }]
  ]
}

example : ((checkModel (DraftModel.make testTarget) |>.mapError LocatedError.error)).toOption.map checkedTargetValue =
    some expectedCheckedTargetValue := by
  native_decide

end Umpire.ModelTests.Compatibility
