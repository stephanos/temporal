import Temporal.Experiment.Compiler
import Temporal.Experiment.Inspect
import Temporal.Experiment.Json

namespace Temporal.ExperimentTests

open Temporal.Experiment

def resource (value : String) : ResourceId := ⟨value⟩
def action (value : String) : ActionId := ⟨value⟩
def property (value : String) : PropertyId := ⟨value⟩
def model (value : String) : ModelId := ⟨value⟩
def regression (value : String) : RegressionId := ⟨value⟩

def workerReady (setup : ResolvedSetup) : Option ModelOutcome :=
  if setup.resources.any (fun item => item.id == resource "worker" && item.value == "ready") then
    some ⟨"advanced"⟩
  else
    none

def observed (_ : ResolvedSetup) : Option ModelOutcome := some ⟨"observed"⟩

def validTarget : ModelTarget := {
  id := model "synthetic-model"
  declaration := "synthetic-declaration-v1"
  resources := [
    { id := resource "worker", value := "ready" },
    { id := resource "queue", value := "available" }
  ]
  actionProjections := [
    { id := action "observe", project := observed },
    { id := action "advance", project := workerReady }
  ]
  propertyObservations := [
    { id := property "visible", contract := "eventually-visible" },
    { id := property "durable", contract := "always-durable" }
  ]
  provenance := { source := "synthetic-model", compiler := "lean-regression" }
}

def validRegression : Regression := {
  id := regression "synthetic-regression"
  target := model "synthetic-model"
  resources := [resource "worker", resource "queue"]
  actionAttempts := [action "observe", action "advance"]
  ordering := [{ before := action "advance", after := action "observe" }]
  expectedProperties := ⟨[property "visible", property "durable"]⟩
  bounds := { resources := 2, actions := 2, precedenceEdges := 1 }
  omissions := ["persistence", "execution"]
}

def expectedIdentity : String :=
  "temporal-model/v1:" ++
  "{\"targetId\":\"synthetic-model\",\"targetDeclaration\":\"synthetic-declaration-v1\"," ++
  "\"resolvedSetup\":[{\"resourceId\":\"queue\",\"value\":\"available\"}," ++
  "{\"resourceId\":\"worker\",\"value\":\"ready\"}]," ++
  "\"projectedOutcomes\":[{\"actionId\":\"advance\",\"outcome\":\"advanced\"}," ++
  "{\"actionId\":\"observe\",\"outcome\":\"observed\"}]," ++
  "\"propertyObservations\":[{\"propertyId\":\"durable\",\"contract\":\"always-durable\"}," ++
  "{\"propertyId\":\"visible\",\"contract\":\"eventually-visible\"}]}"

def expectedSpec : ExperimentSpec := {
  formatVersion := "temporal-experiment/v1"
  regressionId := regression "synthetic-regression"
  targetId := model "synthetic-model"
  modelIdentity := expectedIdentity
  resources := [resource "queue", resource "worker"]
  resolvedSetup := ⟨[
    { id := resource "queue", value := "available" },
    { id := resource "worker", value := "ready" }
  ]⟩
  actionAttempts := [action "advance", action "observe"]
  projectedOutcomes := [
    { actionId := action "advance", outcome := ⟨"advanced"⟩ },
    { actionId := action "observe", outcome := ⟨"observed"⟩ }
  ]
  ordering := [{ before := action "advance", after := action "observe" }]
  expectedProperties := [
    { propertyId := property "durable", observationContract := "always-durable" },
    { propertyId := property "visible", observationContract := "eventually-visible" }
  ]
  bounds := { resources := 2, actions := 2, precedenceEdges := 1 }
  omissions := ["execution", "persistence"]
  provenance := { source := "synthetic-model", compiler := "lean-regression" }
}

example : (compile validTarget validRegression).toOption = some expectedSpec := by
  native_decide

