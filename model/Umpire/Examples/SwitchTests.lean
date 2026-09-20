import Umpire.Examples.Switch
import Umpire.Variations.Tests.Compilation
import Umpire.Variations.Tests.Determinism
import Umpire.Variations.Tests.Intent
import Umpire.Variations.Tests.Metadata
import Umpire.Variations.Tests.Validation
import Umpire.Json

/-!
# What the switch example says

The pins over `Umpire.Examples.Switch`: the Definition IDs the commands derived under the `umpire`
root, the Model the `machine` command enumerated, the specimens the hand-written example carried --
a closed valid declaration needs no proof of its own, an invalid one is rejected where it is
written, and each rejection is the error its stage returns -- and the exact Query and artifact
bytes that are its goldens.
-/

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

private def behaviorErrorOf : Except ScenarioError CheckedScenario → Option ScenarioError
  | .ok _ => none
  | .error error => some error

/-! ### The Model the commands declared -/

#guard twoState.actionKeys == #["flip"]
#guard twoState.table.states.length == 2
#guard twoState.stuck == none

/-- info: 'Umpire.Examples.Switch.twoState' depends on axioms: [propext] -/
#guard_msgs in
#print axioms twoState

/-! The Definition IDs hang off the `umpire` root through the `switch` family, each member under
the machine that owns it. -/
example : source = {
    path := "Umpire/Examples/Switch.lean"
    line := 1
    column := 1
    provenance := "lean-model"
  } ∧
    targetId.value = "umpire.switch.target.twoState" ∧
    kernelId.value = "umpire.switch.kernel.twoState.planner" ∧
    switchCapabilityId.value = "umpire.switch.capability.twoState.transitions" ∧
    switchProviderId.value = "umpire.switch.provider.twoState.finite-table" ∧
    flipLawId.value = "umpire.switch.law.twoState.canonical-table" ∧
    switchRoleId.value = "umpire.switch.role.twoState.subject" ∧
    powerStateId.value = "umpire.switch.state-field.twoState.power" ∧
    flipActionId.value = "umpire.switch.action.twoState.flip" ∧
    appliedOutcomeId.value = "umpire.switch.outcome.twoState.applied" ∧
    deferredOutcomeId.value = "umpire.switch.outcome.twoState.deferred" ∧
    powerObservationId.value = "umpire.switch.fact.twoState.off" ∧
    flipPropertyId.value = "umpire.switch.property.flipTurnsOn" ∧
    flipOccurrenceId.value = "umpire.switch.occurrence.oneFlip.1" ∧
    relationIds = [offFlipRelationId, onFlipRelationId] ∧
    offFlipRelationId.value = "umpire.switch.relation.twoState.off-flip" ∧
    onFlipRelationId.value = "umpire.switch.relation.twoState.on-flip" ∧
    exploratoryBehaviorId.value = "umpire.switch.behavior.explore" ∧
    exactActionBehaviorId.value = "umpire.switch.behavior.oneFlip" ∧
    exactTraceBehaviorId.value = "umpire.switch.behavior.exactTrace" ∧
    exploratoryQueryId.value = "umpire.switch.query.explore" ∧
    exactActionQueryId.value = "umpire.switch.query.exactAction" ∧
    exactTraceQueryId.value = "umpire.switch.query.exactTrace" ∧
    flipLaw.body = flipLawId.value ∧
    machine.metadata.id = kernelId := by
  native_decide

/-! Each state, each fact and each outcome is its own definition; the `power` field is one more,
under which a Property reads the position apart from the state. -/
example : [offState, onState, flipAction, appliedOutcome, deferredOutcome, powerOffObservation,
    powerOnObservation].map (fun value => (value.definitionId.value, value.value)) = [
    ("umpire.switch.state.twoState.off", "off"),
    ("umpire.switch.state.twoState.on", "on"),
    ("umpire.switch.action.twoState.flip", "flip"),
    ("umpire.switch.outcome.twoState.applied", "applied"),
    ("umpire.switch.outcome.twoState.deferred", "deferred"),
    ("umpire.switch.fact.twoState.off", "off"),
    ("umpire.switch.fact.twoState.on", "on")
  ] ∧
    target.stateFields offState = [ModelValue.named powerStateId "off"] ∧
    target.stateFields onState = [ModelValue.named powerStateId "on"] := by
  native_decide

