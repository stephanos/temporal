import Temporal.System.Execution.Nexus
import Temporal.System.Nexus.ImplementationLink
import Temporal.Tool.RunEvaluation.Protocol
import Umpire.Artifact.PortableEvaluationContract

/-!
Lean owns the semantic specialization from one checked caller-closure Test into the portable
Evaluation Contract vocabulary. The Go boundary receives only the resulting canonical ProtoJSON;
it does not select model definitions or interpret richer checked constructs.
-/

namespace Temporal.Tool.PortableEvaluationContract

open Umpire
open Umpire.Artifact.PortableEvaluationContract

structure NonPortableError where
  sourceDefinitionId : DefinitionId
  source : SourceLocation
  construct : String
  deriving BEq, DecidableEq, Repr

private def nonPortable
    (sourceDefinitionId : DefinitionId)
    (source : SourceLocation)
    (construct : String) : Except NonPortableError α :=
  .error { sourceDefinitionId, source, construct }

private def idLe (left right : DefinitionId) : Bool :=
  decide (left.value ≤ right.value)

private def sourceLe (left right : SourceLocation) : Bool :=
  decide (left.path < right.path) ||
    (left.path == right.path && decide (left.line < right.line)) ||
    (left.path == right.path && left.line == right.line && decide (left.column < right.column)) ||
    (left.path == right.path && left.line == right.line && left.column == right.column &&
      decide (left.provenance ≤ right.provenance))

private def portableDefinitionKind
    (owner : DefinitionId)
    (source : SourceLocation) : Umpire.DefinitionKind → Except NonPortableError PortableDefinitionKind
  | .state => pure .state
  | .action => pure .action
  | .outcome => pure .outcome
  | .observation => pure .observation
  | .relation => pure .relation
  | .capability => pure .capability
  | kind => nonPortable owner source ("definition-kind." ++ kind.name)

private def definitionBinding
    (owner : DefinitionId)
    (source : SourceLocation)
    (definitions : List DefinitionMetadata)
    (id : DefinitionId)
    (kind : Umpire.DefinitionKind) : Except NonPortableError DefinitionBinding := do
  let some definition := definitions.find? fun candidate =>
      candidate.id == id && candidate.kind == kind
    | nonPortable owner source ("missing-definition." ++ id.value)
  pure {
    definitionId := id
    behaviorFingerprint := implementationSemanticFingerprint definition definition.canonicalBehavior
  }

/-- Lower only version-one Observation operators present in the protobuf interpreter table. -/
def lowerObservationExpression
    (sourceDefinitionId : DefinitionId)
    (source : SourceLocation) :
    CheckedObservationExpression → Except NonPortableError
      Umpire.Artifact.PortableEvaluationContract.ObservationExpression
  | .text value => pure (.literalText value)
  | .natural value => pure (.literalNatural value)
  | .field reference _ _ => pure (.field reference)
  | .normalize .naturalRenderV1 operand =>
      return .naturalRenderV1 (← lowerObservationExpression sourceDefinitionId source operand)
  | .present operand =>
      return .present (← lowerObservationExpression sourceDefinitionId source operand)
  | .equals left right =>
      return .equals
        (← lowerObservationExpression sourceDefinitionId source left)
        (← lowerObservationExpression sourceDefinitionId source right)
  | .and left right => do
      let loweredLeft ← lowerObservationExpression sourceDefinitionId source left
      let loweredRight ← lowerObservationExpression sourceDefinitionId source right
      let operands := match loweredRight with
        | .all rest => loweredLeft :: rest
        | _ => [loweredLeft, loweredRight]
      pure (.all operands)
  | .or left right => do
      let loweredLeft ← lowerObservationExpression sourceDefinitionId source left
      let loweredRight ← lowerObservationExpression sourceDefinitionId source right
      let operands := match loweredRight with
        | .any rest => loweredLeft :: rest
        | _ => [loweredLeft, loweredRight]
      pure (.any operands)
  | .boolean _ => nonPortable sourceDefinitionId source "observation.literal-boolean"
  | .binding id _ _ => nonPortable sourceDefinitionId source ("observation.binding." ++ id.value)
  | .normalize .textTrimV1 _ =>
      nonPortable sourceDefinitionId source "observation.normalize.text-trim-v1"
  | .normalize .textLowercaseV1 _ =>
      nonPortable sourceDefinitionId source "observation.normalize.text-lowercase-v1"
  | .not _ => nonPortable sourceDefinitionId source "observation.not"
  | .contributionMarker _ =>
      nonPortable sourceDefinitionId source "observation.contribution-marker"
  | .digestToken policy _ =>
      nonPortable sourceDefinitionId source ("observation.digest-token." ++ policy.id.value)

private def firstEvidenceKind : CheckedObservationExpression → Option DefinitionId
  | .field reference _ _ => some reference.kind
  | .normalize _ operand | .present operand | .not operand |
      .contributionMarker operand | .digestToken _ operand => firstEvidenceKind operand
  | .equals left right | .and left right | .or left right =>
      firstEvidenceKind left <|> firstEvidenceKind right
  | .text _ | .natural _ | .boolean _ | .binding _ _ _ => none

private def sourceForKind
    (owner : DefinitionId)
    (source : SourceLocation)
    (kind : DefinitionId) : Except NonPortableError DefinitionId :=
  if kind == Temporal.System.Nexus.Observation.Profile.cleanupKind then
    pure (DefinitionId.of "umpire.evidence.source.cleanup")
  else if kind == Temporal.System.Nexus.Observation.Profile.controlReceiptKind then
    pure (DefinitionId.of "umpire.evidence.source.control-receipt")
  else if kind == Temporal.System.Nexus.Observation.Profile.historyKind then
    pure (DefinitionId.of "umpire.evidence.source.history")
  else if kind == Temporal.System.Nexus.Observation.Profile.participantKind then
    pure (DefinitionId.of "umpire.evidence.source.participant-output")
  else
    nonPortable owner source ("evidence-kind-source." ++ kind.value)

private def portableDisposition
    (owner : DefinitionId)
    (source : SourceLocation) :
    Umpire.FieldDisposition → Except NonPortableError
      (Umpire.Artifact.PortableEvaluationContract.FieldDisposition × Option DefinitionId)
  | .retain => pure (.retain, none)
  | .redact => pure (.redact, none)
  | .reject => pure (.reject, none)
  | .hash (some policy) => pure (.hash, some policy)
  | .hash none => nonPortable owner source "evidence.hash-without-digest-policy"

private def lowerEvidenceKind
    (plan : CheckedObservationPlan)
    (kind : Umpire.EvidenceKindDeclaration) :
    Except NonPortableError
      Umpire.Artifact.PortableEvaluationContract.EvidenceKindDeclaration := do
  let fields ← kind.fields.mapM fun field => do
    let reference : EvidenceFieldReference := { kind := kind.id, field := field.id }
    let some declaredDisposition := plan.dispositions.find? fun item => item.field == reference
      | nonPortable plan.id plan.source ("evidence.missing-disposition." ++ field.id.value)
    let (disposition, digestPolicyDefinitionId) ←
      portableDisposition plan.id plan.source declaredDisposition.disposition
    pure {
      fieldDefinitionId := field.id
      valueType := field.valueType
      disposition
      digestPolicyDefinitionId
    }
  pure {
    kindDefinitionId := kind.id
    sourceDefinitionId := ← sourceForKind plan.id plan.source kind.id
    fields
  }

