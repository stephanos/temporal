import Temporal.Feature.Nexus.Lifecycle.Model
import Temporal.Shared
import Umpire.Search
import Umpire.Search.Branches

/-! Typed finite authoring of the focused Nexus lifecycle under an independent identity root. -/

namespace Temporal.Feature.Nexus2.Lifecycle

open Umpire

private def id (value : String) : DefinitionId := Temporal.Shared.definitionId value

def source : SourceLocation :=
  Temporal.Shared.sourceLocation "Temporal/Feature/Nexus2/Lifecycle.lean"

def targetId : DefinitionId := id "temporal.nexus2.basic-lifecycle.target"
def kernelId : DefinitionId := id "temporal.nexus2.basic-lifecycle.kernel"
def lifecycleCapabilityId : DefinitionId := id "temporal.nexus2.basic-lifecycle.capability"
def lifecycleProviderId : DefinitionId := id "temporal.nexus2.basic-lifecycle.provider"
def lifecycleLawId : DefinitionId := id "temporal.nexus2.basic-lifecycle.law.authoritative-table"
def operationStateId : DefinitionId := id "temporal.nexus2.basic-lifecycle.state.operation"
def startActionId : DefinitionId := id "temporal.nexus2.basic-lifecycle.action.start"
def cancelActionId : DefinitionId := id "temporal.nexus2.basic-lifecycle.action.cancel"
def reportSuccessActionId : DefinitionId := id "temporal.nexus2.basic-lifecycle.action.succeed"
def transitionOutcomeId : DefinitionId := id "temporal.nexus2.basic-lifecycle.outcome.transition"
def lifecycleFactId : DefinitionId := id "temporal.nexus2.basic-lifecycle.fact.state"
def operationRoleId : DefinitionId := id "temporal.nexus2.basic-lifecycle.role.operation"

inductive Setup where
  | scheduled
  | started
  deriving BEq, DecidableEq, Repr

inductive State where
  | scheduled
  | started
  | canceled
  | succeeded
  deriving BEq, DecidableEq, Repr

inductive Action where
  | cancel
  | start
  | reportSuccess
  deriving BEq, DecidableEq, Repr

inductive Outcome where
  | started
  | canceled
  | succeeded
  deriving BEq, DecidableEq, Repr

inductive Fact where
  | started
  | canceled
  | succeeded
  deriving BEq, DecidableEq, Repr

def startedResult : Step State Outcome Fact := {
  outcome := .started
  state := .started
  facts := [.started]
}

def canceledResult : Step State Outcome Fact := {
  outcome := .canceled
  state := .canceled
  facts := [.canceled]
}

def succeededResult : Step State Outcome Fact := {
  outcome := .succeeded
  state := .succeeded
  facts := [.succeeded]
}

/-- The complete authored model. Catalog order is the explicit deterministic planning order. -/
def table : FiniteTable Setup State Action Outcome Fact := {
  setups := [
    { value := .scheduled, key := "scheduled" },
    { value := .started, key := "started" }
  ]
  states := [
    { value := .scheduled, key := "scheduled" },
    { value := .started, key := "started" },
    { value := .canceled, key := "canceled" },
    { value := .succeeded, key := "succeeded" }
  ]
  actions := [
    { value := .cancel, key := "cancel" },
    { value := .start, key := "start" },
    { value := .reportSuccess, key := "handler-reports-success" }
  ]
  outcomes := [
    { value := .started, key := "started" },
    { value := .canceled, key := "canceled" },
    { value := .succeeded, key := "succeeded" }
  ]
  facts := [
    { value := .started, key := "started" },
    { value := .canceled, key := "canceled" },
    { value := .succeeded, key := "succeeded" }
  ]
  initial := [
    { setup := .scheduled, states := [.scheduled] },
    { setup := .started, states := [.started] }
  ]
  transitions := [
    { key := "cancel", source := .started, action := .cancel, results := [canceledResult] },
    { key := "start", source := .scheduled, action := .start, results := [startedResult] },
    { key := "report-success", source := .started, action := .reportSuccess,
      results := [succeededResult] }
  ]
}

/-- Stable Target identities are explicit and separate from the typed semantic vocabulary. -/
def identity : FiniteModelIdentity Setup State Action Outcome Fact := {
  setupBindings := fun setup => [{
    roleId := operationRoleId
    state := match setup with | .scheduled => .scheduled | .started => .started
  }]
  stateId := fun _ => operationStateId
  actionId := fun
    | .cancel => cancelActionId
    | .start => startActionId
    | .reportSuccess => reportSuccessActionId
  outcomeId := fun _ => transitionOutcomeId
  factId := fun _ => lifecycleFactId
}

/-- One independently stated required row; unrelated extensions do not change this obligation. -/
private def hasExactTransition
    (rows : List (FiniteTransitionRow State Action Outcome Fact))
    (source : State)
    (action : Action)
    (result : Step State Outcome Fact) : Bool :=
  rows.any fun row =>
    row.source == source && row.action == action && row.results == [result]

/-- Independently stated product requirements for the three focused lifecycle transitions. -/
def satisfiesLifecycleRequirement
    (rows : List (FiniteTransitionRow State Action Outcome Fact)) : Bool :=
  hasExactTransition rows .started .cancel {
    outcome := .canceled
    state := .canceled
    facts := [.canceled]
  } &&
  hasExactTransition rows .scheduled .start {
    outcome := .started
    state := .started
    facts := [.started]
  } &&
  hasExactTransition rows .started .reportSuccess {
    outcome := .succeeded
    state := .succeeded
    facts := [.succeeded]
  }