/-! The declared definitions: the target, its kernel, capability, provider and law, then every
member the provider means, then one relation per enumerated row. Each carries its own id as its
behavior version. -/
example : definitions.map (fun definition => (definition.id.value, definition.kind)) = [
    ("umpire.switch.target.twoState", .target),
    ("umpire.switch.kernel.twoState.planner", .machine),
    ("umpire.switch.capability.twoState.transitions", .capability),
    ("umpire.switch.provider.twoState.finite-table", .provider),
    ("umpire.switch.law.twoState.canonical-table", .law),
    ("umpire.switch.state.twoState.off", .state),
    ("umpire.switch.state.twoState.on", .state),
    ("umpire.switch.state-field.twoState.power", .state),
    ("umpire.switch.action.twoState.flip", .action),
    ("umpire.switch.outcome.twoState.applied", .outcome),
    ("umpire.switch.outcome.twoState.deferred", .outcome),
    ("umpire.switch.fact.twoState.off", .fact),
    ("umpire.switch.fact.twoState.on", .fact),
    ("umpire.switch.relation.twoState.off-flip", .relation),
    ("umpire.switch.relation.twoState.on-flip", .relation)
  ] ∧
    definitions.all (fun definition =>
      definition.source == source && definition.version == 1 &&
        definition.behaviorVersion == definition.id.value && definition.documentation == "") ∧
    modelSpec.definitions = definitions ∧
    switchProvider.id = switchProviderId ∧
    switchProvider.contract.id = switchCapabilityId ∧
    switchProvider.contract.requiredLaws = [flipLaw] ∧
    switchProvider.meanings.map (·.definitionId) =
      (definitions.filter fun definition =>
        definition.kind == .state || definition.kind == .action ||
          definition.kind == .outcome || definition.kind == .fact).map (·.id) := by
  native_decide

/-! The records `Umpire.Search` reads compose back into the checked target: the same kernel, the
same provider, the same finite planning. -/
example : (checkModel targetAuthoring).isOk = true := by
  native_decide

example :
    PropertyPattern.exact .selectedAction flipActionId flipAction.value = {
      field := .selectedAction
      reference := flipActionId
      constraint := .equals flipAction.value
    } ∧
    SetupConstraint.roleEquals
      (DefinitionId.of "umpire.switch.setup.oneFlip.subject") switchRoleId offState = {
        id := DefinitionId.of "umpire.switch.setup.oneFlip.subject"
        relation := .equal
        left := .role switchRoleId
        right := .value offState
      } ∧
    Scenario.Trace.singleStep switchSetup offState flipAction appliedResult = {
      setup := switchSetup
      trace := {
        initialState := offState
        steps := [ModelTraceStep.result flipAction appliedResult]
      }
    } := by
  exact ⟨rfl, rfl, rfl⟩

/-! The `scenario` command's Scenario: the subject starts off, one flip is selected and nothing
else, under the machine's own capability. -/
example : exactActionBehaviorDeclaration.id = exactActionBehaviorId ∧
    exactActionBehaviorDeclaration.requires = [switchCapabilityId] ∧
    exactActionBehaviorDeclaration.roles = [switchRole] ∧
    exactActionBehaviorDeclaration.setup = [setupConstraint] ∧
    setupConstraint.id.value = "umpire.switch.setup.oneFlip.subject" ∧
    exactActionBehaviorDeclaration.requiredOccurrences =
      [{ id := flipOccurrenceId, action := flipActionId }] ∧
    exactActionBehaviorDeclaration.actionsExactly = some [flipActionId] ∧
    exploratoryBehaviorDeclaration = {
      exactActionBehaviorDeclaration with
      id := exploratoryBehaviorId
      setup := [{ setupConstraint with id := DefinitionId.of "umpire.switch.setup.explore.subject" }]
      requiredOccurrences := [{
        id := DefinitionId.of "umpire.switch.occurrence.explore.1", action := flipActionId }]
    } := by
  native_decide

