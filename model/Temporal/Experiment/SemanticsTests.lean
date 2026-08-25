import Temporal.Experiment.Semantics

namespace Temporal.Experiment.SemanticsTests

open Temporal.Experiment

def id (value : String) : DeclarationId := DeclarationId.of value

def source (path : String) : SemanticSource := {
  path
  line := 1
  column := 1
  provenance := "lean-test"
}

def metadata
    (value : String)
    (kind : DeclarationKind)
    (digest : String := "contract-v1") : DeclarationMetadata := {
  id := id value
  kind
  source := source "Temporal/Experiment/SemanticsTests.lean"
  contractDigest := digest
}

def providerLaw : LawRequirement := {
  id := id "umpire.law.provider-sound"
  semanticDigest := "provider-sound/v1"
}

def connectorLaw : LawRequirement := {
  id := id "umpire.law.connector-sound"
  semanticDigest := "connector-sound/v1"
}

def witness (requirement : LawRequirement) : LawWitness := {
  requirement
  statement := True
  proof := True.intro
}

def transition (state action : Bool) : TransitionResult Bool Bool Bool := {
  modelOutcome := action
  resultingState := action
  observations := [state]
}

def workflowKernel : TransitionKernel Unit Bool Bool Bool Bool := {
  metadata := {
    id := id "workflow-nexus.kernel.transition"
    contractDigest := "workflow-nexus-kernel/v1"
    source := source "Temporal/Experiment/SemanticsTests.lean"
  }
  initialStates := fun _ => [false]
  authoritativeInitial := fun _ state => state = false
  initialSound := by simp
  initialComplete := by simp_all
  steps := fun state action => [transition state action]
  authoritativeStep := fun state action result => result = transition state action
  stepSound := by simp
  stepComplete := by simp_all
}

def workflowProvider : CapabilityProvider := {
  id := id "workflow.provider.lifecycle"
  source := source "WorkflowSemantic.lean"
  contract := {
    id := id "workflow.capability.lifecycle"
    semanticDigest := "workflow-lifecycle/v1"
    requiredLaws := [providerLaw]
  }
  meanings := [{
    declaration := id "workflow-nexus.relation.owns-operation"
    kind := .relation
    semanticDigest := "workflow-ownership/v1"
  }]
  lawWitnesses := [witness providerLaw]
}

def nexusProvider : CapabilityProvider := {
  id := id "nexus.provider.cancellation"
  source := source "NexusSemantic.lean"
  contract := {
    id := id "nexus.capability.cancellation"
    semanticDigest := "nexus-cancellation/v1"
    requiredLaws := [providerLaw]
  }
  meanings := [{
    declaration := id "workflow-nexus.relation.owns-operation"
    kind := .relation
    semanticDigest := "nexus-ownership/v1"
  }]
  lawWitnesses := [witness providerLaw]
}

def ownershipConnector : CapabilityConnector := {
  id := id "workflow-nexus.connector.ownership"
  source := source "WorkflowNexusSemantic.lean"
  semanticDigest := "workflow-nexus-ownership/v1"
  reconciliations := [{
    declaration := id "workflow-nexus.relation.owns-operation"
    kind := .relation
    providers := [workflowProvider.id, nexusProvider.id]
    semanticDigest := "workflow-nexus-ownership/reconciled-v1"
  }]
  requiredLaws := [connectorLaw]
  lawWitnesses := [witness connectorLaw]
}

def workflowDeclarations : List DeclarationMetadata := [
  metadata "workflow-nexus.target.caller-closure" .target,
  metadata "workflow-nexus.kernel.transition" .kernel,
  metadata "workflow.capability.lifecycle" .capability,
  metadata "nexus.capability.cancellation" .capability,
  metadata "workflow.provider.lifecycle" .provider,
  metadata "nexus.provider.cancellation" .provider,
  metadata "umpire.law.provider-sound" .law,
  metadata "umpire.law.connector-sound" .law,
  metadata "workflow-nexus.connector.ownership" .connector,
  metadata "workflow-nexus.relation.owns-operation" .relation,
  metadata "nexus.action.request-cancel" .action,
  metadata "nexus.observation.cancel-delivered" .observation
]

