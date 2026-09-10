import Temporal.Feature.Nexus2.Lifecycle

/-! Independently authored start, cancellation, and successful-completion checks. -/

namespace Temporal.Feature.Nexus2.Cancellation

open Umpire
open Temporal.Feature.Nexus2.Lifecycle

private def id (value : String) : DefinitionId := Temporal.Shared.definitionId value

def source : SourceLocation :=
  Temporal.Shared.sourceLocation "Temporal/Feature/Nexus2/Cancellation.lean"

structure ModelVocabulary where
  scheduledState : ModelValue
  startedState : ModelValue
  canceledState : ModelValue
  succeededState : ModelValue
  cancelAction : ModelValue
  startAction : ModelValue
  reportSuccessAction : ModelValue
  startedOutcome : ModelValue
  canceledOutcome : ModelValue
  succeededOutcome : ModelValue
  startedFact : ModelValue
  canceledFact : ModelValue
  succeededFact : ModelValue
  scheduledSetup : List RoleBinding
  startedSetup : List RoleBinding
  deriving BEq, DecidableEq, Repr

/-- Vocabulary lowering stays behind the generic finite identity owner. -/
def modelVocabulary : Except FiniteTableError ModelVocabulary := do
  let model ← (table.validate.map (·.withIdentity identity))
  pure {
    scheduledState := ← model.stateValue .scheduled
    startedState := ← model.stateValue .started
    canceledState := ← model.stateValue .canceled
    succeededState := ← model.stateValue .succeeded
    cancelAction := ← model.actionValue .cancel
    startAction := ← model.actionValue .start
    reportSuccessAction := ← model.actionValue .reportSuccess
    startedOutcome := ← model.outcomeValue .started
    canceledOutcome := ← model.outcomeValue .canceled
    succeededOutcome := ← model.outcomeValue .succeeded
    startedFact := ← model.factValue .started
    canceledFact := ← model.factValue .canceled
    succeededFact := ← model.factValue .succeeded
    scheduledSetup := ← model.setupValue .scheduled
    startedSetup := ← model.setupValue .started
  }

def operationRole : Scenario.Role := { id := operationRoleId, valueKind := .state }

namespace Start

def propertyId : DefinitionId := id "temporal.nexus2.basic-lifecycle.property.start"
def behaviorId : DefinitionId := id "temporal.nexus2.basic-lifecycle.behavior.start"
def queryId : DefinitionId := id "temporal.nexus2.basic-lifecycle.query.start"
def occurrenceId : DefinitionId := id "temporal.nexus2.basic-lifecycle.occurrence.start"
def setupConstraintId : DefinitionId := id "temporal.nexus2.basic-lifecycle.setup.scheduled"

def authoredProperty (model : ModelVocabulary) : Property := {
  id := propertyId
  source
  requires := [lifecycleCapabilityId]
  clauses := [
    .transitionContract (id "temporal.nexus2.basic-lifecycle.property.start.state")
      (PropertyPattern.exact .selectedAction startActionId model.startAction.value)
      (PropertyPattern.exact .resultingState operationStateId model.startedState.value),
    .transitionContract (id "temporal.nexus2.basic-lifecycle.property.start.outcome")
      (PropertyPattern.exact .selectedAction startActionId model.startAction.value)
      (PropertyPattern.exact .outcome transitionOutcomeId model.startedOutcome.value),
    .inputOutput (id "temporal.nexus2.basic-lifecycle.property.start.fact")
      (PropertyPattern.exact .selectedAction startActionId model.startAction.value)
      (PropertyPattern.exact .observation lifecycleFactId model.startedFact.value)
  ]
  documentation := "Starting a scheduled operation yields the started lifecycle result."
}

def authoredScenario (model : ModelVocabulary) : Scenario :=
  Scenario.exactlyOneAction behaviorId source
    { id := occurrenceId, action := startActionId }
    (requires := [lifecycleCapabilityId])
    (roles := [operationRole])
    (setup := [SetupConstraint.roleEquals setupConstraintId operationRoleId model.scheduledState])

def authoredQuery
    (property : CheckedProperty) (behavior : CheckedScenario) : Query := {
  id := queryId
  source
  target := targetId
  form := .find property
  behavior
  limits := Limits.bounded 1 1 8
  policy := { strategy := .shortest, seed := 17 }
}

end Start

namespace Cancel

def propertyId : DefinitionId := id "temporal.nexus2.basic-lifecycle.property.cancel"
def behaviorId : DefinitionId := id "temporal.nexus2.basic-lifecycle.behavior.cancel"
def queryId : DefinitionId := id "temporal.nexus2.basic-lifecycle.query.cancel"
def occurrenceId : DefinitionId := id "temporal.nexus2.basic-lifecycle.occurrence.cancel"
def setupConstraintId : DefinitionId := id "temporal.nexus2.basic-lifecycle.setup.started-cancel"

def authoredProperty (model : ModelVocabulary) : Property := {
  id := propertyId
  source
  requires := [lifecycleCapabilityId]
  clauses := [
    .transitionContract (id "temporal.nexus2.basic-lifecycle.property.cancel.state")
      (PropertyPattern.exact .selectedAction cancelActionId model.cancelAction.value)
      (PropertyPattern.exact .resultingState operationStateId model.canceledState.value),
    .transitionContract (id "temporal.nexus2.basic-lifecycle.property.cancel.outcome")
      (PropertyPattern.exact .selectedAction cancelActionId model.cancelAction.value)
      (PropertyPattern.exact .outcome transitionOutcomeId model.canceledOutcome.value),
    .inputOutput (id "temporal.nexus2.basic-lifecycle.property.cancel.fact")
      (PropertyPattern.exact .selectedAction cancelActionId model.cancelAction.value)
      (PropertyPattern.exact .observation lifecycleFactId model.canceledFact.value)
  ]
  documentation := "Canceling a started operation yields the canceled lifecycle result."
}

