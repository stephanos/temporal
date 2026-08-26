import Umpire.Property

/-! Shared semantic vocabulary, context, clauses, traces, and helpers for the Property concern tests. -/

namespace Umpire.PropertyTests

open Umpire

def id (value : String) : DeclarationId := DeclarationId.of value

def source : SemanticSource := {
  path := "Umpire/Property/Tests.lean"
  line := 1
  column := 1
  provenance := "lean-test"
}

def metadata (value : String) (kind : DeclarationKind) : DeclarationMetadata := {
  id := id value
  kind
  source
  contractDigest := value ++ "/v1"
}

def cancellationCapability : DeclarationId := id "test.capability.cancellation"
def hiddenCapability : DeclarationId := id "test.capability.hidden"

def pendingCount : DeclarationId := id "test.state.pending-count"
def cancellationPhase : DeclarationId := id "test.state.cancellation-phase"
def requestCancel : DeclarationId := id "test.action.request-cancel"
def tick : DeclarationId := id "test.action.tick"
def deliveredOutcome : DeclarationId := id "test.outcome.cancel-delivered"
def cancelRequested : DeclarationId := id "test.observation.cancel-requested"
def cancelDelivered : DeclarationId := id "test.observation.cancel-delivered"
def logicalTime : DeclarationId := id "test.observation.logical-time"
def ownsOperation : DeclarationId := id "test.relation.owns-resource"
def hiddenObservation : DeclarationId := id "test.observation.hidden-record"

def declarations : List DeclarationMetadata := [
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
    (declaration : DeclarationId)
    (kind : DeclarationKind) : MeaningProvision := {
  declaration
  kind
  semanticDigest := declaration.value ++ "/meaning-v1"
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

def cancelBudget : PropertyBoundProfile := {
  id := id "test.bound.cancel-budget"
  source
  bound := { value := 2, unit := .observationPositions }
}

def context : PropertyCheckContext := {
  declarations
  providers := [
    { id := cancellationCapability, version := 1, semanticDigest := "test-cancellation/v1" },
    { id := hiddenCapability, version := 1, semanticDigest := "test-hidden/v1" }
  ]
  meanings :=
    cancellationMeanings.map (fun item => (cancellationCapability, item)) ++
      [(hiddenCapability, meaning hiddenObservation .observation)]
  boundProfiles := [cancelBudget]
}

def pattern
    (field : PropertyTraceField)
    (reference : DeclarationId)
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

def value (identity : DeclarationId) (payload : String) : SemanticValue := {
  identity
  value := payload
}

def positiveTrace : SemanticTrace SemanticValue SemanticValue SemanticValue SemanticValue := {
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
    (trace : SemanticTrace SemanticValue SemanticValue SemanticValue SemanticValue) :
    Option PropertyEvaluation :=
  (checkProperty context (.portable declaration)).toOption.map fun property =>
    evaluateProperty property trace

def errorKindOf
    (result : Except PropertyError CheckedProperty) : Option PropertyErrorKind :=
  match result with
  | .ok _ => none
  | .error error => some error.kind

end Umpire.PropertyTests