/-! The `property` command's Property: one clause, a selected flip whose resulting state is on. -/
example : authoredProperty.id = flipPropertyId ∧
    authoredProperty.requires = [switchCapabilityId] ∧
    authoredProperty.clauses = [
      .transitionContract (DefinitionId.of "umpire.switch.property.flipTurnsOn.state-on")
        (PropertyPattern.exact .selectedAction flipActionId flipAction.value)
        (PropertyPattern.exact .resultingState onState.definitionId onState.value)
    ] := by
  native_decide

/-! A closed valid declaration needs no proof of its own: the default `native_decide` supplies it. -/
example :
    Property.checked (PropertyCheckContext.ofTarget target) (authoredProperty) =
      propertyResult.toOption.get (by native_decide) ∧
    Scenario.checked (.ofTarget target) exactActionBehaviorDeclaration =
      exactActionBehaviorResult.toOption.get (by native_decide) ∧
    Property.checked (PropertyCheckContext.ofTarget target) (authoredProperty) = flipProperty ∧
    Scenario.checked (.ofTarget target) exactActionBehaviorDeclaration = exactActionBehavior := by
  native_decide

/-! An invalid declaration is rejected where it is written, by the same default proof. -/
/--
error: could not synthesize default value for parameter 'valid' using tactics
-/
#guard_msgs (error, substring := true) in
def propertyWithoutValidityProof : CheckedProperty :=
  Property.checked (PropertyCheckContext.ofTarget target)
    { authoredProperty with id := DefinitionId.of "" }

/--
error: could not synthesize default value for parameter 'valid' using tactics
-/
#guard_msgs (error, substring := true) in
def behaviorWithoutValidityProof : CheckedScenario :=
  Scenario.checked (.ofTarget target) { exactActionBehaviorDeclaration with id := DefinitionId.of "" }