private def referenceLe (left right : EvidenceFieldReference) : Bool :=
  decide (left.kind.value < right.kind.value) ||
    (left.kind == right.kind && decide (left.field.value ≤ right.field.value))

private def referencesForField
    (plan : CheckedObservationPlan)
    (field : DefinitionId) : List EvidenceFieldReference :=
  let references := plan.profile.kinds.filterMap fun kind =>
    if kind.fields.any fun candidate => candidate.id == field then
      some { kind := kind.id, field }
    else none
  references.mergeSort referenceLe

private def correlationSlots (plan : CheckedObservationPlan) : List CorrelationSlot :=
  [{
    definitionId := DefinitionId.of "umpire.evidence.correlation.operation"
    kind := .operation
    fields := referencesForField plan Temporal.System.Nexus.Observation.Profile.operationCorrelationField
  }, {
    definitionId := DefinitionId.of "umpire.evidence.correlation.run"
    kind := .run
    fields := referencesForField plan Temporal.System.Nexus.Observation.Profile.runCorrelationField
  }, {
    definitionId := DefinitionId.of "umpire.evidence.correlation.workflow"
    kind := .workflow
    fields := referencesForField plan Temporal.System.Nexus.Observation.Profile.workflowCorrelationField
  }]

private def lowerEvidenceProfile
    (plan : CheckedObservationPlan)
    (profileFingerprint : BehaviorFingerprint) : Except NonPortableError EvidenceProfile := do
  let sources ← plan.closures.mapM fun closure => sourceForKind plan.id plan.source closure.kind
  let kinds ← plan.profile.kinds.mapM (lowerEvidenceKind plan)
  pure {
    definition := { definitionId := plan.profile.id, behaviorFingerprint := profileFingerprint }
    version := plan.profile.version
    sources := sources.mergeSort idLe |>.eraseDups
    kinds
    digestPolicies := plan.digestPolicies.map fun policy => {
      definitionId := policy.id
      algorithm := .syntheticDigestV1
    }
    cardinalities := plan.profile.kinds.map fun kind => {
      kindDefinitionId := kind.id
      minimum := 0
      maximum := plan.evidenceBound.value
    }
    correlationSlots := correlationSlots plan
  }

private def ruleCoordinate
    (plan : CheckedObservationPlan)
    (rule : CheckedObservationRule) : Except NonPortableError
      Umpire.Artifact.PortableEvaluationContract.ModelCoordinate :=
  if rule.output == Temporal.System.Nexus.CallerClosure.actionId then
    pure { field := .selectedAction, step := 1 }
  else if rule.output == Temporal.System.Nexus.CallerClosure.outcomeId then
    pure { field := .modelOutcome, step := 1 }
  else if rule.output == Temporal.System.Nexus.CallerClosure.deliveryObservationId then
    pure { field := .observation, step := 1, position := 1 }
  else if rule.output == Temporal.System.Nexus.CallerClosure.cancellationCountObservationId then
    pure { field := .observation, step := 1, position := 2 }
  else if rule.output == Temporal.System.Nexus.CallerClosure.ownershipObservationId then
    pure { field := .observation, step := 1, position := 3 }
  else
    nonPortable plan.id plan.source ("observation.coordinate." ++ rule.output.value)

private def lowerOrdinaryRule
    (plan : CheckedObservationPlan)
    (definitions : List DefinitionMetadata)
    (rule : CheckedObservationRule) : Except NonPortableError Emit := do
  let some condition := rule.condition
    | nonPortable plan.id plan.source ("observation.unconditional-rule." ++ rule.id.value)
  let some sourceKind := firstEvidenceKind condition
    | nonPortable plan.id plan.source ("observation.source-kind." ++ rule.id.value)
  let outputKind ← portableDefinitionKind plan.id plan.source rule.outputKind
  pure {
    definitionId := rule.id.value
    sourceKindDefinitionId := sourceKind
    outputDefinition := ← definitionBinding plan.id plan.source definitions rule.output rule.outputKind
    outputKind
    coordinate := ← ruleCoordinate plan rule
    condition := ← lowerObservationExpression plan.id plan.source condition
    value := ← lowerObservationExpression plan.id plan.source rule.value
  }

private def lowerStateRule
    (plan : CheckedObservationPlan)
    (definitions : List DefinitionMetadata)
    (rule : CheckedObservationRule) : Except NonPortableError (List Emit) := do
  let outputDefinition ← definitionBinding plan.id plan.source definitions rule.output .state
  let field : EvidenceFieldReference := {
    kind := Temporal.System.Nexus.Observation.Profile.historyKind
    field := Temporal.System.Nexus.Observation.Profile.eventTypeField
  }
  let emit (suffix expected : String)
      (coordinate : Umpire.Artifact.PortableEvaluationContract.ModelCoordinate) : Emit := {
    definitionId := rule.id.value ++ suffix
    sourceKindDefinitionId := field.kind
    outputDefinition
    outputKind := .state
    coordinate
    condition := .equals (.field field) (.literalText expected)
    value := .field field
  }
  pure [
    emit ".initial" "temporal.history.WorkflowExecutionStarted" { field := .initialState },
    emit ".resulting" "temporal.history.WorkflowExecutionCanceled"
      { field := .resultingState, step := 1 }
  ]

private def emitLe (left right : Emit) : Bool :=
  decide (left.definitionId ≤ right.definitionId)

private def orderingLe (left right : EmitOrdering) : Bool :=
  decide (left.predecessorEmitDefinitionId < right.predecessorEmitDefinitionId) ||
    (left.predecessorEmitDefinitionId == right.predecessorEmitDefinitionId &&
      decide (left.successorEmitDefinitionId ≤ right.successorEmitDefinitionId))

private def assembleObservationProgram
    (plan : CheckedObservationPlan)
    (program : Temporal.System.Execution.Nexus.ObservationProgramDefinition)
    (emits : List Emit) : Except NonPortableError ObservationProgram := do
  let stateRuleId := (plan.rules.find? fun rule =>
    rule.output == Temporal.System.Nexus.CallerClosure.stateId).map (·.id.value)
      |>.getD "missing-state-rule"
  let actionRuleId := (plan.rules.find? fun rule =>
    rule.output == Temporal.System.Nexus.CallerClosure.actionId).map (·.id.value)
      |>.getD "missing-action-rule"
  let retained := emits.map Emit.definitionId
  let ordering := (({
    predecessorEmitDefinitionId := stateRuleId ++ ".initial"
    successorEmitDefinitionId := actionRuleId
  } :: plan.ordering.map fun item => {
    predecessorEmitDefinitionId := if item.before.value == stateRuleId then
      stateRuleId ++ ".resulting" else item.before.value
    successorEmitDefinitionId := if item.after.value == stateRuleId then
      stateRuleId ++ ".resulting" else item.after.value
  }).filter fun item => retained.contains item.predecessorEmitDefinitionId &&
      retained.contains item.successorEmitDefinitionId).mergeSort orderingLe
  pure {
    definition := {
      definitionId := program.reference.definitionId
      behaviorFingerprint := program.reference.behaviorFingerprint
    }
    source := plan.source
    mapping := { definitionId := plan.id, behaviorFingerprint := plan.behaviorFingerprint }
    mappingVersion := plan.version
    profile := ← lowerEvidenceProfile plan program.profile.behaviorFingerprint
    emits
    ordering
  }

