import NexusAutoClose
import Temporal.Experiment.Property

namespace Temporal.Experiment.PropertyTests

open Temporal.Experiment
open NexusAutoClose

def id (value : String) : DeclarationId := DeclarationId.of value

def source : SemanticSource := {
  path := "Temporal/Experiment/PropertyTests.lean"
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

def cancellationCapability : DeclarationId := id "nexus.capability.cancellation"
def hiddenCapability : DeclarationId := id "storage.capability.internal"

def pendingCount : DeclarationId := id "nexus.state.pending-cancel-count"
def cancellationPhase : DeclarationId := id "nexus.state.cancellation-phase"
def requestCancel : DeclarationId := id "nexus.action.request-cancel"
def tick : DeclarationId := id "nexus.action.tick"
def deliveredOutcome : DeclarationId := id "nexus.outcome.cancel-delivered"
def cancelRequested : DeclarationId := id "nexus.observation.cancel-requested"
def cancelDelivered : DeclarationId := id "nexus.observation.cancel-delivered"
def logicalTime : DeclarationId := id "nexus.observation.logical-time"
def ownsOperation : DeclarationId := id "workflow-nexus.relation.owns-operation"
def hiddenObservation : DeclarationId := id "storage.observation.record-written"

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
  id := id "nexus.bound.cancel-budget"
  source
  bound := { value := 2, unit := .observationPositions }
}

