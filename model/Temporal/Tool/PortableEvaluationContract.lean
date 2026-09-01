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
  let stateRuleId := (plan.rules.find? fun rule =>
    rule.output == Temporal.System.Nexus.CallerClosure.stateId).map (·.id.value)
      |>.getD "missing-state-rule"
  let actionRuleId := (plan.rules.find? fun rule =>
    rule.output == Temporal.System.Nexus.CallerClosure.actionId).map (·.id.value)
      |>.getD "missing-action-rule"
  let ordering := ({
    predecessorEmitDefinitionId := stateRuleId ++ ".initial"
    successorEmitDefinitionId := actionRuleId
  } :: plan.ordering.map fun item => {
    predecessorEmitDefinitionId := if item.before.value == stateRuleId then
      stateRuleId ++ ".resulting" else item.before.value
    successorEmitDefinitionId := if item.after.value == stateRuleId then
      stateRuleId ++ ".resulting" else item.after.value
  }).mergeSort orderingLe
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
    let source ← modelValue declaration.id declaration.source
      link.sourceTarget.definitions .observation mapping.source
    let destination ← modelValue declaration.id declaration.source
      link.destinationTarget.definitions .observation mapping.destination
    pure ({ source := source, destination := destination } : RenameExactEntry)
  let duplicateEntries ← if duplicateDelivery then do
    let source ← modelValue declaration.id declaration.source link.sourceTarget.definitions
      .observation
      Temporal.System.Nexus.ImplementationLink.CallerClosure.DuplicateDelivery.sourceCancellationCountTwo
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

private def gapLe
    (left right : Umpire.Artifact.PortableEvaluationContract.KnownGap) : Bool :=
  let rank : PortableKnownGapKind → Nat
    | .capabilityContract => 1
    | .input => 2
    | .interpretation => 3
    | .claim => 4
  decide (rank left.kind < rank right.kind) ||
    (left.kind == right.kind && decide (left.code < right.code)) ||
    (left.kind == right.kind && left.code == right.code && decide (left.subject < right.subject)) ||
    (left.kind == right.kind && left.code == right.code && left.subject == right.subject &&
      decide (left.detail ≤ right.detail))

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
  let knownGaps := (experiment.plan.knownGaps ++ runtimeConfiguration.knownGaps)
    |>.map portableKnownGap |>.mergeSort gapLe |>.eraseDups
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
    observation := ← lowerObservationProgram plan program
      Temporal.System.Nexus.CallerClosure.target.definitions
    implementationLink := link
    properties := [property]
    knownGaps
    provenance := if provenance.isEmpty then [source] else provenance
  }

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

/-- Canonical ProtoJSON bytes for the ordinary checked caller-closure Test. -/
def normalContractProtoJSON : Except NonPortableError String :=
  normalContract.map canonicalProtoJSON

/-- Canonical ProtoJSON bytes for the checked duplicate-delivery negative control. -/
def duplicateContractProtoJSON : Except NonPortableError String :=
  duplicateContract.map canonicalProtoJSON

end Temporal.Tool.PortableEvaluationContract