example : [
    propertyErrorOf (Property.check (PropertyCheckContext.ofTarget target) ({
      authoredProperty with
      id := DefinitionId.of ""
      source := { source with path := "" }
    })),
    propertyErrorOf (Property.check (PropertyCheckContext.ofTarget target) ({
      authoredProperty with
      id := DefinitionId.of "property"
      source := { source with path := "" }
    })),
    propertyErrorOf (Property.check (PropertyCheckContext.ofTarget target) ({
      authoredProperty with
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
    propertyErrorJsonOf (Property.check (PropertyCheckContext.ofTarget target) ({
      authoredProperty with
      id := DefinitionId.of ""
      source := { source with path := "" }
    })),
    propertyErrorJsonOf (Property.check (PropertyCheckContext.ofTarget target) ({
      authoredProperty with
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
    behaviorErrorOf (Scenario.check (.ofTarget target) {
      exploratoryBehaviorDeclaration with
      id := DefinitionId.of ""
      source := { source with path := "" }
    }),
    behaviorErrorOf (Scenario.check (.ofTarget target) {
      exploratoryBehaviorDeclaration with
      id := DefinitionId.of "behavior"
      source := { source with path := "" }
    }),
    behaviorErrorOf (Scenario.check (.ofTarget target) {
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

example : target.machine.initialStates switchSetup = [offState] ∧
    target.machine.steps offState flipAction = [appliedResult, deferredResult] ∧
    target.machine.steps onState flipAction = [appliedFromOnResult, deferredFromOnResult] := by
  native_decide

/-! The kernel `Umpire.Search` reads is the checked target's own, with its two results per flip,
and both flips from off are authoritative steps. -/
theorem direct_kernel_keeps_independent_authority_and_two_results :
    machine.authoritativeInitial = authoritativeInitial ∧
    machine.authoritativeStep = authoritativeStep ∧
    modelSpec.machine = .checked machine ∧
    stepResults offState flipAction = [appliedResult, deferredResult] ∧
    authoritativeStep offState flipAction appliedResult ∧
    authoritativeStep offState flipAction deferredResult := by
  exact ⟨rfl, rfl, rfl, by native_decide,
    target_off_flip_applied_authoritative,
    target.machine.stepSound offState flipAction deferredResult (by rw [target_steps.1]; simp)⟩

/-! What the target admits is exactly the switch's vocabulary, as a proof over its domains reads
it. -/
example (value : List RoleBinding) (admitted : target.machine.setupDomain value) :
    value = switchSetup :=
  target_setupDomain value admitted

example (value : ModelValue) (admitted : target.machine.stateDomain value) :
    value = offState ∨ value = onState :=
  target_stateDomain value admitted

theorem direct_kernel_golden_behavior_fingerprint :
    target.behaviorFingerprint.render =
      "sha256:4bd4815c4faf255165b36a5c2e82e8a2148e45d85d646e545f1e66b3b5a2ef4b" := by
  native_decide

example : target.requiredCapabilities = [switchCapabilityId] ∧
    flipProperty.requires = [switchCapabilityId] ∧
    exploratoryBehavior.requires = [switchCapabilityId] ∧
    exactActionQuery.modelProviders = [switchCapabilityId, switchProviderId] := by
  native_decide

example : exactActionQuery.completeness.map (fun evidence =>
    (evidence.roleDomainFingerprint, evidence.actionDomainFingerprint)) =
    (ModelCompleteness.ofTarget target).completeness.map (fun evidence =>
      (evidence.roleDomainFingerprint, evidence.actionDomainFingerprint)) := by
  native_decide

example : (match target.planning with
    | .unavailable => none
    | .available capability => some capability.actions) =
    exactActionQuery.completeness.map (fun evidence => evidence.actions) := by
  native_decide

/-! The exact-action Query is the `query` command's own, and admitting it again through
`Search.admit` searches to the same run. -/
example : (exactAction.toOption.map fun admitted => canonicalQueryJson admitted.query) =
      some (canonicalQueryJson exactActionQuery) ∧
    exactActionAdmitted.query.canonicalMetadata = exactActionQuery.canonicalMetadata ∧
    exactActionRunResult.toOption.map PlanResult.result = some exactActionRun.result ∧
    exactActionQueryResult.toOption.map canonicalQueryJson =
      some (canonicalQueryJson exactActionQuery) := by
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
    exploratoryRun.toOption.map (fun run => run.result.outcome.name),
    some exactActionRun.result.outcome.name,
    exactTraceRun.toOption.map (fun run => run.result.outcome.name)
  ] = [some "found", some "found", some "found"] := by
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
    compiledArtifact.plan.linearExtension.map (·.definitionId) = [flipOccurrenceId] ∧
    compiledArtifact.properties.map PortableProperty.definitionId = [flipPropertyId] ∧
    compiledArtifact.properties.map PortableProperty.behaviorFingerprint = [flipProperty.behaviorFingerprint] ∧
    compiledArtifact.observationRequirementDefinitionIds =
      [powerOffObservation.definitionId, powerOnObservation.definitionId] ∧
    compiledArtifact.provenance.sourceLocations = [source] ∧
    compiledArtifact.plan.provenance = compiledArtifact.provenance := by
  native_decide

example : canonicalPlanBytes compiledArtifact = expectedCompiledArtifactJson := by
  native_decide

end Umpire.Examples.SwitchTests