def context : PropertyCheckContext := {
  declarations
  providers := [
    { id := cancellationCapability, version := 1, semanticDigest := "nexus-cancellation/v1" },
    { id := hiddenCapability, version := 1, semanticDigest := "storage-internal/v1" }
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
  .stateInvariant (id "nexus.property.cancel-is-unique")
    (pattern .state pendingCount (.naturalAtMost 1))

def deliveryContract : PropertyClause :=
  .transitionContract (id "nexus.property.delivery-contract")
    (pattern .selectedAction requestCancel)
    (pattern .modelOutcome deliveredOutcome (.equals "delivered"))

def ownershipIsPresent : PropertyClause :=
  .identityRelation (id "workflow-nexus.property.ownership")
    (pattern .relation ownsOperation (.equals "caller:operation"))

def requestHasObservation : PropertyClause :=
  .inputOutput (id "nexus.property.request-has-observation")
    (pattern .selectedAction requestCancel)
    (pattern .observation cancelRequested)

def requestPrecedesDelivery : PropertyClause :=
  .ordered (id "nexus.property.request-precedes-delivery")
    (pattern .observation cancelRequested)
    (pattern .observation cancelDelivered)
    .observationPositions

def honoredDelivery : PropertyClause :=
  .eventuallyWithin (id "nexus.property.honored-delivery")
    (pattern .observation cancelRequested)
    (pattern .observation cancelDelivered)
    (.named cancelBudget.id .observationPositions)

def deliveryIsQuiescent : PropertyClause :=
  .quiescentWithin (id "nexus.property.delivery-is-quiescent")
    (pattern .observation cancelDelivered)
    (pattern .observation cancelRequested)
    (.exact { value := 0, unit := .observationPositions })

def portableProperty : PropertyDeclaration := {
  id := id "nexus.property.caller-close-cancellation"
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
        value ownsOperation "caller:operation",
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

def negativeTrace : SemanticTrace SemanticValue SemanticValue SemanticValue SemanticValue := {
  positiveTrace with
  steps := positiveTrace.steps.mapIdx fun index step =>
    if index == 0 then { step with resultingState := value pendingCount "2" } else step
}

def uniquenessProperty : PropertyDeclaration := {
  portableProperty with
  id := id "nexus.property.uniqueness-only"
  clauses := [cancelIsUnique]
}

def uniquenessTrace
    (config : Config) : SemanticTrace SemanticValue SemanticValue SemanticValue SemanticValue := {
  initialState := value pendingCount (toString config.cancels.length)
  steps := []
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

example : errorKindOf (checkProperty context authoredProperty) = none := by
  native_decide

example : (evaluationOf portableProperty positiveTrace).map PropertyEvaluation.satisfied = some true := by
  native_decide

example : (evaluationOf portableProperty negativeTrace).map PropertyEvaluation.satisfied = some false := by
  native_decide

example : AtMostOneEvent (autoClose .upgrade wClash) :=
  upgrade_preserves_uniqueness wClash (wClash_reachable .upgrade)

example :
    (evaluationOf uniquenessProperty (uniquenessTrace (autoClose .upgrade wClash))).map
      PropertyEvaluation.satisfied = some true := by
  native_decide

example : ¬AtMostOneEvent (autoClose .duplicate wClash) := by
  simp [AtMostOneEvent, autoClose, applyResolution, wClash]

example :
    (evaluationOf uniquenessProperty (uniquenessTrace (autoClose .duplicate wClash))).map
      PropertyEvaluation.satisfied = some false := by
  native_decide

def samePositionBoundary : PropertyDeclaration := {
  portableProperty with
  id := id "nexus.property.same-position-boundary"
  clauses := [
    .eventuallyWithin (id "nexus.property.same-position-boundary.clause")
      (pattern .observation cancelDelivered)
      (pattern .observation cancelDelivered)
      (.exact { value := 0, unit := .observationPositions })
  ]
}

example :
    (evaluationOf samePositionBoundary positiveTrace).map PropertyEvaluation.satisfied = some true := by
  native_decide

/-- The reusable theorem applies to positive, negative, and boundary fixtures without a
constructor-specific proof escape hatch. -/
example (clause : ResolvedPropertyClause) :
    ∀ view : PropertyTraceView,
      evaluatePropertyClause clause view = true ↔ clause.denote view := by
  intro view
  exact evaluatePropertyClause_agrees clause view

def hiddenReference : PropertyDeclaration := {
  portableProperty with
  id := id "nexus.property.hidden-reference"
  clauses := [
    .identityRelation (id "nexus.property.hidden-reference.clause")
      (pattern .observation hiddenObservation)
  ]
}

example :
    errorKindOf (checkProperty context (.portable hiddenReference)) = some .undeclaredReference := by
  native_decide

def admittedObservationIds : Option (List DeclarationId) :=
  (checkProperty context authoredProperty).toOption.map fun property =>
    (property.traceView positiveTrace).steps.flatMap fun step =>
      step.observations.map SemanticValue.identity

example : admittedObservationIds.map (fun ids => ids.contains hiddenObservation) = some false := by
  native_decide

def traceWithoutHidden : SemanticTrace SemanticValue SemanticValue SemanticValue SemanticValue := {
  positiveTrace with
  steps := positiveTrace.steps.map fun step => {
    step with
    observations := step.observations.filter fun observation =>
      observation.identity != hiddenObservation
  }
}

example : evaluationOf portableProperty positiveTrace = evaluationOf portableProperty traceWithoutHidden := by
  native_decide

def mixedUnitProperty : PropertyDeclaration := {
  portableProperty with
  id := id "nexus.property.mixed-unit"
  clauses := [
    .eventuallyWithin (id "nexus.property.mixed-unit.clause")
      (pattern .observation cancelRequested)
      (pattern .observation cancelDelivered)
      (.named cancelBudget.id .selectedActions)
  ]
}

example :
    errorKindOf (checkProperty context (.portable mixedUnitProperty)) = some .unitMismatch := by
  native_decide

def missingLogicalTimeProperty : PropertyDeclaration := {
  portableProperty with
  id := id "nexus.property.missing-logical-time"
  clauses := [
    .eventuallyWithin (id "nexus.property.missing-logical-time.clause")
      (pattern .observation cancelRequested)
      (pattern .observation cancelDelivered)
      (.exact { value := 1, unit := .logicalTime })
  ]
}

example :
    errorKindOf (checkProperty context (.portable missingLogicalTimeProperty)) =
      some .missingLogicalTimeSource := by
  native_decide

def logicalEventuallyProperty : PropertyDeclaration := {
  portableProperty with
  id := id "nexus.property.logical-eventually"
  logicalTimeSource := some logicalTime
  clauses := [
    .eventuallyWithin (id "nexus.property.logical-eventually.clause")
      (pattern .observation cancelRequested)
      (pattern .observation cancelDelivered)
      (.exact { value := 1, unit := .logicalTime })
  ]
}

def logicalQuiescentProperty : PropertyDeclaration := {
  portableProperty with
  id := id "nexus.property.logical-quiescent"
  logicalTimeSource := some logicalTime
  clauses := [
    .quiescentWithin (id "nexus.property.logical-quiescent.clause")
      (pattern .observation cancelDelivered)
      (pattern .observation cancelRequested)
      (.exact { value := 0, unit := .logicalTime })
  ]
}

def traceWithLogicalTime
    (first second : String) :
    SemanticTrace SemanticValue SemanticValue SemanticValue SemanticValue := {
  positiveTrace with
  steps := positiveTrace.steps.mapIdx fun index step => {
    step with
    observations := step.observations ++ [value logicalTime (if index == 0 then first else second)]
  }
}

example :
    (evaluationOf logicalEventuallyProperty (traceWithLogicalTime "1" "2")).map
      PropertyEvaluation.satisfied = some true := by
  native_decide

example :
    (evaluationOf logicalQuiescentProperty (traceWithLogicalTime "1" "2")).map
      PropertyEvaluation.satisfied = some true := by
  native_decide

example :
    (evaluationOf logicalEventuallyProperty positiveTrace).map PropertyEvaluation.satisfied =
      some false := by
  native_decide

example :
    (evaluationOf logicalQuiescentProperty positiveTrace).map PropertyEvaluation.satisfied =
      some false := by
  native_decide

example :
    (evaluationOf logicalEventuallyProperty (traceWithLogicalTime "not-a-time" "2")).map
      PropertyEvaluation.satisfied = some false := by
  native_decide

example :
    (evaluationOf logicalQuiescentProperty (traceWithLogicalTime "not-a-time" "2")).map
      PropertyEvaluation.satisfied = some false := by
  native_decide

example :
    ((evaluationOf logicalEventuallyProperty (traceWithLogicalTime "2" "1")).map
        PropertyEvaluation.satisfied,
      (evaluationOf logicalQuiescentProperty (traceWithLogicalTime "2" "1")).map
        PropertyEvaluation.satisfied) = (some false, some false) := by
  native_decide

example :
    errorKindOf (checkProperty context
      (.opaque (id "nexus.property.expert-only") source)) = some .opaqueDeclaration := by
  native_decide

def reorderedContext : PropertyCheckContext := {
  context with
  declarations := context.declarations.reverse
  providers := context.providers.reverse
  meanings := context.meanings.reverse
}

def reorderedProperty : PropertyDeclaration := {
  portableProperty with
  clauses := portableProperty.clauses.reverse
}

def canonicalOf
    (check : Except PropertyError CheckedProperty) : Option String :=
  check.toOption.map canonicalPropertyJson

def digestOf
    (check : Except PropertyError CheckedProperty) : Option String :=
  check.toOption.map CheckedProperty.semanticDigest

example : canonicalOf (checkProperty context authoredProperty) =
    canonicalOf (checkProperty reorderedContext (.portable reorderedProperty)) := by
  native_decide

example : canonicalOf (checkProperty context authoredProperty) =
    canonicalOf (checkProperty context authoredProperty) := by
  rfl

def changedConstructor : PropertyDeclaration := {
  portableProperty with
  clauses := portableProperty.clauses.map fun clause =>
    if clause.id == honoredDelivery.id then
      .quiescentWithin honoredDelivery.id
        (pattern .observation cancelRequested)
        (pattern .observation cancelDelivered)
        (.exact cancelBudget.bound)
    else
      clause
}

def changedReference : PropertyDeclaration := {
  portableProperty with
  clauses := portableProperty.clauses.map fun clause =>
    if clause.id == cancelIsUnique.id then
      .stateInvariant cancelIsUnique.id
        (pattern .state cancellationPhase (.naturalAtMost 1))
    else
      clause
}

def changedBound : PropertyDeclaration := {
  portableProperty with
  clauses := portableProperty.clauses.map fun clause =>
    if clause.id == honoredDelivery.id then
      .eventuallyWithin honoredDelivery.id
        (pattern .observation cancelRequested)
        (pattern .observation cancelDelivered)
        (.exact { value := 3, unit := .observationPositions })
    else
      clause
}

example : digestOf (checkProperty context authoredProperty) ≠
    digestOf (checkProperty context (.portable changedConstructor)) := by
  native_decide

example : digestOf (checkProperty context authoredProperty) ≠
    digestOf (checkProperty context (.portable changedReference)) := by
  native_decide

example : digestOf (checkProperty context authoredProperty) ≠
    digestOf (checkProperty context (.portable changedBound)) := by
  native_decide

def focusedClauseResult : Option PropertyClauseResult := do
  let evaluation ← evaluationOf portableProperty positiveTrace
  evaluation.clauses.find? fun result => result.clauseId == honoredDelivery.id

example : focusedClauseResult.map PropertyClauseResult.evaluatedBound =
    some (some cancelBudget.bound) := by
  native_decide

example : focusedClauseResult.map (fun result =>
    result.traceSpan.isSome && !result.semanticProvenance.isEmpty) = some true := by
  native_decide

end Temporal.Experiment.PropertyTests