private def lowerObservationProgram
    (plan : CheckedObservationPlan)
    (program : Temporal.System.Execution.Nexus.ObservationProgramDefinition)
    (definitions : List DefinitionMetadata) : Except NonPortableError ObservationProgram := do
  if !plan.bindings.isEmpty then
    nonPortable plan.id plan.source "observation.bindings"
  let nested ← plan.rules.mapM fun rule =>
    if rule.output == Temporal.System.Nexus.CallerClosure.stateId then
      lowerStateRule plan definitions rule
    else
      return [← lowerOrdinaryRule plan definitions rule]
  let emits := nested.flatten.mergeSort emitLe
  assembleObservationProgram plan program emits

private def lowerOrdinaryRuleWithFailures
    (plan : CheckedObservationPlan)
    (definitions : List DefinitionMetadata)
    (rule : CheckedObservationRule) : Except NonPortableError
      (Option Emit × List NonPortableError) := do
  let some condition := rule.condition
    | nonPortable plan.id plan.source ("observation.unconditional-rule." ++ rule.id.value)
  let outputKind ← portableDefinitionKind plan.id plan.source rule.outputKind
  let outputDefinition ← definitionBinding plan.id plan.source definitions rule.output rule.outputKind
  let coordinate ← ruleCoordinate plan rule
  let loweredCondition := lowerObservationExpression plan.id plan.source condition
  let loweredValue := lowerObservationExpression plan.id plan.source rule.value
  let sourceKind := firstEvidenceKind condition
  match loweredCondition, loweredValue with
  | .error conditionFailure, .error valueFailure =>
      pure (none, [conditionFailure, valueFailure])
  | .error failure, .ok _ | .ok _, .error failure => pure (none, [failure])
  | .ok portableCondition, .ok value =>
      let some sourceKind := sourceKind
        | nonPortable plan.id plan.source ("observation.source-kind." ++ rule.id.value)
      pure (some {
        definitionId := rule.id.value
        sourceKindDefinitionId := sourceKind
        outputDefinition
        outputKind
        coordinate
        condition := portableCondition
        value
      }, [])

private def lowerStateRuleWithFailures
    (plan : CheckedObservationPlan)
    (definitions : List DefinitionMetadata)
    (rule : CheckedObservationRule) : Except NonPortableError
      (Option (List Emit) × List NonPortableError) := do
  let some condition := rule.condition
    | nonPortable plan.id plan.source ("observation.unconditional-rule." ++ rule.id.value)
  let loweredCondition := lowerObservationExpression plan.id plan.source condition
  let loweredValue := lowerObservationExpression plan.id plan.source rule.value
  match loweredCondition, loweredValue with
  | .error conditionFailure, .error valueFailure =>
      pure (none, [conditionFailure, valueFailure])
  | .error failure, .ok _ | .ok _, .error failure => pure (none, [failure])
  | .ok _, .ok _ => pure (some (← lowerStateRule plan definitions rule), [])

private def lowerObservationProgramWithFailures
    (plan : CheckedObservationPlan)
    (program : Temporal.System.Execution.Nexus.ObservationProgramDefinition)
    (definitions : List DefinitionMetadata) : Except NonPortableError
      (ObservationProgram × List NonPortableError) := do
  if !plan.bindings.isEmpty then
    nonPortable plan.id plan.source "observation.bindings"
  let lowered ← plan.rules.mapM fun rule =>
    if rule.output == Temporal.System.Nexus.CallerClosure.stateId then
      lowerStateRuleWithFailures plan definitions rule
    else do
      let (emit, failures) ← lowerOrdinaryRuleWithFailures plan definitions rule
      pure (emit.map (· :: []), failures)
  let emits := lowered.flatMap (fun item => item.1.getD []) |>.mergeSort emitLe
  let failures := lowered.flatMap Prod.snd
  pure (← assembleObservationProgram plan program emits, failures)

private def duplicateParticipantKind : DefinitionId :=
  DefinitionId.of "umpire.evidence.kind.participant-command.synthetic-duplicate"

private partial def rebindExpressionKind
    (fromKind toKind : DefinitionId) :
    Umpire.Artifact.PortableEvaluationContract.ObservationExpression →
      Umpire.Artifact.PortableEvaluationContract.ObservationExpression
  | .literalText value => .literalText value
  | .literalNatural value => .literalNatural value
  | .field reference =>
      .field { reference with kind := if reference.kind == fromKind then toKind else reference.kind }
  | .naturalRenderV1 operand => .naturalRenderV1 (rebindExpressionKind fromKind toKind operand)
  | .present operand => .present (rebindExpressionKind fromKind toKind operand)
  | .equals left right =>
      .equals (rebindExpressionKind fromKind toKind left)
        (rebindExpressionKind fromKind toKind right)
  | .all operands => .all (operands.map (rebindExpressionKind fromKind toKind))
  | .any operands => .any (operands.map (rebindExpressionKind fromKind toKind))

private def specializeDuplicateObservationProgram
    (program : ObservationProgram) : Except NonPortableError ObservationProgram := do
  let participantKind := Temporal.System.Nexus.Observation.Profile.participantKind
  let some participant := program.profile.kinds.find? fun kind =>
      kind.kindDefinitionId == participantKind
    | nonPortable program.mapping.definitionId program.source
        "evidence.missing-duplicate-participant-kind"
  let duplicateKind := { participant with kindDefinitionId := duplicateParticipantKind }
  let kinds := (duplicateKind :: program.profile.kinds).mergeSort fun left right =>
    decide (left.kindDefinitionId.value ≤ right.kindDefinitionId.value)
  let cardinalities := ({
    kindDefinitionId := duplicateParticipantKind
    minimum := 0
    maximum := program.profile.cardinalities.foldl (fun maximum cardinality =>
      Nat.max maximum cardinality.maximum) 0
  } :: program.profile.cardinalities).mergeSort fun left right =>
    decide (left.kindDefinitionId.value ≤ right.kindDefinitionId.value)
  let correlationSlots := program.profile.correlationSlots.map fun slot => {
    slot with fields := slot.fields.flatMap fun reference =>
      if reference.kind == participantKind then
        [reference, { reference with kind := duplicateParticipantKind }]
      else [reference]
  }
  let cancellationRuleId :=
    Temporal.System.Nexus.Observation.DuplicateDelivery.Mapping.cancellationCountRuleId.value
  let emits := program.emits.map fun emit =>
    if emit.definitionId == cancellationRuleId then {
      emit with
      sourceKindDefinitionId := duplicateParticipantKind
      condition := rebindExpressionKind participantKind duplicateParticipantKind emit.condition
      value := rebindExpressionKind participantKind duplicateParticipantKind emit.value
    } else emit
  pure {
    program with
    profile := { program.profile with kinds, cardinalities, correlationSlots }
    emits
  }

