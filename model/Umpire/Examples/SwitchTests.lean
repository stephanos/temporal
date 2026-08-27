import Umpire.Examples.Switch

namespace Umpire.Examples.SwitchTests

open Umpire
open Umpire.Examples.Switch

private def expectedExactActionQueryJson : String :=
  include_str "Fixtures/SwitchExactActionQuery.json"

private def expectedCompiledArtifactJson : String :=
  include_str "Fixtures/SwitchCompiledArtifact.json"

example : source = {
    path := "Umpire/Examples/Switch.lean"
    line := 1
    column := 1
    provenance := "lean-model"
  } ∧
    targetId.value = "switch.target.two-state" ∧
    kernelId.value = "switch.kernel.two-state" ∧
    flipPropertyId.value = "switch.property.flip-turns-on" ∧
    exactActionBehaviorId.value = "switch.behavior.exact-action" ∧
    exactActionQueryId.value = "switch.query.exact-action" ∧
    flipLaw.body = "switch-flip-preserves-domain-law/v1" ∧
    transitionKernel.metadata.id = kernelId := by
  native_decide

example : (checkTarget targetAuthoring).isOk = true := by
  native_decide

example : target.kernel.initialStates switchSetup = [offState] ∧
    target.kernel.steps offState flipAction = [appliedResult, deferredResult] := by
  native_decide

example : target.requiredCapabilities = [switchCapabilityId] ∧
    flipProperty.requires = [switchCapabilityId] ∧
    exploratoryBehavior.requires = [switchCapabilityId] ∧
    exactActionQuery.targetComposition = [switchCapabilityId, switchProviderId] := by
  native_decide

example : exactActionQuery.completeness.map (fun evidence =>
    (evidence.roleDomainFingerprint, evidence.actionDomainFingerprint)) =
    (CheckedQueryTarget.ofTarget target).completeness.map (fun evidence =>
      (evidence.roleDomainFingerprint, evidence.actionDomainFingerprint)) := by
  native_decide

example : (match target.planning with
    | .unavailable => none
    | .available capability => some capability.actions) =
    exactActionQuery.completeness.map (fun evidence => evidence.actions) := by
  native_decide

example : canonicalQueryJson exactActionQuery ++ "\n" = expectedExactActionQueryJson := by
  native_decide

example : exactActionBehavior.admits appliedTrace &&
    exactActionBehavior.admits deferredTrace := by
  native_decide

example : exactTraceBehavior.admits appliedTrace &&
    !exactTraceBehavior.admits deferredTrace := by
  native_decide

example : [
    exploratoryRun.result.outcome.name,
    exactActionRun.result.outcome.name,
    exactTraceRun.result.outcome.name
  ] = ["found", "found", "found"] := by
  native_decide

example : compiledArtifact.formatVersion = "umpire-experiment/v2" ∧
    compiledArtifact.plan.formatVersion = "umpire-drive-plan/v2" ∧
    compiledArtifact.plan.queryDefinitionId = exactActionQueryId ∧
    compiledArtifact.plan.queryBehaviorFingerprint = exactActionQuery.behaviorFingerprint ∧
    compiledArtifact.plan.behaviorDefinitionId = exactActionBehaviorId ∧
    compiledArtifact.plan.behaviorFingerprint = exactActionBehavior.behaviorFingerprint ∧
    compiledArtifact.plan.targetDefinitionId = targetId ∧
    compiledArtifact.plan.targetBehaviorFingerprint = target.behaviorFingerprint ∧
    compiledArtifact.plan.kernelDefinitionId = kernelId ∧
    compiledArtifact.plan.kernelBehaviorFingerprint = target.behaviorFingerprint ∧
    compiledArtifact.plan.requestedActions = [flipAction] ∧
    compiledArtifact.plan.modelOutcomes = [appliedOutcome] ∧
    compiledArtifact.plan.resultingStates = [onState] ∧
    compiledArtifact.properties.map PortableProperty.definitionId = [flipPropertyId] ∧
    compiledArtifact.properties.map PortableProperty.behaviorFingerprint = [flipProperty.behaviorFingerprint] ∧
    compiledArtifact.provenance.sourceLocations = [source] ∧
    compiledArtifact.plan.provenance = compiledArtifact.provenance := by
  native_decide

example : canonicalExperimentSpecBytes compiledArtifact = expectedCompiledArtifactJson := by
  native_decide

end Umpire.Examples.SwitchTests
