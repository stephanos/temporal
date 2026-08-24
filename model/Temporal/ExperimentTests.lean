import Temporal.Experiment.Compiler
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

example : [
    errorKindAndSubject (compile validTarget duplicateOrdering),
    errorKindAndSubject (compile validTarget selfOrdering),
    errorKindAndSubject (compile validTarget cyclicOrdering)
  ] = [
    some (.duplicateOrdering, "advance->observe"),
    some (.selfOrdering, "advance->advance"),
    some (.cyclicOrdering, "advance->observe")
  ] := by
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

def identityAndJson (result : Except CompileError ExperimentSpec) : Option (String × String) :=
  result.toOption.map (fun spec => (spec.modelIdentity, canonicalJson spec))

example : identityAndJson (compile changedOutcomeTarget validRegression) ≠
    identityAndJson (compile validTarget validRegression) := by
  native_decide

example : identityAndJson (compile changedPropertyTarget validRegression) ≠
    identityAndJson (compile validTarget validRegression) := by
  native_decide

end Temporal.ExperimentTests