private def modelValue
    (owner : DefinitionId)
    (source : SourceLocation)
    (definitions : List DefinitionMetadata)
    (kind : Umpire.DefinitionKind)
    (value : Umpire.ModelValue) : Except NonPortableError
      Umpire.Artifact.PortableEvaluationContract.ModelValue := do
  pure {
    definition := ← definitionBinding owner source definitions value.definitionId kind
    kind := ← portableDefinitionKind owner source kind
    value := match kind with
      | .observation =>
          match value.value.toNat? with
          | some natural => .natural natural
          | none => .text value.value
      | _ => .text value.value
  }

private def entryLe (left right : RenameExactEntry) : Bool :=
  decide (left.source.definition.definitionId.value < right.source.definition.definitionId.value) ||
    (left.source.definition.definitionId == right.source.definition.definitionId &&
      decide (reprStr left.source.value ≤ reprStr right.source.value))

private def definitionEntryLe (left right : DefinitionRenameEntry) : Bool :=
  decide (left.source.definitionId.value ≤ right.source.definitionId.value)

private def lowerImplementationLink
    (duplicateDelivery : Bool) : Except NonPortableError RenameExactLink := do
  let link := Temporal.System.Nexus.ImplementationLink.CallerClosure.checked
  let declaration := link.declaration
  if !declaration.setupKnownGaps.isEmpty || !declaration.stateKnownGaps.isEmpty ||
      !declaration.actionKnownGaps.isEmpty || !declaration.outcomeKnownGaps.isEmpty ||
      !declaration.observationKnownGaps.isEmpty || !declaration.relationKnownGaps.isEmpty ||
      !declaration.capabilityKnownGaps.isEmpty then
    nonPortable declaration.id declaration.source "implementation-link.known-gap"
  let states ← declaration.stateMappings.mapM fun mapping => do
    let source ← modelValue declaration.id declaration.source
      link.sourceTarget.definitions .state mapping.source
    let destination ← modelValue declaration.id declaration.source
      link.destinationTarget.definitions .state mapping.destination
    pure ({ source := source, destination := destination } : RenameExactEntry)
  let actions ← declaration.actionMappings.mapM fun mapping => do
    let source ← modelValue declaration.id declaration.source
      link.sourceTarget.definitions .action mapping.source
    let destination ← modelValue declaration.id declaration.source
      link.destinationTarget.definitions .action mapping.destination
    pure ({ source := source, destination := destination } : RenameExactEntry)
  let outcomes ← declaration.outcomeMappings.mapM fun mapping => do
    let source ← modelValue declaration.id declaration.source
      link.sourceTarget.definitions .outcome mapping.source
    let destination ← modelValue declaration.id declaration.source
      link.destinationTarget.definitions .outcome mapping.destination
    pure ({ source := source, destination := destination } : RenameExactEntry)
  let observations ← declaration.observationMappings.mapM fun mapping => do
    let loweredSource ← modelValue declaration.id declaration.source
      link.sourceTarget.definitions .observation mapping.source
    let source := if mapping.source.definitionId ==
        Temporal.System.Nexus.CallerClosure.cancellationCountObservationId then
      { loweredSource with value := .text mapping.source.value }
    else loweredSource
    let destination ← modelValue declaration.id declaration.source
      link.destinationTarget.definitions .observation mapping.destination
    pure ({ source := source, destination := destination } : RenameExactEntry)
  let duplicateEntries ← if duplicateDelivery then do
    let loweredSource ← modelValue declaration.id declaration.source link.sourceTarget.definitions
      .observation
      Temporal.System.Nexus.ImplementationLink.CallerClosure.DuplicateDelivery.sourceCancellationCountTwo
    let duplicateSource :=
      Temporal.System.Nexus.ImplementationLink.CallerClosure.DuplicateDelivery.sourceCancellationCountTwo
    let source := { loweredSource with value := .text duplicateSource.value }
    let destination ← modelValue declaration.id declaration.source link.destinationTarget.definitions
      .observation
      Temporal.System.Nexus.ImplementationLink.CallerClosure.DuplicateDelivery.destinationCancellationCountTwo
    pure [({ source := source, destination := destination } : RenameExactEntry)]
  else pure []
  let definitionEntries := (declaration.relationMappings ++ declaration.capabilityMappings).map
    fun mapping => {
      source := {
        definitionId := mapping.source.id
        behaviorFingerprint := mapping.source.behaviorFingerprint
      }
      kind := if mapping.source.kind == .relation then .relation else .capability
      destination := {
        definitionId := mapping.destination.id
        behaviorFingerprint := mapping.destination.behaviorFingerprint
      }
    }
  let definitionId := if duplicateDelivery then
    Temporal.System.Nexus.ImplementationLink.CallerClosure.DuplicateDelivery.observedImplementationLinkId
  else declaration.id
  let behaviorFingerprint := if duplicateDelivery then
    Temporal.System.Nexus.ImplementationLink.CallerClosure.DuplicateDelivery.behaviorFingerprint
  else link.behaviorFingerprint
  pure {
    definition := {
      definitionId
      behaviorFingerprint
    }
    source := declaration.source
    sourceTarget := {
      definitionId := declaration.sourceTarget.id
      behaviorFingerprint := declaration.sourceTarget.behaviorFingerprint
    }
    destinationTarget := {
      definitionId := declaration.destinationTarget.id
      behaviorFingerprint := declaration.destinationTarget.behaviorFingerprint
    }
    entries := (states ++ actions ++ outcomes ++ observations ++ duplicateEntries).mergeSort entryLe
    definitionEntries := definitionEntries.mergeSort definitionEntryLe
    applicationLimit := {
      value := declaration.applicationLimit.value
      unit := declaration.applicationLimit.unit.name
    }
  }

private def requirementBinding
    (property : CheckedProperty)
    (link : RenameExactLink)
    (requirement : DefinitionId) : Except NonPortableError DefinitionBinding :=
  match link.definitionEntries.find? fun entry => entry.destination.definitionId == requirement with
  | some entry => pure entry.destination
  | none => nonPortable property.id property.source ("property.requirement." ++ requirement.value)