def workflowTarget : TargetDeclaration Unit Bool Bool Bool Bool := {
  id := id "workflow-nexus.target.caller-closure"
  source := source "WorkflowNexusSemantic.lean"
  declarations := workflowDeclarations
  requiredCapabilities := [
    id "workflow.capability.lifecycle",
    id "nexus.capability.cancellation"
  ]
  providers := [workflowProvider, nexusProvider]
  connectors := [ownershipConnector]
  resolvedSetups := [()]
  kernel := .checked workflowKernel
}

example : (composeTarget workflowTarget).isOk = true := by
  native_decide

def switchKernel : TransitionKernel Unit Bool Bool Bool Bool := {
  workflowKernel with
  metadata := {
    id := id "switch.kernel.transition"
    contractDigest := "switch-kernel/v1"
    source := source "SwitchSemantic.lean"
  }
}

def switchProvider : CapabilityProvider := {
  id := id "switch.provider.toggle"
  source := source "SwitchSemantic.lean"
  contract := {
    id := id "switch.capability.toggle"
    semanticDigest := "switch-toggle/v1"
    requiredLaws := [providerLaw]
  }
  meanings := [{
    declaration := id "switch.action.toggle"
    kind := .action
    semanticDigest := "switch-action/v1"
  }]
  lawWitnesses := [witness providerLaw]
}

def switchTarget : TargetDeclaration Unit Bool Bool Bool Bool := {
  id := id "switch.target.two-state"
  source := source "SwitchSemantic.lean"
  declarations := [
    metadata "switch.target.two-state" .target,
    metadata "switch.kernel.transition" .kernel,
    metadata "switch.capability.toggle" .capability,
    metadata "switch.provider.toggle" .provider,
    metadata "switch.action.toggle" .action,
    metadata "umpire.law.provider-sound" .law
  ]
  requiredCapabilities := [id "switch.capability.toggle"]
  providers := [switchProvider]
  connectors := []
  resolvedSetups := [()]
  kernel := .checked switchKernel
}

-- A second model with unrelated vocabulary composes through the exact same public interface.
example : (composeTarget switchTarget).isOk = true := by
  native_decide

def errorOf {Target : Type}
    (result : Except DeclarationError Target) : Option DeclarationError :=
  match result with
  | .error error => some error
  | .ok _ => none

def emptyIdentityTarget : TargetDeclaration Unit Bool Bool Bool Bool := {
  workflowTarget with
  declarations := metadata "" .action :: workflowDeclarations
}

example : (errorOf (composeTarget emptyIdentityTarget)) = some {
    kind := .emptyIdentity
    declarationId := id "umpire.declaration.anonymous"
    sourcePath := "Temporal/Experiment/SemanticsTests.lean"
    offendingValue := "<empty>"
    relatedIdentities := [id ""]
  } := by
  native_decide

def duplicateIdentityTarget : TargetDeclaration Unit Bool Bool Bool Bool := {
  workflowTarget with
  declarations := metadata "workflow-nexus.target.caller-closure" .target :: workflowDeclarations
}

example : (errorOf (composeTarget duplicateIdentityTarget)).map DeclarationError.kind =
    some .duplicateIdentity := by
  native_decide

def unknownIdentityTarget : TargetDeclaration Unit Bool Bool Bool Bool := {
  workflowTarget with
  requiredCapabilities := [id "missing.capability.value"]
}

example : (errorOf (composeTarget unknownIdentityTarget)) = some {
    kind := .unknownIdentity
    declarationId := workflowTarget.id
    sourcePath := "WorkflowNexusSemantic.lean"
    offendingValue := "missing.capability.value"
    relatedIdentities := [id "missing.capability.value"]
  } := by
  native_decide

def wrongKindTarget : TargetDeclaration Unit Bool Bool Bool Bool := {
  workflowTarget with
  requiredCapabilities := [id "nexus.action.request-cancel"]
}

example : (errorOf (composeTarget wrongKindTarget)).map DeclarationError.kind = some .wrongKind := by
  native_decide

def missingLawProvider : CapabilityProvider := {
  workflowProvider with lawWitnesses := []
}

def missingLawTarget : TargetDeclaration Unit Bool Bool Bool Bool := {
  workflowTarget with providers := [missingLawProvider, nexusProvider]
}

example : (errorOf (composeTarget missingLawTarget)).map DeclarationError.kind = some .missingLaw := by
  native_decide

def missingProviderTarget : TargetDeclaration Unit Bool Bool Bool Bool := {
  workflowTarget with providers := [workflowProvider]
}