def expectedJson : String :=
  "{\"formatVersion\":\"temporal-experiment/v1\",\"regressionId\":\"synthetic-regression\"," ++
  "\"targetId\":\"synthetic-model\",\"modelIdentity\":\"temporal-model/v1:" ++
  "{\\\"targetId\\\":\\\"synthetic-model\\\",\\\"targetDeclaration\\\":\\\"synthetic-declaration-v1\\\"," ++
  "\\\"resolvedSetup\\\":[{\\\"resourceId\\\":\\\"queue\\\",\\\"value\\\":\\\"available\\\"}," ++
  "{\\\"resourceId\\\":\\\"worker\\\",\\\"value\\\":\\\"ready\\\"}]," ++
  "\\\"projectedOutcomes\\\":[{\\\"actionId\\\":\\\"advance\\\",\\\"outcome\\\":\\\"advanced\\\"}," ++
  "{\\\"actionId\\\":\\\"observe\\\",\\\"outcome\\\":\\\"observed\\\"}]," ++
  "\\\"propertyObservations\\\":[{\\\"propertyId\\\":\\\"durable\\\",\\\"contract\\\":\\\"always-durable\\\"}," ++
  "{\\\"propertyId\\\":\\\"visible\\\",\\\"contract\\\":\\\"eventually-visible\\\"}]}\"," ++
  "\"resources\":[\"queue\",\"worker\"]," ++
  "\"resolvedSetup\":[{\"resourceId\":\"queue\",\"value\":\"available\"}," ++
  "{\"resourceId\":\"worker\",\"value\":\"ready\"}]," ++
  "\"actionAttempts\":[\"advance\",\"observe\"]," ++
  "\"projectedOutcomes\":[{\"actionId\":\"advance\",\"outcome\":\"advanced\"}," ++
  "{\"actionId\":\"observe\",\"outcome\":\"observed\"}]," ++
  "\"ordering\":[{\"before\":\"advance\",\"after\":\"observe\"}]," ++
  "\"expectedProperties\":[{\"propertyId\":\"durable\",\"observationContract\":\"always-durable\"}," ++
  "{\"propertyId\":\"visible\",\"observationContract\":\"eventually-visible\"}]," ++
  "\"bounds\":{\"resources\":2,\"actions\":2,\"precedenceEdges\":1}," ++
  "\"omissions\":[\"execution\",\"persistence\"]," ++
  "\"provenance\":{\"source\":\"synthetic-model\",\"compiler\":\"lean-regression\"}}"

example : canonicalJson expectedSpec = expectedJson := by
  native_decide

def errorOf (result : Except CompileError ExperimentSpec) : Option CompileError :=
  match result with
  | .error error => some error
  | .ok _ => none

def errorKindAndSubject (result : Except CompileError ExperimentSpec) : Option (CompileErrorKind × String) :=
  (errorOf result).map (fun error => (error.kind, error.subject))

def withResources (items : List ResourceId) : Regression := { validRegression with resources := items }
def withActions (items : List ActionId) : Regression := { validRegression with actionAttempts := items }
def withProperties (items : List PropertyId) : Regression :=
  { validRegression with expectedProperties := ⟨items⟩ }

example : [
    errorKindAndSubject (compile validTarget (withResources [resource "", resource "worker"])),
    errorKindAndSubject (compile validTarget (withActions [action "", action "observe"])),
    errorKindAndSubject (compile validTarget (withProperties [property "", property "visible"]))
  ] = [
    some (.missingIdentity, ""),
    some (.missingIdentity, ""),
    some (.missingIdentity, "")
  ] := by
  native_decide

example : [
    errorKindAndSubject (compile validTarget (withResources [resource "worker", resource "worker"])),
    errorKindAndSubject (compile validTarget (withActions [action "observe", action "observe"])),
    errorKindAndSubject (compile validTarget (withProperties [property "visible", property "visible"]))
  ] = [
    some (.duplicateIdentity, "worker"),
    some (.duplicateIdentity, "observe"),
    some (.duplicateIdentity, "visible")
  ] := by
  native_decide

example : errorKindAndSubject (compile validTarget (withProperties [])) =
    some (.emptyExpectations, "synthetic-regression") := by
  native_decide

example : [
    errorKindAndSubject (compile validTarget (withResources [resource "missing-resource"])),
    errorKindAndSubject (compile validTarget (withProperties [property "missing-property"])),
    errorKindAndSubject (compile validTarget {
      validRegression with
      actionAttempts := [action "advance"]
      ordering := [{ before := action "advance", after := action "missing-action" }]
    })
  ] = [
    some (.unresolvedResource, "missing-resource"),
    some (.unresolvedProperty, "missing-property"),
    some (.unresolvedAction, "missing-action")
  ] := by
  native_decide