private def pattern
    (property : CheckedProperty)
    (_link : RenameExactLink)
    (value : PropertyPattern) : Except NonPortableError Pattern := do
  let field ← match value.field with
    | .state | .priorState => pure TraceField.priorState
    | .resultingState => pure TraceField.resultingState
    | .selectedAction => pure TraceField.selectedAction
    | .modelOutcome => pure TraceField.modelOutcome
    | .observation | .relation => pure TraceField.observation
  let kind := value.field.definitionKind
  let definition ← definitionBinding property.id property.source
    Temporal.Feature.Nexus.Experimental.CallerClosure.target.definitions value.reference kind
  let operator ← match value.constraint with
    | .equals expected => pure (PatternOperator.equalsText expected)
    | .naturalAtMost maximum => pure (PatternOperator.naturalAtMost maximum)
    | .present => nonPortable property.id property.source "property.pattern.present"
    | .notEquals _ => nonPortable property.id property.source "property.pattern.not-equals"
    | .naturalAtLeast _ =>
        nonPortable property.id property.source "property.pattern.natural-at-least"
  pure { field, definition, operator }

private def lowerClause
    (property : CheckedProperty)
    (link : RenameExactLink) : ResolvedPropertyClause → Except NonPortableError
      Umpire.Artifact.PortableEvaluationContract.PropertyClause
  | .transitionContract id trigger required => do
      let trigger ← pattern property link trigger
      let required ← pattern property link required
      pure { definitionId := id.value, provenance := .transitionContract, trigger, required }
  | .inputOutput id trigger required => do
      let trigger ← pattern property link trigger
      let required ← pattern property link required
      pure { definitionId := id.value, provenance := .inputOutput, trigger, required }
  | clause => nonPortable property.id property.source ("property.clause." ++ clause.id.value)

private def lowerProperty
    (checked : CheckedProperty)
    (artifact : PortableProperty)
    (link : RenameExactLink) : Except NonPortableError Property := do
  if checked.id != artifact.definitionId ||
      checked.behaviorFingerprint != artifact.behaviorFingerprint ||
      checked.requires != artifact.requirementDefinitionIds then
    nonPortable checked.id checked.source "property.artifact-binding"
  pure {
    definition := {
      definitionId := checked.id
      behaviorFingerprint := checked.behaviorFingerprint
    }
    source := checked.source
    requirements := ← checked.requires.mapM (requirementBinding checked link)
    clauses := ← checked.clauses.mapM (lowerClause checked link)
  }

private def portableKnownGapKind : Umpire.KnownGapKind → PortableKnownGapKind
  | .capabilityContract => .capabilityContract
  | .input => .input
  | .interpretation => .interpretation
  | .claim => .claim

private def portableKnownGap (gap : Umpire.KnownGap) :
    Umpire.Artifact.PortableEvaluationContract.KnownGap := {
  kind := portableKnownGapKind gap.kind
  code := gap.code.value
  subject := gap.subject.map DefinitionId.value |>.getD ""
  detail := gap.detail.getD ""
}

namespace Internal

/-- Lower phase-owned Known Gaps only after composing them through the checked semantic boundary. -/
def lowerKnownGaps
    (sourceDefinitionId : DefinitionId)
    (source : SourceLocation)
    (left right : KnownGapSet) :
    Except NonPortableError (List Umpire.Artifact.PortableEvaluationContract.KnownGap) := do
  let gaps ← (KnownGapSet.union left right).mapError fun _ => {
    sourceDefinitionId
    source
    construct := "known-gaps.conflict"
  }
  pure (gaps.toList.map portableKnownGap)

end Internal

private def fallbackSource : SourceLocation := {
  path := "Temporal/Tool/PortableEvaluationContract.lean"
  line := 1
  column := 1
  provenance := "lean-model"
}

private def limits
    (experiment : ExperimentSpec)
    (runtimeConfiguration : RuntimeConfiguration)
    (plan : CheckedObservationPlan) : EvaluationLimits := {
  maxContractBytes := 1024 * 1024
  maxInputBytes := 16 * 1024 * 1024
  maxEvidenceRecords := plan.evidenceBound.value
  maxExpressionDepth := 64
  maxCollectionItems := 10000
  maxNatural := 4294967295
  maxEvaluationWork := Nat.max 1000 (experiment.plan.expandedLimits.search.value * 100000)
  maxDiagnosticBytes := 64 * 1024
  maxResultBytes := 4 * 1024 * 1024
  maxTotalDurationMilliseconds := runtimeConfiguration.phaseLimits.foldl
    (fun total limit => total + limit.durationMilliseconds) 0
  maxOperatorCount := 10000
}

private def lowerCheckedTest
    (duplicateDelivery : Bool)
    (experiment : ExperimentSpec)
    (runtimeConfiguration : RuntimeConfiguration)
    (plan : CheckedObservationPlan)
    (program : Temporal.System.Execution.Nexus.ObservationProgramDefinition) :
    Except NonPortableError Contract := do
  let link ← lowerImplementationLink duplicateDelivery
  let source := experiment.provenance.sourceLocations.getD 0 fallbackSource
  let artifactProperty ← match experiment.properties with
    | [property] => pure property
    | _ => nonPortable experiment.plan.queryDefinitionId source "test.property-closure"
  let property ← lowerProperty
    Temporal.Feature.Nexus.Experimental.CallerClosure.callerClosureProperty artifactProperty link
  let query : DefinitionBinding := {
    definitionId := experiment.plan.queryDefinitionId
    behaviorFingerprint := experiment.plan.queryBehaviorFingerprint
  }
  let knownGaps ← Internal.lowerKnownGaps experiment.plan.queryDefinitionId source
    experiment.plan.knownGaps runtimeConfiguration.knownGaps
  let definitionSources :=
    (Temporal.System.Nexus.CallerClosure.target.definitions ++
      Temporal.Feature.Nexus.Experimental.CallerClosure.target.definitions).map (·.source)
  let provenance := (experiment.provenance.sourceLocations ++
    runtimeConfiguration.provenance.sourceLocations ++ [
      plan.source,
      link.source,
      Temporal.System.Nexus.CallerClosure.target.source,
      Temporal.Feature.Nexus.Experimental.CallerClosure.target.source,
      property.source
    ] ++ definitionSources)
      |>.mergeSort sourceLe |>.eraseDups
  let observation ← lowerObservationProgram plan program
    Temporal.System.Nexus.CallerClosure.target.definitions
  let observation ← if duplicateDelivery then
    specializeDuplicateObservationProgram observation
  else pure observation
  pure {
    contractId := experiment.plan.queryDefinitionId.value ++ ".evaluation-contract"
    experiment := experiment.artifactBinding
    runtimeConfig := runtimeConfiguration.artifactBinding
    test := {
      definitionId := DefinitionId.of (experiment.plan.queryDefinitionId.value ++ ".test")
      behaviorFingerprint := behaviorFingerprintOf
        (canonicalExperimentSpecBytes experiment ++
          canonicalRuntimeConfigurationBytes runtimeConfiguration)
    }
    query
    limits := limits experiment runtimeConfiguration plan
    observation
    implementationLink := link
    properties := [property]
    knownGaps
    provenance := if provenance.isEmpty then [source] else provenance
  }