def authoredScenario (model : ModelVocabulary) : Scenario :=
  Scenario.exactlyOneAction behaviorId source
    { id := occurrenceId, action := cancelActionId }
    (requires := [lifecycleCapabilityId])
    (roles := [operationRole])
    (setup := [SetupConstraint.roleEquals setupConstraintId operationRoleId model.startedState])

def authoredQuery
    (property : CheckedProperty) (behavior : CheckedScenario) : Query := {
  id := queryId
  source
  target := targetId
  form := .find property
  behavior
  limits := Limits.bounded 1 1 8
  policy := { strategy := .shortest, seed := 17 }
}

end Cancel

namespace Success

def propertyId : DefinitionId := id "temporal.nexus2.basic-lifecycle.property.success"
def behaviorId : DefinitionId := id "temporal.nexus2.basic-lifecycle.behavior.success"
def queryId : DefinitionId := id "temporal.nexus2.basic-lifecycle.query.success"
def occurrenceId : DefinitionId := id "temporal.nexus2.basic-lifecycle.occurrence.success"
def setupConstraintId : DefinitionId := id "temporal.nexus2.basic-lifecycle.setup.started-success"

def authoredProperty (model : ModelVocabulary) : Property := {
  id := propertyId
  source
  requires := [lifecycleCapabilityId]
  clauses := [
    .transitionContract (id "temporal.nexus2.basic-lifecycle.property.success.state")
      (PropertyPattern.exact .selectedAction reportSuccessActionId model.reportSuccessAction.value)
      (PropertyPattern.exact .resultingState operationStateId model.succeededState.value),
    .transitionContract (id "temporal.nexus2.basic-lifecycle.property.success.outcome")
      (PropertyPattern.exact .selectedAction reportSuccessActionId model.reportSuccessAction.value)
      (PropertyPattern.exact .outcome transitionOutcomeId model.succeededOutcome.value),
    .inputOutput (id "temporal.nexus2.basic-lifecycle.property.success.fact")
      (PropertyPattern.exact .selectedAction reportSuccessActionId model.reportSuccessAction.value)
      (PropertyPattern.exact .observation lifecycleFactId model.succeededFact.value)
  ]
  documentation := "Reporting success yields the succeeded lifecycle result."
}

def authoredScenario (model : ModelVocabulary) : Scenario :=
  Scenario.exactlyOneAction behaviorId source
    { id := occurrenceId, action := reportSuccessActionId }
    (requires := [lifecycleCapabilityId])
    (roles := [operationRole])
    (setup := [SetupConstraint.roleEquals setupConstraintId operationRoleId model.startedState])

def authoredQuery
    (property : CheckedProperty) (behavior : CheckedScenario) : Query := {
  id := queryId
  source
  target := targetId
  form := .find property
  behavior
  limits := Limits.bounded 1 1 8
  policy := { strategy := .shortest, seed := 17 }
}

end Success

inductive BaselineAdmissionError where
  | invalidTarget (error : TableAdmissionError)
  | invalidVocabulary (error : FiniteTableError)
  | invalidProperty (error : PropertyError)
  | invalidBehavior (error : ScenarioError)
  | invalidQuery (error : QueryError)
  | invalidPlanner (error : FiniteSearchAdmissionError)
  | invalidKnownGap (error : KnownGapError)

structure CheckedOperation where
  property : CheckedProperty
  behavior : CheckedScenario
  query : CheckedQuery LawStatement
  run : PlanResult

structure CheckedBaseline where
  target : QueryModel LawStatement
  model : ModelVocabulary
  start : CheckedOperation
  cancel : CheckedOperation
  success : CheckedOperation

private def checkOperation
    (target : QueryModel LawStatement)
    (authoredProperty : Property)
    (authoredScenario : Scenario)
    (authoredQuery : CheckedProperty → CheckedScenario → Query) :
    Except BaselineAdmissionError CheckedOperation := do
  let property ← Property.check (PropertyCheckContext.ofTarget target) (authoredProperty)
    |>.mapError BaselineAdmissionError.invalidProperty
  let behavior ← Scenario.check (.ofTarget target) authoredScenario
    |>.mapError BaselineAdmissionError.invalidBehavior
  let query ← Query.check (.ofTarget target) (authoredQuery property behavior)
    |>.mapError BaselineAdmissionError.invalidQuery
  let kernel ← SearchView.ofCheckedQuery target.id query
    |>.mapError BaselineAdmissionError.invalidPlanner
  let run ← search query kernel |>.mapError BaselineAdmissionError.invalidKnownGap
  pure { property, behavior, query, run }

/-- All three journeys proceed only through successful Target and declaration admission branches. -/
def checkBaseline : Except BaselineAdmissionError CheckedBaseline := do
  let target ← targetResult.mapError BaselineAdmissionError.invalidTarget
  let model ← modelVocabulary.mapError BaselineAdmissionError.invalidVocabulary
  let start ← checkOperation target (Start.authoredProperty model)
    (Start.authoredScenario model) Start.authoredQuery
  let cancel ← checkOperation target (Cancel.authoredProperty model)
    (Cancel.authoredScenario model) Cancel.authoredQuery
  let success ← checkOperation target (Success.authoredProperty model)
    (Success.authoredScenario model) Success.authoredQuery
  pure { target, model, start, cancel, success }

end Temporal.Feature.Nexus2.Cancellation
