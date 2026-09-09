import Temporal.Feature.Nexus3.Authoring

/-! The success-slice command grammar and its expansion into typed Authoring declarations. -/

namespace Temporal.Feature.Nexus3

open Umpire

macro "model" name:ident "role" role:ident
    "states" stateType:ident
    "actions" actionType:ident "outcomes" outcomeType:ident "facts" factType:ident
    "initial" "[" initial:ident "]" "terminal" "[" terminal:ident "]" "transitions"
    startRel:ident ":" scheduled:ident "+" awaitStart:ident "→"
      "{" "state" ":=" started:ident "," "outcome" ":=" acknowledged:ident ","
        "facts" ":=" "[" startedFact:ident "]" "}"
    successRel:ident ":" startedSource:ident "+" awaitSuccess:ident "→"
      "{" "state" ":=" succeeded:ident "," "outcome" ":=" completed:ident ","
        "facts" ":=" "[" succeededFact:ident "]" "}" : command => do
    let spellings := [role.getId, stateType.getId, actionType.getId, outcomeType.getId,
      factType.getId, initial.getId, terminal.getId, startRel.getId, scheduled.getId,
      awaitStart.getId, started.getId, acknowledged.getId, startedFact.getId, successRel.getId,
      startedSource.getId, awaitSuccess.getId, succeeded.getId, completed.getId,
      succeededFact.getId]
    unless spellings == [`operation, `State, `Action, `Outcome, `Fact, `scheduled,
        `succeeded, `start, `scheduled, `awaitStart, `started, `acknowledged, `started, `success,
        `started, `awaitSuccess, `succeeded, `completed, `succeeded] do
      Lean.Macro.throwError "unsupported Nexus3 success model spelling"
    let key := Lean.quote name.getId.toString
    let roleKey := Lean.quote role.getId.toString
    let setupKey := Lean.quote initial.getId.toString
    let scheduledKey := Lean.quote scheduled.getId.toString
    let startedKey := Lean.quote started.getId.toString
    let succeededKey := Lean.quote succeeded.getId.toString
    let awaitStartKey := Lean.quote awaitStart.getId.toString
    let awaitSuccessKey := Lean.quote awaitSuccess.getId.toString
    let acknowledgedKey := Lean.quote acknowledged.getId.toString
    let completedKey := Lean.quote completed.getId.toString
    let startedFactKey := Lean.quote startedFact.getId.toString
    let succeededFactKey := Lean.quote succeededFact.getId.toString
    let startRelationKey := Lean.quote startRel.getId.toString
    let successRelationKey := Lean.quote successRel.getId.toString
    let setupScheduled := Lean.mkIdentFrom name `Setup.scheduled
    let stateScheduled := Lean.mkIdentFrom name `State.scheduled
    let stateStarted := Lean.mkIdentFrom name `State.started
    let stateSucceeded := Lean.mkIdentFrom name `State.succeeded
    let actionAwaitStart := Lean.mkIdentFrom name `Action.awaitStart
    let actionAwaitSuccess := Lean.mkIdentFrom name `Action.awaitSuccess
    let outcomeAcknowledged := Lean.mkIdentFrom name `Outcome.acknowledged
    let outcomeCompleted := Lean.mkIdentFrom name `Outcome.completed
    let factStarted := Lean.mkIdentFrom name `Fact.started
    let factSucceeded := Lean.mkIdentFrom name `Fact.succeeded
    `(command| def $name := Authoring.successModel {
        declaration := $key
        roleName := $roleKey
        setup := $setupKey
        states := [$scheduledKey, $startedKey, $succeededKey]
        actions := [$awaitStartKey, $awaitSuccessKey]
        outcomes := [$acknowledgedKey, $completedKey]
        facts := [$startedFactKey, $succeededFactKey]
      }
        ($setupScheduled)
        [($stateScheduled), ($stateStarted), ($stateSucceeded)]
        [($actionAwaitStart), ($actionAwaitSuccess)]
        [($outcomeAcknowledged), ($outcomeCompleted)]
        [($factStarted), ($factSucceeded)]
        [($stateScheduled)] [($stateSucceeded)]
        [{ key := $startRelationKey, source := ($stateScheduled), action := ($actionAwaitStart),
            results := [Authoring.successResult ($outcomeAcknowledged) ($stateStarted)
              ($factStarted)] },
          { key := $successRelationKey, source := ($stateStarted),
            action := ($actionAwaitSuccess),
            results := [Authoring.successResult ($outcomeCompleted) ($stateSucceeded)
              ($factSucceeded)] }]
        (by exact ⟨rfl, rfl, rfl⟩))

macro "property" name:ident "on" modelRef:ident "for" roleRef:ident "when" "action" actionRef:ident
    "require" stateClause:ident ":" "resultingState" stateRef:ident
    "require" outcomeClause:ident ":" "outcome" outcomeRef:ident
    "require" factClause:ident ":" "fact" factRef:ident : command => do
    unless [roleRef.getId, actionRef.getId, stateClause.getId,
        stateRef.getId, outcomeClause.getId, outcomeRef.getId, factClause.getId, factRef.getId] ==
      [`operation, `awaitSuccess, `successState, `succeeded,
        `successOutcome, `completed, `successFact, `succeeded] do
      Lean.Macro.throwError "unsupported Nexus3 success Property spelling"
    let ownerKey := Lean.quote name.getId.toString
    let stateClauseKey := Lean.quote stateClause.getId.toString
    let outcomeClauseKey := Lean.quote outcomeClause.getId.toString
    let factClauseKey := Lean.quote factClause.getId.toString
    `(command| def $name (values : Authoring.ModelVocabulary) : PropertySpec :=
        Authoring.propertySpec ($modelRef) values {
          declaration := $ownerKey
          stateClause := $stateClauseKey
          outcomeClause := $outcomeClauseKey
          factClause := $factClauseKey
        })

macro "behavior" name:ident "on" modelRef:ident roleRef:ident "starts" setupRef:ident
    "actions" "exactly" "[" startLabel:ident ":" startAction:ident ","
      completionLabel:ident ":" completionAction:ident "]" : command => do
    unless [roleRef.getId, setupRef.getId, startLabel.getId,
        startAction.getId, completionLabel.getId, completionAction.getId] ==
      [`operation, `scheduled, `start, `awaitStart,
        `completion, `awaitSuccess] do
      Lean.Macro.throwError "unsupported Nexus3 success Behavior spelling"
    let ownerKey := Lean.quote name.getId.toString
    let roleKey := Lean.quote roleRef.getId.toString
    let startKey := Lean.quote startLabel.getId.toString
    let completionKey := Lean.quote completionLabel.getId.toString
    `(command| def $name (values : Authoring.ModelVocabulary) : ExactSequenceSpec :=
        Authoring.behaviorSpec ($modelRef) values {
          declaration := $ownerKey
          roleName := $roleKey
          startOccurrence := $startKey
          completionOccurrence := $completionKey
        })

macro "limits" name:ident "transitions" transitionCount:num "selected_actions" actionCount:num
    "candidate_evaluations" candidateCount:num : command => do
    unless transitionCount.raw.isNatLit? == some 2 && actionCount.raw.isNatLit? == some 2 &&
        candidateCount.raw.isNatLit? == some 16 do
      Lean.Macro.throwError "unsupported Nexus3 success Limits spelling"
    `(command| def $name : QueryLimitSpec := QueryLimitSpec.mk 2 2 16)

macro "query" name:ident "on" modelRef:ident "witness" propertyRef:ident "in" behaviorRef:ident
    "limits" limitsRef:ident : command => do
    let queryKey := Lean.quote name.getId.toString
    `(command| def $name : Except Authoring.AdmissionError (Authoring.CheckedModel ($modelRef)) :=
        Authoring.check ($modelRef) $queryKey ($limitsRef) ($propertyRef) ($behaviorRef))

macro "query" _name:ident "on" _modelRef:ident "all" _propertyRef:ident
    "in" _behaviorRef:ident "limits" _limitsRef:ident : command =>
  Lean.Macro.throwError "unsupported Nexus3 success Query spelling"

end Temporal.Feature.Nexus3