example : errorKindAndSubject (compile validTarget {
    validRegression with target := model "other-model"
  }) = some (.targetMismatch, "other-model") := by
  native_decide

example : errorKindAndSubject (compile validTarget {
    validRegression with
    actionAttempts := [action "missing-action"]
    ordering := []
  }) = some (.unmappedAction, "missing-action") := by
  native_decide

example : errorOf (compile validTarget {
    validRegression with
    resources := [resource "queue"]
    actionAttempts := [action "advance"]
    ordering := []
    bounds := { resources := 1, actions := 1, precedenceEdges := 0 }
  }) = some {
    kind := .impossibleAction
    subject := "advance"
    context := "[{\"resourceId\":\"queue\",\"value\":\"available\"}]"
  } := by
  native_decide

def duplicateOrdering : Regression := {
  validRegression with
  ordering := [
    { before := action "advance", after := action "observe" },
    { before := action "advance", after := action "observe" }
  ]
  bounds := { resources := 2, actions := 2, precedenceEdges := 2 }
}

def selfOrdering : Regression := {
  validRegression with
  ordering := [{ before := action "advance", after := action "advance" }]
}

def cyclicOrdering : Regression := {
  validRegression with
  ordering := [
    { before := action "observe", after := action "advance" },
    { before := action "advance", after := action "observe" }
  ]
  bounds := { resources := 2, actions := 2, precedenceEdges := 2 }
}

def prefixedCycleTarget : ModelTarget := {
  validTarget with
  actionProjections := [
    { id := action "alpha", project := observed },
    { id := action "beta", project := observed },
    { id := action "gamma", project := observed }
  ]
}

def prefixedCycleOrdering : Regression := {
  validRegression with
  actionAttempts := [action "alpha", action "beta", action "gamma"]
  ordering := [
    { before := action "alpha", after := action "beta" },
    { before := action "beta", after := action "gamma" },
    { before := action "gamma", after := action "beta" }
  ]
  bounds := { resources := 2, actions := 3, precedenceEdges := 3 }
}

example : [
    errorKindAndSubject (compile validTarget duplicateOrdering),
    errorKindAndSubject (compile validTarget selfOrdering),
    errorKindAndSubject (compile validTarget cyclicOrdering)
  ] = [
    some (.duplicateOrdering, "advance->observe"),
    some (.selfOrdering, "advance->advance"),
    some (.cyclicOrdering, "advance")
  ] := by
  native_decide

example : errorKindAndSubject (compile prefixedCycleTarget prefixedCycleOrdering) =
    some (.cyclicOrdering, "beta") := by
  native_decide

example : [
    errorKindAndSubject (compile validTarget {
      validRegression with bounds := { resources := 0, actions := 2, precedenceEdges := 1 }
    }),
    errorKindAndSubject (compile validTarget {
      validRegression with bounds := { resources := 2, actions := 0, precedenceEdges := 1 }
    })
  ] = [
    some (.invalidBound, "resources"),
    some (.invalidBound, "actions")
  ] := by
  native_decide

example : [
    errorKindAndSubject (compile validTarget {
      validRegression with bounds := { resources := 1, actions := 2, precedenceEdges := 1 }
    }),
    errorKindAndSubject (compile validTarget {
      validRegression with bounds := { resources := 2, actions := 1, precedenceEdges := 1 }
    }),
    errorKindAndSubject (compile validTarget {
      validRegression with bounds := { resources := 2, actions := 2, precedenceEdges := 0 }
    })
  ] = [
    some (.boundExceeded, "resources"),
    some (.boundExceeded, "actions"),
    some (.boundExceeded, "precedenceEdges")
  ] := by
  native_decide

example : (compile validTarget {
    validRegression with
    ordering := []
    bounds := { resources := 2, actions := 2, precedenceEdges := 0 }
  }).isOk = true := by
  native_decide

def reorderedTarget : ModelTarget := {
  validTarget with
  resources := validTarget.resources.reverse
  actionProjections := validTarget.actionProjections.reverse
  propertyObservations := validTarget.propertyObservations.reverse
}