private def syntheticBinding (id : DefinitionId) (canonicalBehavior : String) : DefinitionBinding := {
  definitionId := id
  behaviorFingerprint := behaviorFingerprintOf canonicalBehavior
}

private def portableModelValue
    (owner : DefinitionId)
    (source : SourceLocation)
    (definitions : List DefinitionMetadata)
    (value : Umpire.ModelValue) : Except NonPortableError
      Umpire.Artifact.PortableEvaluationContract.ModelValue := do
  let some definition := definitions.find? fun candidate => candidate.id == value.definitionId
    | nonPortable owner source ("missing-definition." ++ value.definitionId.value)
  modelValue owner source definitions definition.kind value

private def syntheticModelValue
    (value : Umpire.ModelValue)
    (kind : PortableDefinitionKind)
    (semanticKind : String) : Umpire.Artifact.PortableEvaluationContract.ModelValue := {
  definition := syntheticBinding value.definitionId
    ("umpire-portable-" ++ semanticKind ++ "/v1:" ++ value.definitionId.value)
  kind
  value := .text value.value
}

private def portableRoleBinding
    (owner : DefinitionId)
    (source : SourceLocation)
    (definitions : List DefinitionMetadata)
    (binding : Umpire.RoleBinding) : Except NonPortableError PortableRoleBinding := do
  pure {
    role := syntheticBinding binding.role ("umpire-portable-role/v1:" ++ binding.role.value)
    value := ← portableModelValue owner source definitions binding.value
  }

private def portableSymbolicRole (role : Umpire.ResourceRole) :
    Except NonPortableError PortableSymbolicRole := do
  pure {
    definition := syntheticBinding role.id ("umpire-portable-role/v1:" ++ role.id.value)
    kind := ← portableDefinitionKind role.id fallbackSource role.valueKind
  }

private def portableOperand
    (owner : DefinitionId)
    (source : SourceLocation)
    (definitions : List DefinitionMetadata) : Umpire.SetupOperand →
    Except NonPortableError ExecutionOperand
  | .role id => pure (.role (syntheticBinding id ("umpire-portable-role/v1:" ++ id.value)))
  | .value value => return .literal (← portableModelValue owner source definitions value)

private def portablePrecondition
    (owner : DefinitionId)
    (source : SourceLocation)
    (definitions : List DefinitionMetadata)
    (precondition : Umpire.SetupConstraint) : Except NonPortableError ExecutionPrecondition := do
  pure {
    definition := syntheticBinding precondition.id
      ("umpire-portable-precondition/v1:" ++ precondition.id.value)
    operator := match precondition.relation with
      | .equal => .equals
      | .different => .notEquals
    left := ← portableOperand owner source definitions precondition.left
    right := ← portableOperand owner source definitions precondition.right
  }

private def portableOccurrence (occurrence : Umpire.PlannedOccurrence) :
    Except NonPortableError PortablePlannedOccurrence := do
  let some authoredDefinitionId := occurrence.authoredDefinitionId
    | nonPortable occurrence.definitionId fallbackSource "occurrence.authored-definition"
  pure {
    definition := syntheticBinding occurrence.definitionId
      ("umpire-portable-occurrence/v1:" ++ occurrence.definitionId.value)
    actionDefinitionId := occurrence.actionDefinitionId
    position := occurrence.position
    authoredDefinitionId
  }

private def portableCheckpoint
    (owner : DefinitionId)
    (source : SourceLocation)
    (definitions : List DefinitionMetadata)
    (checkpoint : Umpire.ObservationCheckpoint) : Except NonPortableError
      PortableExecutionCheckpoint := do
  pure {
    transition := checkpoint.transition
    observations := ← checkpoint.observations.mapM (portableModelValue owner source definitions)
  }

private def portablePhaseLimit (limit : Umpire.PhaseLimit) : PortableExecutionPhaseLimit := {
  phase := limit.phase
  durationMilliseconds := limit.durationMilliseconds
  maxAttempts := limit.maxAttempts
  maxRecords := limit.maxRecords
  maxBytes := limit.maxBytes
}

private def portableCapabilityBinding
    (owner : DefinitionId)
    (source : SourceLocation)
    (link : RenameExactLink)
    (id : DefinitionId) : Except NonPortableError DefinitionBinding :=
  match link.definitionEntries.find? fun entry => entry.destination.definitionId == id with
  | some entry => pure entry.destination
  | none => nonPortable owner source ("missing-capability." ++ id.value)

private def portableParticipant
    (owner : DefinitionId)
    (source : SourceLocation)
    (link : RenameExactLink)
    (participant : Umpire.ParticipantBinding) : Except NonPortableError
      PortableParticipantBinding := do
  pure {
    participant := syntheticBinding participant.participantDefinitionId
      ("umpire-portable-participant/v1:" ++ participant.participantDefinitionId.value)
    protocol := syntheticBinding participant.protocolDefinitionId
      ("umpire-portable-participant-protocol/v1:" ++ participant.protocolDefinitionId.value)
    protocolVersion := participant.protocolVersion
    program := {
      definitionId := participant.programDefinitionId
      behaviorFingerprint := participant.programBehaviorFingerprint
    }
    capabilities := ← participant.capabilityDefinitionIds.mapM
      (portableCapabilityBinding owner source link)
  }

private def portableRuntimeProgram
    (owner : DefinitionId)
    (source : SourceLocation)
    (link : RenameExactLink)
    (runtimeConfiguration : RuntimeConfiguration)
    (observation : ObservationProgram) : Except NonPortableError PortableRuntimeProgram := do
  pure {
    authorityProfile := {
      definitionId := runtimeConfiguration.authorityProfile.definitionId
      behaviorFingerprint := runtimeConfiguration.authorityProfile.behaviorFingerprint
    }
    config := {
      definitionId := runtimeConfiguration.configurationDefinitionId
      behaviorFingerprint := runtimeConfiguration.behaviorFingerprint
    }
    participantBindings := ← runtimeConfiguration.participantBindings.mapM
      (portableParticipant owner source link)
    observationConfig := {
      profile := observation.profile.definition
      program := observation.definition
      mapping := observation.mapping
    }
    phaseLimits := runtimeConfiguration.phaseLimits.map portablePhaseLimit
    termination := syntheticBinding (DefinitionId.of "umpire.execution.termination.complete")
      "umpire-portable-termination-complete/v1"
    cleanup := syntheticBinding (DefinitionId.of "umpire.execution.cleanup.complete")
      "umpire-portable-cleanup-complete/v1"
    authorityRequiredCapabilities := ←
      runtimeConfiguration.authorityProfile.requiredCapabilityDefinitionIds.mapM
        (portableCapabilityBinding owner source link)
  }

private def portablePlanSelectionReason : Umpire.SelectionReason → PlanSelectionReason
  | .satisfyingWitness => .satisfyingWitness
  | .violatingCounterexample => .violatingCounterexample
  | .behaviorSelection => .behaviorSelection

