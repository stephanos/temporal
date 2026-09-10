import Temporal.Feature.Nexus2.Cancellation
import Temporal.Feature.Nexus2.Race
import Umpire.Property.Authoring
import Umpire.Behavior.Authoring
import Umpire.Query.Authoring

/-! Constructor comparison specimens over the already admitted Nexus2 Targets. -/

namespace Temporal.Feature.Nexus2.Authoring

open Umpire

namespace Baseline

open Temporal.Feature.Nexus2.Lifecycle

def family : DefinitionFamily := {
  root := Temporal.Shared.definitionId "temporal.nexus2.basic-lifecycle"
}

def propertySpec
    (key : String)
    (action state outcome fact : ModelValue) : PropertySpec := {
  family
  key
  source := Cancellation.source
  requires := [lifecycleCapabilityId]
  clauses := stepClauses family key action state outcome fact
}

def behaviorSpec
    (key setupKey occurrenceKey : String)
    (initialState action : ModelValue) : ExactSequenceSpec := {
  family
  key
  source := Cancellation.source
  requires := [lifecycleCapabilityId]
  roles := [Cancellation.operationRole]
  setup := [SetupConstraint.roleEquals (family.id "setup" setupKey)
    operationRoleId initialState]
  occurrences := [{ key := occurrenceKey, action := action.definitionId }]
}

def querySpec
    (key : String)
    (property : CheckedProperty)
    (behavior : CheckedBehavior) : QuerySpec := {
  family
  key
  source := Cancellation.source
  target := targetId
  form := .witness property
  behavior
  limits := { transitions := 1, selectedActions := 1, candidateEvaluations := 8 }
  policy := .shortest
}

inductive AdmissionError where
  | invalidTarget (error : FiniteTargetAdmissionError)
  | invalidVocabulary (error : FiniteTableError)
  | invalidProperty (error : PropertyError)
  | invalidBehavior (error : BehaviorError)
  | invalidQuery (error : QueryError)

structure CheckedOperation where
  property : CheckedProperty
  behavior : CheckedBehavior
  query : CheckedQuery LawStatement

structure CheckedBaseline where
  target : QueryTarget LawStatement
  model : Cancellation.ModelVocabulary
  start : CheckedOperation
  cancel : CheckedOperation
  success : CheckedOperation

private def checkOperation
    (target : QueryTarget LawStatement)
    (property : PropertySpec)
    (behavior : ExactSequenceSpec)
    (queryKey : String) : Except AdmissionError CheckedOperation := do
  let property ← property.check (PropertyCheckContext.ofTarget target)
    |>.mapError AdmissionError.invalidProperty
  let behavior ← behavior.check (.ofTarget target)
    |>.mapError AdmissionError.invalidBehavior
  let query ← (querySpec queryKey property behavior).check target
    |>.mapError AdmissionError.invalidQuery
  pure { property, behavior, query }

/-- Constructor values publish only after every existing language checker succeeds. -/
def checkBaseline : Except AdmissionError CheckedBaseline := do
  let target ← targetResult.mapError AdmissionError.invalidTarget
  let model ← Cancellation.modelVocabulary.mapError AdmissionError.invalidVocabulary
  let start ← checkOperation target
    (propertySpec "start" model.startAction model.startedState model.startedOutcome model.startedFact)
    (behaviorSpec "start" "scheduled" "start" model.scheduledState model.startAction) "start"
  let cancel ← checkOperation target
    (propertySpec "cancel" model.cancelAction model.canceledState model.canceledOutcome model.canceledFact)
    (behaviorSpec "cancel" "started-cancel" "cancel" model.startedState model.cancelAction) "cancel"
  let success ← checkOperation target
    (propertySpec "success" model.reportSuccessAction model.succeededState
      model.succeededOutcome model.succeededFact)
    (behaviorSpec "success" "started-success" "success" model.startedState
      model.reportSuccessAction) "success"
  pure { target, model, start, cancel, success }

end Baseline

namespace GuardedRace

open Temporal.Feature.Nexus2.Race

def family : DefinitionFamily := {
  root := Temporal.Shared.definitionId "temporal.nexus2.cancellation-race"
}

private def sameStep
    (key : String)
    (expectation : PropertyPredicate) : PropertySameStepClause := {
  id := family.id "case" key
  source
  expectation
}

def requestCase (model : ModelVocabulary) : PropertyCase := {
  id := family.id "case" "request"
  source
  guard := .selectedActionIs model.requestCancelAction
  clauses := [sameStep "request.state" (.resultingStateIs model.cancelRequestedState)]
  temporalClauses := [.eventuallyWithin
    (family.id "case" "request.terminal") source
    (.selectedAction model.requestCancelAction) (.fact model.terminalFact)
    (.exact { value := 1, unit := .semanticTransitions })]
}

def resolutionCase (model : ModelVocabulary) : PropertyCase := {
  id := family.id "case" "resolve"
  source
  guard := .selectedActionIs model.resolveAction
  clauses := [sameStep "resolve.terminal" (.factIs model.terminalFact)]
}

def propertySpec (model : ModelVocabulary) : PropertySpec := {
  family
  key := "cases"
  source
  version := 2
  requires := [capabilityId]
  clauses := [.sameStepCases {
    id := family.id "case-group" "lifecycle"
    source
    guard := .any [
      .selectedActionIs model.requestCancelAction,
      .selectedActionIs model.resolveAction
    ]
    cases := [requestCase model, resolutionCase model]
    complete := true
    exclusive := true
  }]
}

def behaviorSpec (model : ModelVocabulary) : ExactSequenceSpec := {
  family
  key := "request-then-resolve"
  source
  requires := [capabilityId]
  roles := [operationRole]
  setup := [SetupConstraint.roleEquals (family.id "setup" "started")
    operationRoleId model.startedState]
  occurrences := [
    { key := "request", action := model.requestCancelAction.definitionId },
    { key := "resolution", action := model.resolveAction.definitionId }
  ]
}

def querySpec
    (property : CheckedProperty)
    (behavior : CheckedBehavior) : QuerySpec := {
  family
  key := "case-analysis"
  source
  target := targetId
  form := .select [property]
  behavior
  limits := { transitions := 2, selectedActions := 2, candidateEvaluations := 32 }
  policy := .exhaustive
}

/-- Separate alternative specimen: the exception is explicit and has no replacement case. -/
def withoutReplacement (model : ModelVocabulary) : PropertySpec :=
  let request := requestCase model
  { propertySpec model with clauses := [.sameStepCases {
    id := family.id "case-group" "missing-replacement"
    source
    guard := .selectedActionIs model.requestCancelAction
    cases := [{ request with
      exception := some {
        id := family.id "exception" "request.already-cancel-requested"
        source
        condition := .priorStateIs model.startedState
      }
    }]
  }] }

end GuardedRace

end Temporal.Feature.Nexus2.Authoring
