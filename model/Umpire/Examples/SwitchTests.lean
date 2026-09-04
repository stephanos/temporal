import Umpire.Examples.Switch
import Umpire.Space.Tests.Compilation
import Umpire.Space.Tests.Determinism
import Umpire.Space.Tests.Intent
import Umpire.Space.Tests.Metadata
import Umpire.Space.Tests.Validation
import Umpire.Json

namespace Umpire.Examples.SwitchTests

open Umpire
open Umpire.Examples.Switch

private def expectedExactActionQueryJson : String :=
  include_str "Fixtures/SwitchExactActionQuery.json"

private def expectedCompiledArtifactJson : String :=
  include_str "Fixtures/SwitchCompiledArtifact.json"

private def propertyErrorOf : Except PropertyError CheckedProperty → Option PropertyError
  | .ok _ => none
  | .error error => some error

private def propertyErrorJsonOf : Except PropertyError CheckedProperty → Option String
  | .ok _ => none
  | .error error => some (canonicalPropertyErrorJson error)

private def behaviorErrorOf : Except BehaviorError CheckedBehavior → Option BehaviorError
  | .ok _ => none
  | .error error => some error

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

example : definitions = [
    { id := targetId, kind := .target, source, version := 1,
      canonicalBehavior := "switch-two-state-target/v1", documentation := "" },
    { id := kernelId, kind := .kernel, source, version := 1,
      canonicalBehavior := "switch-two-state-kernel/v1", documentation := "" },
    { id := switchCapabilityId, kind := .capability, source, version := 1,
      canonicalBehavior := "switch-state/v1", documentation := "" },
    { id := switchProviderId, kind := .provider, source, version := 1,
      canonicalBehavior := "switch-state-provider/v1", documentation := "" },
    { id := flipLawId, kind := .law, source, version := 1,
      canonicalBehavior := "switch-flip-preserves-domain-law/v1", documentation := "" },
    { id := powerStateId, kind := .state, source, version := 1,
      canonicalBehavior := "switch-power-state/v1", documentation := "" },
    { id := flipActionId, kind := .action, source, version := 1,
      canonicalBehavior := "switch-flip-action/v1", documentation := "" },
    { id := appliedOutcomeId, kind := .outcome, source, version := 1,
      canonicalBehavior := "switch-applied-outcome/v1", documentation := "" },
    { id := deferredOutcomeId, kind := .outcome, source, version := 1,
      canonicalBehavior := "switch-deferred-outcome/v1", documentation := "" },
    { id := powerObservationId, kind := .observation, source, version := 1,
      canonicalBehavior := "switch-power-observation/v1", documentation := "" }
  ] := by
  native_decide

example : (checkTarget targetAuthoring).isOk = true := by
  native_decide

example :
    PropertyPattern.exact .selectedAction flipActionId flipAction.value = {
      field := .selectedAction
      reference := flipActionId
      constraint := .equals flipAction.value
    } ∧
    SetupConstraint.roleEquals
      (DefinitionId.of "switch.setup.subject-is-off") switchRoleId offState = {
        id := DefinitionId.of "switch.setup.subject-is-off"
        relation := .equal
        left := .role switchRoleId
        right := .value offState
      } ∧
    BehaviorTrace.singleStep switchSetup offState flipAction appliedResult = {
      setup := switchSetup
      trace := {
        initialState := offState
        steps := [ModelTraceStep.result flipAction appliedResult]
      }
    } ∧
    BehaviorDeclaration.exactlyOneAction exactActionBehaviorId source
      { id := DefinitionId.of "switch.occurrence.flip", action := flipActionId }
      (requires := [switchCapabilityId])
      (roles := [switchRole])
      (setup := [setupConstraint])
      (documentation := "Select one flip while leaving its outcome to the switch model.") = {
        exploratoryBehaviorDeclaration with
        id := exactActionBehaviorId
        actionsExactly := some [flipActionId]
        documentation := "Select one flip while leaving its outcome to the switch model."
      } := by
  exact ⟨rfl, rfl, rfl, rfl⟩

example :
    checkedProperty (PropertyCheckContext.ofTarget target) (.portable propertyDeclaration)
      (by native_decide) = propertyResult.toOption.get (by native_decide) ∧
    checkedBehavior (.ofTarget target) exactActionBehaviorDeclaration
      (by native_decide) = exactActionBehaviorResult.toOption.get (by native_decide) := by
  native_decide

#guard_msgs (error, substring := true) in
def propertyWithoutValidityProof : CheckedProperty :=
  checkedProperty (PropertyCheckContext.ofTarget target) (.portable propertyDeclaration)

#guard_msgs (error, substring := true) in
def behaviorWithoutValidityProof : CheckedBehavior :=
  checkedBehavior (.ofTarget target) exactActionBehaviorDeclaration