def reorderedRegression : Regression := {
  validRegression with
  resources := validRegression.resources.reverse
  actionAttempts := validRegression.actionAttempts.reverse
  expectedProperties := ⟨validRegression.expectedProperties.items.reverse⟩
  omissions := validRegression.omissions.reverse
}

example : (compile reorderedTarget reorderedRegression).toOption.map canonicalJson =
    (compile validTarget validRegression).toOption.map canonicalJson := by
  native_decide

def denseActions : List ActionId :=
  (List.range 22).map (fun index => action ("action-" ++ toString index))

def denseOrdering : List PrecedenceEdge :=
  denseActions.flatMap fun before =>
    denseActions.filterMap fun after =>
      if before.value < after.value then some { before, after } else none

def denseTarget : ModelTarget := {
  validTarget with
  actionProjections := denseActions.map (fun id => { id, project := observed })
}

def denseRegression : Regression := {
  validRegression with
  resources := [resource "queue"]
  actionAttempts := denseActions
  ordering := denseOrdering
  expectedProperties := ⟨[property "durable"]⟩
  bounds := {
    resources := 1
    actions := denseActions.length
    precedenceEdges := denseOrdering.length
  }
}

example : (compile denseTarget denseRegression).isOk = true := by
  native_decide

def changedOutcomeTarget : ModelTarget := {
  validTarget with
  actionProjections := [
    { id := action "observe", project := observed },
    { id := action "advance", project := fun _ => some ⟨"changed-outcome"⟩ }
  ]
}

def changedPropertyTarget : ModelTarget := {
  validTarget with
  propertyObservations := [
    { id := property "visible", contract := "changed-contract" },
    { id := property "durable", contract := "always-durable" }
  ]
}

example : (compile changedOutcomeTarget validRegression).toOption.map ExperimentSpec.modelIdentity ≠
    (compile validTarget validRegression).toOption.map ExperimentSpec.modelIdentity := by
  native_decide

example : (compile changedOutcomeTarget validRegression).toOption.map canonicalJson ≠
    (compile validTarget validRegression).toOption.map canonicalJson := by
  native_decide

example : (compile changedPropertyTarget validRegression).toOption.map ExperimentSpec.modelIdentity ≠
    (compile validTarget validRegression).toOption.map ExperimentSpec.modelIdentity := by
  native_decide

example : (compile changedPropertyTarget validRegression).toOption.map canonicalJson ≠
    (compile validTarget validRegression).toOption.map canonicalJson := by
  native_decide

namespace NexusCallerClosureTests

open NexusCallerClosure

def expectedClashSetupValue : String :=
  "{ op := NexusAutoClose.OpState.started,\n" ++
    "  policy := NexusAutoClose.Policy.requestCancel,\n" ++
    "  cancels := [NexusAutoClose.Initiator.user],\n" ++
    "  callerOpen := true,\n" ++
    "  slack := false }"

def expectedUpgradedOutcomeValue : String :=
  "{ op := NexusAutoClose.OpState.started,\n" ++
    "  policy := NexusAutoClose.Policy.requestCancel,\n" ++
    "  cancels := [NexusAutoClose.Initiator.system],\n" ++
    "  callerOpen := false,\n" ++
    "  slack := false }"

def expectedPilotSetup : ResolvedSetup := ⟨[{
  id := ⟨"caller-closure-clash"⟩
  value := expectedClashSetupValue
}]⟩

def expectedPilotOutcomes : List ProjectedOutcome := [{
  actionId := ⟨"caller-force-close"⟩
  outcome := ⟨expectedUpgradedOutcomeValue⟩
}]

def expectedPilotProperties : List ExpectedProperty := [
  {
    propertyId := ⟨"cancellation-uniqueness"⟩
    observationContract :=
      "NexusAutoClose.upgrade_preserves_uniqueness" ++
        "(NexusAutoClose.wClash,NexusAutoClose.wClash_reachable(upgrade))"
  },
  {
    propertyId := ⟨"honored-delivery"⟩
    observationContract := "NexusAutoClose.upgrade_honors_delivery(NexusAutoClose.wClash)"
  }
]