example : (errorOf (composeTarget missingProviderTarget)).map DeclarationError.kind =
    some .missingProvider := by
  native_decide

def conflictingTarget : TargetDeclaration Unit Bool Bool Bool Bool := {
  workflowTarget with connectors := []
}

example : (errorOf (composeTarget conflictingTarget)).map DeclarationError.kind =
    some .conflictingProviders := by
  native_decide

def secondOwnershipConnector : CapabilityConnector := {
  ownershipConnector with
  id := id "workflow-nexus.connector.alternate-ownership"
  source := source "AlternateWorkflowNexusSemantic.lean"
}

def ambiguousConnectorTarget : TargetDeclaration Unit Bool Bool Bool Bool := {
  workflowTarget with
  declarations := metadata "workflow-nexus.connector.alternate-ownership" .connector :: workflowDeclarations
  connectors := [secondOwnershipConnector, ownershipConnector]
}

example : (errorOf (composeTarget ambiguousConnectorTarget)).map DeclarationError.kind =
    some .ambiguousConnector := by
  native_decide

def compatibleNexusProvider : CapabilityProvider := {
  nexusProvider with
  meanings := [{
    declaration := id "workflow-nexus.relation.owns-operation"
    kind := .relation
    semanticDigest := "workflow-ownership/v1"
  }]
}

def compatibleTarget : TargetDeclaration Unit Bool Bool Bool Bool := {
  workflowTarget with
  providers := [workflowProvider, compatibleNexusProvider]
  connectors := []
}

example : (composeTarget compatibleTarget).isOk = true := by
  native_decide

def incompleteKernelTarget : TargetDeclaration Unit Bool Bool Bool Bool := {
  workflowTarget with
  kernel := .incomplete workflowKernel.metadata [
    id "umpire.kernel-proof.initial-complete",
    id "umpire.kernel-proof.step-sound"
  ]
}

example : (errorOf (composeTarget incompleteKernelTarget)) = some {
    kind := .incompleteKernel
    declarationId := workflowTarget.id
    sourcePath := "Temporal/Experiment/SemanticsTests.lean"
    offendingValue := workflowKernel.metadata.id.value
    relatedIdentities := [
      id "umpire.kernel-proof.initial-complete",
      id "umpire.kernel-proof.step-sound"
    ]
  } := by
  native_decide

-- An emitted step outside the authoritative relation cannot inhabit a checked kernel proof.
def outsideRelation : TransitionResult Bool Bool Bool := {
  modelOutcome := false
  resultingState := false
  observations := [true]
}

example : ¬workflowKernel.authoritativeStep false true outsideRelation := by
  simp [workflowKernel, outsideRelation, transition]

example (result : TransitionResult Bool Bool Bool)
    (member : result ∈ workflowKernel.steps false true) :
    workflowKernel.authoritativeStep false true result :=
  workflowKernel.stepSound false true result member

def reorderedWorkflowTarget : TargetDeclaration Unit Bool Bool Bool Bool := {
  workflowTarget with
  declarations := workflowTarget.declarations.reverse
  requiredCapabilities := workflowTarget.requiredCapabilities.reverse
  providers := workflowTarget.providers.reverse
  connectors := workflowTarget.connectors.reverse
}

example : (composeTarget reorderedWorkflowTarget).toOption.map CheckedTarget.canonicalMetadata =
    (composeTarget workflowTarget).toOption.map CheckedTarget.canonicalMetadata := by
  native_decide

example : (composeTarget reorderedWorkflowTarget).toOption.map CheckedTarget.semanticDigest =
    (composeTarget workflowTarget).toOption.map CheckedTarget.semanticDigest := by
  native_decide

def reorderedConflictTarget : TargetDeclaration Unit Bool Bool Bool Bool := {
  conflictingTarget with
  declarations := conflictingTarget.declarations.reverse
  providers := conflictingTarget.providers.reverse
}

example : (errorOf (composeTarget reorderedConflictTarget)).map canonicalDeclarationErrorJson =
    (errorOf (composeTarget conflictingTarget)).map canonicalDeclarationErrorJson := by
  native_decide

def changedIdentityTarget : TargetDeclaration Unit Bool Bool Bool Bool := {
  workflowTarget with
  id := id "workflow-nexus.target.caller-closure-v2"
  declarations := metadata "workflow-nexus.target.caller-closure-v2" .target ::
    workflowDeclarations.filter (fun declaration => declaration.id != workflowTarget.id)
}