example : [
    propertyErrorOf (checkProperty (PropertyCheckContext.ofTarget target) (.portable {
      propertyDeclaration with
      id := DefinitionId.of ""
      source := { source with path := "" }
    })),
    propertyErrorOf (checkProperty (PropertyCheckContext.ofTarget target) (.portable {
      propertyDeclaration with
      id := DefinitionId.of "property"
      source := { source with path := "" }
    })),
    propertyErrorOf (checkProperty (PropertyCheckContext.ofTarget target) (.portable {
      propertyDeclaration with
      source := { source with path := "" }
      requires := [
        DefinitionId.of "switch.capability.z",
        DefinitionId.of "switch.capability.a",
        DefinitionId.of "switch.capability.z",
        DefinitionId.of "switch.capability.a"
      ]
    }))
  ] = [
    some {
      kind := .emptyDefinitionId
      definitionId := DefinitionId.of "umpire.property.anonymous"
      sourcePath := "<unknown>"
      offendingValue := "<empty>"
      relatedDefinitionIds := [DefinitionId.of ""]
    },
    some {
      kind := .invalidDefinitionId
      definitionId := DefinitionId.of "property"
      sourcePath := "<unknown>"
      offendingValue := "property"
      relatedDefinitionIds := [DefinitionId.of "property"]
    },
    some {
      kind := .duplicateDefinitionId
      definitionId := flipPropertyId
      sourcePath := "<unknown>"
      offendingValue := "switch.capability.a"
      relatedDefinitionIds := [DefinitionId.of "switch.capability.a"]
    }
  ] := by
  native_decide

example : [
    propertyErrorJsonOf (checkProperty (PropertyCheckContext.ofTarget target) (.portable {
      propertyDeclaration with
      id := DefinitionId.of ""
      source := { source with path := "" }
    })),
    propertyErrorJsonOf (checkProperty (PropertyCheckContext.ofTarget target) (.portable {
      propertyDeclaration with
      id := DefinitionId.of "property"
      source := { source with path := "" }
    }))
  ] = [
    some ("{\"kind\":\"empty-definition-id\",\"definitionId\":" ++
      "\"umpire.property.anonymous\",\"sourcePath\":\"<unknown>\"," ++
      "\"offendingValue\":\"<empty>\",\"relatedDefinitionIds\":[\"\"]}"),
    some ("{\"kind\":\"invalid-definition-id\",\"definitionId\":\"property\"," ++
      "\"sourcePath\":\"<unknown>\",\"offendingValue\":\"property\"," ++
      "\"relatedDefinitionIds\":[\"property\"]}")
  ] := by
  native_decide

example : [
    behaviorErrorOf (checkBehavior (.ofTarget target) {
      exploratoryBehaviorDeclaration with
      id := DefinitionId.of ""
      source := { source with path := "" }
    }),
    behaviorErrorOf (checkBehavior (.ofTarget target) {
      exploratoryBehaviorDeclaration with
      id := DefinitionId.of "behavior"
      source := { source with path := "" }
    }),
    behaviorErrorOf (checkBehavior (.ofTarget target) {
      exploratoryBehaviorDeclaration with
      source := { source with path := "" }
      requires := [
        DefinitionId.of "switch.capability.z",
        DefinitionId.of "switch.capability.a",
        DefinitionId.of "switch.capability.z",
        DefinitionId.of "switch.capability.a"
      ]
    })
  ] = [
    some {
      kind := .emptyDefinitionId
      definitionId := DefinitionId.of "umpire.behavior.anonymous"
      sourcePath := "<unknown>"
      offendingValue := "<empty>"
      relatedDefinitionIds := [DefinitionId.of ""]
    },
    some {
      kind := .invalidDefinitionId
      definitionId := DefinitionId.of "behavior"
      sourcePath := "<unknown>"
      offendingValue := "behavior"
      relatedDefinitionIds := [DefinitionId.of "behavior"]
    },
    some {
      kind := .duplicateDefinitionId
      definitionId := exploratoryBehaviorId
      sourcePath := "<unknown>"
      offendingValue := "switch.capability.a"
      relatedDefinitionIds := [DefinitionId.of "switch.capability.a"]
    }
  ] := by
  native_decide

example : target.kernel.initialStates switchSetup = [offState] ∧
    target.kernel.steps offState flipAction = [appliedResult, deferredResult] := by
  native_decide

theorem direct_kernel_keeps_independent_authority_and_two_results :
    transitionKernel.authoritativeInitial = authoritativeInitial ∧
    transitionKernel.authoritativeStep = authoritativeStep ∧
    targetDefinition.kernel = .checked transitionKernel ∧
    stepResults offState flipAction = [appliedResult, deferredResult] ∧
    authoritativeStep offState flipAction appliedResult ∧
    authoritativeStep offState flipAction deferredResult := by
  exact ⟨rfl, rfl, rfl, by native_decide,
    ⟨rfl, .inl ⟨rfl, .inl rfl⟩⟩,
    ⟨rfl, .inl ⟨rfl, .inr rfl⟩⟩⟩

theorem direct_kernel_golden_behavior_fingerprint :
    target.behaviorFingerprint.render =
      "sha256:0443154d4f2860a69590a3d3867f4992ad17024ebf62b424545382c41b871666" := by
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

example : Json.prettyBytes (canonicalQueryJson exactActionQuery) = expectedExactActionQueryJson := by
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
