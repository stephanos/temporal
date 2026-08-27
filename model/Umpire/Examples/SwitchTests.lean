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
    flipLaw.semanticDigest = "switch-flip-preserves-domain-law/v1" ∧
    transitionKernel.metadata.contractDigest = "switch-two-state-kernel/v1" := by
  native_decide

example : (composeTarget targetDeclaration).isOk = true := by
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
    (evidence.roleDomainDigest, evidence.actionDomainDigest)) =
    some ("switch-role-domain/v1", "switch-action-domain/v1") := by
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

example : compiledArtifact.formatVersion = "umpire-experiment/v1" ∧
    compiledArtifact.plan.formatVersion = "umpire-drive-plan/v1" ∧
    compiledArtifact.plan.queryIdentity = exactActionQueryId ∧
    compiledArtifact.plan.querySemanticDigest = exactActionQuery.semanticDigest ∧
    compiledArtifact.plan.behaviorIdentity = exactActionBehaviorId ∧
    compiledArtifact.plan.behaviorSemanticDigest = exactActionBehavior.semanticDigest ∧
    compiledArtifact.plan.targetIdentity = targetId ∧
    compiledArtifact.plan.targetSemanticDigest = target.semanticDigest ∧
    compiledArtifact.plan.kernelIdentity = kernelId ∧
    compiledArtifact.plan.kernelSemanticDigest = "switch-two-state-kernel/v1" ∧
    compiledArtifact.plan.requestedActions = [flipAction] ∧
    compiledArtifact.plan.modelOutcomes = [appliedOutcome] ∧
    compiledArtifact.plan.resultingStates = [onState] ∧
    compiledArtifact.properties.map PortableProperty.identity = [flipPropertyId] ∧
    compiledArtifact.properties.map PortableProperty.semanticDigest = [flipProperty.semanticDigest] ∧
    compiledArtifact.provenance.sources = [source] ∧
    compiledArtifact.plan.provenance = compiledArtifact.provenance := by
  native_decide

example : canonicalExperimentSpecJson compiledArtifact ++ "\n" = expectedCompiledArtifactJson := by
  native_decide

end Umpire.Examples.SwitchTests
