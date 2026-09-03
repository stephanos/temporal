import Umpire.Property
import Umpire.Shared.Test

/-! Shared semantic vocabulary, context, clauses, traces, and helpers for the Property concern tests. -/

namespace Umpire.PropertyTests

open Umpire

def id (value : String) : DefinitionId := Shared.Test.definitionId value

def source : SourceLocation := Shared.Test.sourceLocation "Umpire/Property/Tests.lean"

def metadata (value : String) (kind : DefinitionKind) : DefinitionMetadata :=
  Shared.Test.definitionMetadata value kind source (value ++ "/v1")

def cancellationCapability : DefinitionId := id "test.capability.cancellation"
def hiddenCapability : DefinitionId := id "test.capability.hidden"

def pendingCount : DefinitionId := id "test.state.pending-count"
def cancellationPhase : DefinitionId := id "test.state.cancellation-phase"
def requestCancel : DefinitionId := id "test.action.request-cancel"
def tick : DefinitionId := id "test.action.tick"
def deliveredOutcome : DefinitionId := id "test.outcome.cancel-delivered"
def cancelRequested : DefinitionId := id "test.observation.cancel-requested"
def cancelDelivered : DefinitionId := id "test.observation.cancel-delivered"
def logicalTime : DefinitionId := id "test.observation.logical-time"
def ownsOperation : DefinitionId := id "test.relation.owns-resource"
def hiddenObservation : DefinitionId := id "test.observation.hidden-record"

def definitions : List DefinitionMetadata := [
  metadata cancellationCapability.value .capability,
  metadata hiddenCapability.value .capability,
  metadata pendingCount.value .state,
  metadata cancellationPhase.value .state,
  metadata requestCancel.value .action,
  metadata tick.value .action,
  metadata deliveredOutcome.value .outcome,
  metadata cancelRequested.value .observation,
  metadata cancelDelivered.value .observation,
  metadata logicalTime.value .observation,
  metadata ownsOperation.value .relation,
  metadata hiddenObservation.value .observation
]

def meaning
    (definitionId : DefinitionId)
    (kind : DefinitionKind) : MeaningProvision := {
  definitionId
  kind
  canonicalBehavior := definitionId.value ++ "/meaning-v1"
}

def cancellationMeanings : List MeaningProvision := [
  meaning pendingCount .state,
  meaning cancellationPhase .state,
  meaning requestCancel .action,
  meaning tick .action,
  meaning deliveredOutcome .outcome,
  meaning cancelRequested .observation,
  meaning cancelDelivered .observation,
  meaning logicalTime .observation,
  meaning ownsOperation .relation
]

def cancelBudget : PropertyLimitProfile := {
  id := id "test.limit.cancel-budget"
  source
  limit := { value := 2, unit := .observationPositions }
}

def context : PropertyCheckContext := {
  definitions
  providers := [
    { id := cancellationCapability, version := 1, canonicalBehavior := "test-cancellation/v1" },
    { id := hiddenCapability, version := 1, canonicalBehavior := "test-hidden/v1" }
  ]
  meanings :=
    cancellationMeanings.map (fun item => (cancellationCapability, item)) ++
      [(hiddenCapability, meaning hiddenObservation .observation)]
  limitProfiles := [cancelBudget]
}

def pattern
    (field : PropertyTraceField)
    (reference : DefinitionId)
    (constraint : ValueConstraint := .present) : PropertyPattern := {
  field
  reference
  constraint
}

def cancelIsUnique : PropertyClause :=
  .stateInvariant (id "test.property.cancel-is-unique")
    (pattern .state pendingCount (.naturalAtMost 1))

def deliveryContract : PropertyClause :=
  .transitionContract (id "test.property.delivery-contract")
    (pattern .selectedAction requestCancel)
    (pattern .modelOutcome deliveredOutcome (.equals "delivered"))

def ownershipIsPresent : PropertyClause :=
  .identityRelation (id "test.property.ownership")
    (pattern .relation ownsOperation (.equals "subject:resource"))

def requestHasObservation : PropertyClause :=
  .inputOutput (id "test.property.request-has-observation")
    (pattern .selectedAction requestCancel)
    (pattern .observation cancelRequested)

def requestPrecedesDelivery : PropertyClause :=
  .ordered (id "test.property.request-precedes-delivery")
    (pattern .observation cancelRequested)
    (pattern .observation cancelDelivered)
    .observationPositions

def honoredDelivery : PropertyClause :=
  .eventuallyWithin (id "test.property.honored-delivery")
    (pattern .observation cancelRequested)
    (pattern .observation cancelDelivered)
    (.named cancelBudget.id .observationPositions)

def deliveryIsQuiescent : PropertyClause :=
  .quiescentWithin (id "test.property.delivery-is-quiescent")
    (pattern .observation cancelDelivered)
    (pattern .observation cancelRequested)
    (.exact { value := 0, unit := .observationPositions })

def portableProperty : PropertyDeclaration := {
  id := id "test.property.cancellation-contract"
  source
  requires := [cancellationCapability]
  clauses := [
    cancelIsUnique,
    deliveryContract,
    ownershipIsPresent,
    requestHasObservation,
    requestPrecedesDelivery,
    honoredDelivery,
    deliveryIsQuiescent
  ]
  documentation := "Portable cancellation clauses."
}

def authoredProperty : PropertyAuthoring := .portable portableProperty

def value (definitionId : DefinitionId) (payload : String) : ModelValue :=
  ModelValue.named definitionId payload

def positiveTrace : ModelTrace ModelValue ModelValue ModelValue ModelValue := {
  initialState := value pendingCount "0"
  steps := [
    {
      selectedAction := value requestCancel "request"
      modelOutcome := value deliveredOutcome "delivered"
      resultingState := value pendingCount "1"
      observations := [
        value cancelRequested "request-1",
        value ownsOperation "subject:resource",
        value hiddenObservation "private-record"
      ]
    },
    {
      selectedAction := value tick "tick"
      modelOutcome := value deliveredOutcome "delivered"
      resultingState := value pendingCount "1"
      observations := [value cancelDelivered "request-1"]
    }
  ]
}

def evaluationOf
    (declaration : PropertyDeclaration)
    (trace : ModelTrace ModelValue ModelValue ModelValue ModelValue) :
    Option PropertyEvaluation :=
  (checkProperty context (.portable declaration)).toOption.map fun property =>
    evaluateProperty property trace

def errorKindOf
    (result : Except PropertyError CheckedProperty) : Option PropertyErrorKind :=
  match result with
  | .ok _ => none
  | .error error => some error.kind

end Umpire.PropertyTests