/-- The capability law binds provider metadata to the independently authored lifecycle requirement. -/
def LawStatement (law : Law) : Prop :=
  law.id = lifecycleLawId ∧
    law.body = "temporal-nexus2-basic-lifecycle-authoritative-table/v1" ∧
    satisfiesLifecycleRequirement table.transitions = true

def lifecycleLaw : Law := {
  id := lifecycleLawId
  body := "temporal-nexus2-basic-lifecycle-authoritative-table/v1"
}

theorem lifecycleLawProof : LawStatement lifecycleLaw := ⟨rfl, rfl, rfl⟩

private def metadata
    (definitionId : DefinitionId)
    (kind : DefinitionKind)
    (behaviorVersion : String) : DefinitionMetadata :=
  Temporal.Shared.definitionMetadata definitionId kind source behaviorVersion

def lifecycleProvider : Provider LawStatement := {
  id := lifecycleProviderId
  source
  contract := {
    id := lifecycleCapabilityId
    behaviorVersion := "temporal-nexus2-basic-lifecycle/v1"
    requiredLaws := [lifecycleLaw]
  }
  meanings := [
    { definitionId := operationStateId, kind := .state,
      behaviorVersion := "temporal-nexus2-basic-lifecycle-state/v1" },
    { definitionId := startActionId, kind := .action,
      behaviorVersion := "temporal-nexus2-basic-lifecycle-start/v1" },
    { definitionId := cancelActionId, kind := .action,
      behaviorVersion := "temporal-nexus2-basic-lifecycle-cancel/v1" },
    { definitionId := reportSuccessActionId, kind := .action,
      behaviorVersion := "temporal-nexus2-basic-lifecycle-report-success/v1" },
    { definitionId := transitionOutcomeId, kind := .outcome,
      behaviorVersion := "temporal-nexus2-basic-lifecycle-outcome/v1" },
    { definitionId := lifecycleFactId, kind := .fact,
      behaviorVersion := "temporal-nexus2-basic-lifecycle-fact/v1" }
  ]
  lawProofs := [{ definition := lifecycleLaw, proof := lifecycleLawProof }]
}

def definitions : List DefinitionMetadata := [
  metadata targetId .target "temporal-nexus2-basic-lifecycle-target/v1",
  metadata kernelId .machine "temporal-nexus2-basic-lifecycle-kernel/v1",
  metadata lifecycleCapabilityId .capability "temporal-nexus2-basic-lifecycle/v1",
  metadata lifecycleProviderId .provider "temporal-nexus2-basic-lifecycle-provider/v1",
  metadata lifecycleLawId .law lifecycleLaw.body,
  metadata operationStateId .state "temporal-nexus2-basic-lifecycle-state/v1",
  metadata startActionId .action "temporal-nexus2-basic-lifecycle-start/v1",
  metadata cancelActionId .action "temporal-nexus2-basic-lifecycle-cancel/v1",
  metadata reportSuccessActionId .action "temporal-nexus2-basic-lifecycle-report-success/v1",
  metadata transitionOutcomeId .outcome "temporal-nexus2-basic-lifecycle-outcome/v1",
  metadata lifecycleFactId .fact "temporal-nexus2-basic-lifecycle-fact/v1"
]

def modelSpec : TableModelSpec := {
  id := targetId
  source
  definitions
  requiredCapabilities := [lifecycleCapabilityId]
  metadata := { id := kernelId, source }
}

def modelProviders : Providers LawStatement :=
  Providers.empty |>.provide lifecycleProvider

/-- Target admission validates the typed table before exposing any checked semantic value. -/
def targetResult : Except TableAdmissionError (QueryModel LawStatement) :=
  table.checkModel identity modelSpec modelProviders

def establishedState : State → Temporal.Feature.Nexus.Lifecycle.OperationState
  | .scheduled => .scheduled
  | .started => .started
  | .canceled => .canceled
  | .succeeded => .succeeded

def establishedAction : Action → Temporal.Feature.Nexus.Lifecycle.OperationEvent
  | .cancel => .cancel
  | .start => .start
  | .reportSuccess => .succeed

def establishedOutcomeKey : Outcome → String
  | .started => Temporal.Feature.Nexus.Lifecycle.startedOutcome.value
  | .canceled => Temporal.Feature.Nexus.Lifecycle.canceledOutcome.value
  | .succeeded => Temporal.Feature.Nexus.Lifecycle.succeededOutcome.value

def establishedFactKey : Fact → String
  | .started => Temporal.Feature.Nexus.Lifecycle.startedObservation.value
  | .canceled => Temporal.Feature.Nexus.Lifecycle.canceledObservation.value
  | .succeeded => Temporal.Feature.Nexus.Lifecycle.succeededObservation.value

def establishedSetup : Setup → List RoleBinding
  | .scheduled => Temporal.Feature.Nexus.Lifecycle.scheduledSetup
  | .started => Temporal.Feature.Nexus.Lifecycle.startedSetup

end Temporal.Feature.Nexus2.Lifecycle