private def portableArtifactProvenance (provenance : Umpire.ArtifactProvenance) :
    PlanArtifactProvenance := {
  sourceDefinitionIds := provenance.sourceDefinitionIds
  sourceLocations := provenance.sourceLocations
}

private def portableArtifactProjection
    (experiment : ExperimentSpec)
    (runtimeConfiguration : RuntimeConfiguration) : PlanArtifactProjection := {
  expandedLimits := {
    maxSemanticTransitions := experiment.plan.expandedLimits.behavior.transitions.value
    maxSelectedActions := experiment.plan.expandedLimits.behavior.selectedActions.value
    maxCandidateEvaluations := experiment.plan.expandedLimits.search.value
  }
  selectionReason := portablePlanSelectionReason experiment.plan.selectionReason
  explored := {
    setups := experiment.plan.explored.setups
    traces := experiment.plan.explored.traces
    transitions := experiment.plan.explored.transitions
    propertyEvaluations := experiment.plan.explored.propertyEvaluations
  }
  experimentKnownGaps := experiment.plan.knownGaps.toList.map portableKnownGap
  experimentProvenance := portableArtifactProvenance experiment.plan.provenance
  runtimeKnownGaps := runtimeConfiguration.knownGaps.toList.map portableKnownGap
  runtimeProvenance := portableArtifactProvenance runtimeConfiguration.provenance
  experimentObservationRequirementDefinitionIds :=
    experiment.observationRequirementDefinitionIds
  runtimeObservationConfig := {
    profile := {
      definitionId := runtimeConfiguration.observation.profileDefinitionId
      behaviorFingerprint := runtimeConfiguration.observation.profileBehaviorFingerprint
    }
    program := {
      definitionId := runtimeConfiguration.observation.programDefinitionId
      behaviorFingerprint := runtimeConfiguration.observation.programBehaviorFingerprint
    }
    mapping := {
      definitionId := runtimeConfiguration.observation.mappingDefinitionId
      behaviorFingerprint := runtimeConfiguration.observation.mappingBehaviorFingerprint
    }
  }
}

private def portableExecutionProgram
    (experiment : ExperimentSpec)
    (runtimeConfiguration : RuntimeConfiguration)
    (contract : Contract) : Except NonPortableError PortableExecutionProgram := do
  let owner := experiment.plan.queryDefinitionId
  let source := experiment.provenance.sourceLocations.getD 0 fallbackSource
  let definitions := Temporal.Feature.Nexus.Experimental.CallerClosure.target.definitions
  pure {
    setup := syntheticBinding (DefinitionId.of (owner.value ++ ".setup"))
      ("umpire-portable-setup/v1:" ++ experiment.plan.artifactChecksum.render)
    query := contract.query
    behavior := {
      definitionId := experiment.plan.behaviorDefinitionId
      behaviorFingerprint := experiment.plan.behaviorFingerprint
    }
    target := {
      definitionId := experiment.plan.targetDefinitionId
      behaviorFingerprint := experiment.plan.targetBehaviorFingerprint
    }
    kernel := {
      definitionId := experiment.plan.kernelDefinitionId
      behaviorFingerprint := experiment.plan.kernelBehaviorFingerprint
    }
    roleBindings := ← experiment.plan.bindings.mapM (portableRoleBinding owner source definitions)
    symbolicRoles := ← experiment.plan.symbolicRoles.mapM portableSymbolicRole
    runtimeBindingSlots := []
    preconditions := ← experiment.plan.modelPreconditions.mapM
      (portablePrecondition owner source definitions)
    initialState := ← portableModelValue owner source definitions experiment.plan.initialState
    requestedActions := ← experiment.plan.requestedActions.mapM
      (portableModelValue owner source definitions)
    modelOutcomes := ← experiment.plan.modelOutcomes.mapM
      (portableModelValue owner source definitions)
    resultingStates := ← experiment.plan.resultingStates.mapM
      (portableModelValue owner source definitions)
    occurrences := ← experiment.plan.linearExtension.mapM portableOccurrence
    selectedChoices := experiment.plan.selectedChoices.map fun value =>
      syntheticModelValue value .relation "selected-choice"
    selectedVariants := experiment.plan.selectedVariants.map fun value =>
      syntheticModelValue value .relation "selected-variant"
    requestedFaults := experiment.plan.requestedFaults.map fun value =>
      syntheticModelValue value .action "requested-fault"
    capabilityRequirements := ← experiment.plan.capabilityRequirementDefinitionIds.mapM
      (portableCapabilityBinding owner source contract.implementationLink)
    checkpoints := ← experiment.plan.checkpoints.mapM
      (portableCheckpoint owner source definitions)
    runtime := ← portableRuntimeProgram owner source contract.implementationLink
      runtimeConfiguration contract.observation
    artifactProjection := portableArtifactProjection experiment runtimeConfiguration
  }

private def portablePlanLimits
    (contract : Contract)
    (runtimeConfiguration : RuntimeConfiguration) : PortableTestPlanLimits :=
  let maximumPhaseRecords := runtimeConfiguration.phaseLimits.foldl
    (fun maximum limit => Nat.max maximum limit.maxRecords) 1
  {
  structural := {
    maxPlanBytes := 1024 * 1024
    maxNestingDepth := 256
    maxCollectionItems := 10000
    maxOperatorCount := 100000
  }
  execution := {
    maxActions := 1
    maxFaults := 1
    maxPhaseAttempts := runtimeConfiguration.phaseLimits.foldl
      (fun maximum limit => Nat.max maximum limit.maxAttempts) 1
    maxPhaseDurationMilliseconds := runtimeConfiguration.phaseLimits.foldl
      (fun maximum limit => Nat.max maximum limit.durationMilliseconds) 1
    maxTotalDurationMilliseconds := runtimeConfiguration.phaseLimits.foldl
      (fun total limit => total + limit.durationMilliseconds) 0
  }
  evidence := {
    maxRecords := Nat.max contract.limits.maxEvidenceRecords maximumPhaseRecords
    maxBytes := contract.limits.maxInputBytes
    maxSources := Nat.max contract.observation.profile.sources.length 1
  }
  evaluation := {
    maxExpressionDepth := contract.limits.maxExpressionDepth
    maxNatural := contract.limits.maxNatural
    maxWork := contract.limits.maxEvaluationWork
  }
  output := {
    maxDiagnosticBytes := contract.limits.maxDiagnosticBytes
    maxResultBytes := contract.limits.maxResultBytes
  }
}

private def compilerContract : DefinitionBinding :=
  syntheticBinding (DefinitionId.of "umpire.compiler.portable-test-plan.v1")
    "umpire-compiler-portable-test-plan/v1"