example : (composeTarget changedIdentityTarget).toOption.map CheckedTarget.semanticDigest ≠
    (composeTarget workflowTarget).toOption.map CheckedTarget.semanticDigest := by
  native_decide

def changedContractProvider : CapabilityProvider := {
  workflowProvider with
  contract := { workflowProvider.contract with semanticDigest := "workflow-lifecycle/v2" }
}

def changedContractTarget : TargetDeclaration Unit Bool Bool Bool Bool := {
  workflowTarget with providers := [changedContractProvider, nexusProvider]
}

example : (composeTarget changedContractTarget).toOption.map CheckedTarget.semanticDigest ≠
    (composeTarget workflowTarget).toOption.map CheckedTarget.semanticDigest := by
  native_decide

def changedConnector : CapabilityConnector := {
  ownershipConnector with semanticDigest := "workflow-nexus-ownership/v2"
}

def changedConnectorTarget : TargetDeclaration Unit Bool Bool Bool Bool := {
  workflowTarget with connectors := [changedConnector]
}

example : (composeTarget changedConnectorTarget).toOption.map CheckedTarget.semanticDigest ≠
    (composeTarget workflowTarget).toOption.map CheckedTarget.semanticDigest := by
  native_decide

def changedKernel : TransitionKernel Unit Bool Bool Bool Bool := {
  workflowKernel with
  metadata := { workflowKernel.metadata with contractDigest := "workflow-nexus-kernel/v2" }
}

def changedKernelTarget : TargetDeclaration Unit Bool Bool Bool Bool := {
  workflowTarget with kernel := .checked changedKernel
}

example : (composeTarget changedKernelTarget).toOption.map CheckedTarget.semanticDigest ≠
    (composeTarget workflowTarget).toOption.map CheckedTarget.semanticDigest := by
  native_decide

def changedLaw : LawRequirement := {
  providerLaw with semanticDigest := "provider-sound/v2"
}

def changedLawProvider : CapabilityProvider := {
  workflowProvider with
  contract := { workflowProvider.contract with requiredLaws := [changedLaw] }
  lawWitnesses := [witness changedLaw]
}

def changedLawTarget : TargetDeclaration Unit Bool Bool Bool Bool := {
  workflowTarget with providers := [changedLawProvider, nexusProvider]
}

example : (composeTarget changedLawTarget).toOption.map CheckedTarget.semanticDigest ≠
    (composeTarget workflowTarget).toOption.map CheckedTarget.semanticDigest := by
  native_decide

def documentedTarget : TargetDeclaration Unit Bool Bool Bool Bool := {
  workflowTarget with
  declarations := workflowDeclarations.map fun declaration =>
    if declaration.id == workflowTarget.id then
      { declaration with documentation := "Non-semantic explanatory text." }
    else
      declaration
}

example : (composeTarget documentedTarget).toOption.map CheckedTarget.semanticDigest =
    (composeTarget workflowTarget).toOption.map CheckedTarget.semanticDigest := by
  native_decide

example : (composeTarget documentedTarget).toOption.map CheckedTarget.canonicalMetadata ≠
    (composeTarget workflowTarget).toOption.map CheckedTarget.canonicalMetadata := by
  native_decide

def exactTrace : SemanticTrace Bool Bool Bool SemanticValue := {
  initialState := false
  steps := [{
    selectedAction := true
    modelOutcome := true
    resultingState := true
    observations := [{
      identity := id "switch.observation.enabled"
      value := "enabled"
    }]
  }]
}

example : exactTrace.initialState = false ∧
    exactTrace.steps.map SemanticTraceStep.selectedAction = [true] ∧
    exactTrace.steps.map SemanticTraceStep.modelOutcome = [true] ∧
    exactTrace.steps.map SemanticTraceStep.resultingState = [true] ∧
    exactTrace.steps.flatMap SemanticTraceStep.observations = [{
      identity := id "switch.observation.enabled"
      value := "enabled"
    }] := by
  native_decide

example : canonicalCapabilityProviderJson workflowProvider =
    canonicalCapabilityProviderJson workflowProvider := by
  rfl

example : canonicalCapabilityConnectorJson ownershipConnector =
    canonicalCapabilityConnectorJson ownershipConnector := by
  rfl

end Temporal.Experiment.SemanticsTests