def expectedPilotSpec : ExperimentSpec := {
  formatVersion := "temporal-experiment/v1"
  regressionId := ⟨"nexus-caller-closure-upgrade"⟩
  targetId := ⟨"nexus-caller-closure"⟩
  modelIdentity := deriveModelIdentity ⟨"nexus-caller-closure"⟩
    "NexusAutoClose.wClash|NexusAutoClose.autoClose:upgrade"
    expectedPilotSetup expectedPilotOutcomes expectedPilotProperties
  resources := [⟨"caller-closure-clash"⟩]
  resolvedSetup := expectedPilotSetup
  actionAttempts := [⟨"caller-force-close"⟩]
  projectedOutcomes := expectedPilotOutcomes
  ordering := []
  expectedProperties := expectedPilotProperties
  bounds := { resources := 1, actions := 1, precedenceEdges := 0 }
  omissions := ["runtime-execution", "state-exploration"]
  provenance := { source := "NexusAutoClose", compiler := "lean-regression" }
}

example : NexusCallerClosure.compiled.toOption = some expectedPilotSpec := by
  native_decide

example : NexusCallerClosure.compiled.toOption.map (fun spec =>
    (spec.actionAttempts, spec.projectedOutcomes)) = some (
      [⟨"caller-force-close"⟩],
      [{ actionId := ⟨"caller-force-close"⟩, outcome := ⟨expectedUpgradedOutcomeValue⟩ }]
    ) := by
  native_decide

def expectedPilotStdout : String := canonicalJson expectedPilotSpec ++ "\n"

example : runCli [regressionId.value] = {
    status := 0
    stdout := expectedPilotStdout
    stderr := ""
  } := by
  native_decide

def repeatedPilotOutput : List String :=
  (List.range 2).map fun _ => (runCli [regressionId.value]).stdout

example : repeatedPilotOutput = List.replicate 2 expectedPilotStdout := by
  native_decide

example : runCli ["missing-pilot"] = {
    status := 1
    stdout := ""
    stderr :=
      "{\"kind\":\"unknownPilot\",\"subject\":\"missing-pilot\"," ++
        "\"context\":\"pilot registry\"}\n"
  } := by
  native_decide

def incompatiblePilot : Pilot := {
  id := "incompatible-pilot"
  target
  regression := { NexusCallerClosure.regression with target := ⟨"other-target"⟩ }
}

example : runInspector [incompatiblePilot] [incompatiblePilot.id] = {
    status := 1
    stdout := ""
    stderr :=
      "{\"kind\":\"incompatibleTarget\",\"subject\":\"other-target\"," ++
        "\"context\":\"nexus-caller-closure\"}\n"
  } := by
  native_decide

def impossibleTarget : ModelTarget := {
  target with resources := [{ id := clashResourceId, value := "not-the-clash" }]
}

def compileFailurePilot : Pilot := {
  id := "compile-failure-pilot"
  target := impossibleTarget
  regression := NexusCallerClosure.regression
}

example : runInspector [compileFailurePilot] [compileFailurePilot.id] = {
    status := 1
    stdout := ""
    stderr :=
      "{\"kind\":\"compileFailure\",\"subject\":\"caller-force-close\"," ++
        "\"context\":\"impossibleAction:[{\\\"resourceId\\\":\\\"caller-closure-clash\\\"," ++
        "\\\"value\\\":\\\"not-the-clash\\\"}]\"}\n"
  } := by
  native_decide

def changedOutcomeTarget : ModelTarget := {
  target with actionProjections := [{
    id := forceCloseActionId
    project := fun setup => if setup == clashSetup then some ⟨"changed-outcome"⟩ else none
  }]
}

def changedObservationTarget : ModelTarget := {
  target with propertyObservations := [
    honoredDeliveryObservation,
    {
      id := cancellationUniquenessPropertyId
      contract := "changed-observation-contract"
    }
  ]
}

def compiledIdentity (candidate : ModelTarget) : Option String :=
  (compile candidate NexusCallerClosure.regression).toOption.map ExperimentSpec.modelIdentity

def compiledJson (candidate : ModelTarget) : Option String :=
  (compile candidate NexusCallerClosure.regression).toOption.map canonicalJson

example : [changedOutcomeTarget, changedObservationTarget].all (fun candidate =>
    compiledIdentity candidate != compiledIdentity target &&
      compiledJson candidate != compiledJson target) = true := by
  native_decide

end NexusCallerClosureTests

end Temporal.ExperimentTests