private def lowerPortableTestPlan
    (experiment : ExperimentSpec)
    (runtimeConfiguration : RuntimeConfiguration)
    (contract : Contract) : Except NonPortableError PortableTestPlan := do
  let execution ← portableExecutionProgram experiment runtimeConfiguration contract
  pure {
    planId := DefinitionId.of (experiment.plan.queryDefinitionId.value ++ ".portable-test-plan")
    modelCompiled := {
      test := contract.test
      query := contract.query
      experiment := experiment.artifactBinding
      runtimeConfig := runtimeConfiguration.artifactBinding
      properties := contract.properties.map Property.definition
      compilerContract
      sources := contract.provenance
    }
    execution
    verification := {
      evidence := contract.observation.profile
      observation := contract.observation
      traceProjection := .renameExactLink contract.implementationLink
      properties := contract.properties
    }
    limits := portablePlanLimits contract runtimeConfiguration
    knownGaps := contract.knownGaps
    externalObligations := []
  }

private def requiredObligationFrom
    (plan : PortableTestPlan)
    (index : Nat)
    (failure : NonPortableError) : ExternalVerificationObligation := {
  definition := syntheticBinding
    (DefinitionId.of (plan.planId.value ++ ".external-obligation." ++ toString index))
    ("umpire-portable-required-obligation/v1:" ++ failure.sourceDefinitionId.value ++ ":" ++
      failure.construct)
  kind := .required
  source := failure.source
  statement := "A separately trusted verifier must check " ++ failure.construct ++ "."
}

private def lowerPortableTestPlanWithObligations
    (experiment : ExperimentSpec)
    (runtimeConfiguration : RuntimeConfiguration)
    (contract : Contract)
    (checkedObservation : CheckedObservationPlan)
    (program : Temporal.System.Execution.Nexus.ObservationProgramDefinition) :
    Except NonPortableError PortableTestPlan := do
  let (observation, failures) ← lowerObservationProgramWithFailures checkedObservation program
    Temporal.System.Nexus.CallerClosure.target.definitions
  let plan ← lowerPortableTestPlan experiment runtimeConfiguration { contract with observation }
  let obligations := failures.zipIdx.map fun (failure, index) =>
    requiredObligationFrom plan (index + 1) failure
  pure { plan with externalObligations := obligations }

private def normalRuntimeConfiguration : RuntimeConfiguration :=
  Temporal.System.Execution.Nexus.runtimeConfigurationFor
    Temporal.Feature.Nexus.Experimental.CallerClosure.compiledArtifact

private def duplicateRuntimeConfiguration : RuntimeConfiguration :=
  Temporal.System.Execution.Nexus.duplicateDeliveryRuntimeConfigurationFor
    Temporal.Tool.RunEvaluation.Protocol.expectedDuplicateDeliveryExperiment

/-- The exact ordinary caller-closure checked Test lowered into one closed contract. -/
def normalContract : Except NonPortableError Contract :=
  lowerCheckedTest false
    Temporal.Feature.Nexus.Experimental.CallerClosure.compiledArtifact
    normalRuntimeConfiguration
    Temporal.System.Nexus.Observation.checkedPlan
    Temporal.System.Execution.Nexus.canonicalObservationProgramDefinition

/-- The exact duplicate-delivery negative control lowered into its distinct closed contract. -/
def duplicateContract : Except NonPortableError Contract :=
  lowerCheckedTest true
    Temporal.Tool.RunEvaluation.Protocol.expectedDuplicateDeliveryExperiment
    duplicateRuntimeConfiguration
    Temporal.System.Nexus.Observation.DuplicateDelivery.checkedPlan
    Temporal.System.Execution.Nexus.duplicateDeliveryObservationProgramDefinition

/-- The ordinary caller-closure checked Test compiled into the shared PortableTestPlan vocabulary. -/
def normalPortablePlan : Except NonPortableError PortableTestPlan := do
  let contract ← normalContract
  lowerPortableTestPlan
    Temporal.Feature.Nexus.Experimental.CallerClosure.compiledArtifact
    normalRuntimeConfiguration
    contract

/-- The duplicate-delivery negative control compiled into the same PortableTestPlan vocabulary. -/
def duplicatePortablePlan : Except NonPortableError PortableTestPlan := do
  let contract ← duplicateContract
  lowerPortableTestPlan
    Temporal.Tool.RunEvaluation.Protocol.expectedDuplicateDeliveryExperiment
    duplicateRuntimeConfiguration
    contract

private def requiredObligationMapping : ObservationMappingDeclaration := {
  Temporal.System.Nexus.Observation.mappingDeclaration with
  rules := Temporal.System.Nexus.Observation.mappingDeclaration.rules.map fun rule =>
    if rule.id == Temporal.System.Nexus.Observation.Mapping.stateRuleId then
      { rule with condition := some (.portable (.boolean false)) }
    else if rule.id == Temporal.System.Nexus.Observation.Mapping.actionRuleId then
      { rule with condition := some (.portable (.boolean true)) }
    else if rule.id == Temporal.System.Nexus.Observation.Mapping.outcomeRuleId then
      { rule with condition := some (.portable (.not (.boolean false))) }
    else rule
}

private def requiredObligationCheckedPlanResult : Except ObservationError CheckedObservationPlan :=
  checkObservation
    (ObservationCheckContext.ofTarget Temporal.System.Nexus.CallerClosure.target
      [Temporal.System.Nexus.Observation.Profile.declaration])
    requiredObligationMapping

private theorem requiredObligationCheckedPlanResult_isSome :
    requiredObligationCheckedPlanResult.toOption.isSome = true := by
  native_decide

private def requiredObligationCheckedPlan : CheckedObservationPlan :=
  requiredObligationCheckedPlanResult.toOption.get requiredObligationCheckedPlanResult_isSome

/-- A compiler fixture proving that an unsupported check remains a required external obligation. -/
def requiredObligationPortablePlan : Except NonPortableError PortableTestPlan := do
  let contract ← normalContract
  lowerPortableTestPlanWithObligations
    Temporal.Feature.Nexus.Experimental.CallerClosure.compiledArtifact
    normalRuntimeConfiguration
    contract
    requiredObligationCheckedPlan
    Temporal.System.Execution.Nexus.canonicalObservationProgramDefinition

/-- Canonical ProtoJSON bytes for the ordinary checked caller-closure Test. -/
def normalContractProtoJSON : Except NonPortableError String :=
  normalContract.map canonicalProtoJSON

/-- Canonical ProtoJSON bytes for the checked duplicate-delivery negative control. -/
def duplicateContractProtoJSON : Except NonPortableError String :=
  duplicateContract.map canonicalProtoJSON

/-- Canonical ProtoJSON for the ordinary model-compiled PortableTestPlan preimage. -/
def normalPortablePlanProtoJSON : Except NonPortableError String :=
  normalPortablePlan.map canonicalPortableTestPlanProtoJSON

/-- Canonical ProtoJSON for the duplicate-delivery model-compiled PortableTestPlan preimage. -/
def duplicatePortablePlanProtoJSON : Except NonPortableError String :=
  duplicatePortablePlan.map canonicalPortableTestPlanProtoJSON

/-- Canonical ProtoJSON retaining one unsupported check as a required obligation. -/
def requiredObligationPortablePlanProtoJSON : Except NonPortableError String :=
  requiredObligationPortablePlan.map canonicalPortableTestPlanProtoJSON

end Temporal.Tool.PortableEvaluationContract
