import Umpire.Artifact.Runtime
import Umpire.Json
import Umpire.Observation.Declaration

/-!
The portable Evaluation Contract is a closed data vocabulary shared with the protobuf boundary.
This module owns only inert data and canonical ProtoJSON; selecting and specializing checked model
values remains the responsibility of a Temporal-owned compiler.
-/

namespace Umpire.Artifact.PortableEvaluationContract

/-- A stable definition identity paired with the checked behavior it denotes. -/
structure DefinitionBinding where
  definitionId : DefinitionId
  behaviorFingerprint : BehaviorFingerprint
  deriving BEq, DecidableEq, Repr

/-- The closed set of semantic definition roles carried by a portable value. -/
inductive PortableDefinitionKind where
  | setup
  | state
  | action
  | outcome
  | observation
  | relation
  | capability
  deriving BEq, DecidableEq, Repr

/-- A scalar model value that can cross the portable evaluation boundary. -/
inductive PortableValue where
  | text (value : String)
  | natural (value : Nat)
  | boolean (value : Bool)
  deriving BEq, DecidableEq, Repr

/-- One typed model value bound to its checked definition. -/
structure ModelValue where
  definition : DefinitionBinding
  kind : PortableDefinitionKind
  value : PortableValue
  deriving BEq, DecidableEq, Repr

/-- A model-trace field addressable by portable properties and observation output. -/
inductive TraceField where
  | initialState
  | priorState
  | selectedAction
  | modelOutcome
  | resultingState
  | observation
  deriving BEq, DecidableEq, Repr

/-- The exact field and position of one value in the model trace. -/
structure ModelCoordinate where
  field : TraceField
  step : Nat := 0
  position : Nat := 0
  deriving BEq, DecidableEq, Repr

/-- The resource ceilings an evaluator must enforce before and during evaluation. -/
structure EvaluationLimits where
  maxContractBytes : Nat
  maxInputBytes : Nat
  maxEvidenceRecords : Nat
  maxExpressionDepth : Nat
  maxCollectionItems : Nat
  maxNatural : Nat
  maxEvaluationWork : Nat
  maxDiagnosticBytes : Nat
  maxResultBytes : Nat
  maxTotalDurationMilliseconds : Nat
  maxOperatorCount : Nat
  deriving BEq, DecidableEq, Repr

/-- The closed set of digest algorithms portable evidence profiles may request. -/
inductive DigestAlgorithm where
  | syntheticDigestV1
  deriving BEq, DecidableEq, Repr

/-- A named digest algorithm available to evidence field declarations. -/
structure DigestPolicy where
  definitionId : DefinitionId
  algorithm : DigestAlgorithm
  deriving BEq, DecidableEq, Repr

/-- The required handling of one evidence field at the portable boundary. -/
inductive FieldDisposition where
  | retain
  | redact
  | hash
  | reject
  deriving BEq, DecidableEq, Repr

/-- The value type and handling policy for one field of an evidence kind. -/
structure EvidenceFieldDeclaration where
  fieldDefinitionId : DefinitionId
  valueType : ObservationValueType
  disposition : FieldDisposition
  digestPolicyDefinitionId : Option DefinitionId := none
  deriving BEq, DecidableEq, Repr

/-- The declared fields and source for one accepted evidence kind. -/
structure EvidenceKindDeclaration where
  kindDefinitionId : DefinitionId
  sourceDefinitionId : DefinitionId
  fields : List EvidenceFieldDeclaration
  deriving BEq, DecidableEq, Repr

/-- The accepted record-count range for one evidence kind. -/
structure EvidenceCardinality where
  kindDefinitionId : DefinitionId
  minimum : Nat
  maximum : Nat
  deriving BEq, DecidableEq, Repr

/-- The closed correlation scopes understood by portable observation programs. -/
inductive CorrelationSlotKind where
  | run
  | workflow
  | operation
  deriving BEq, DecidableEq, Repr

/-- The evidence fields that jointly identify one correlation scope. -/
structure CorrelationSlot where
  definitionId : DefinitionId
  kind : CorrelationSlotKind
  fields : List EvidenceFieldReference
  deriving BEq, DecidableEq, Repr

/-- The complete closed-world evidence schema consumed by an observation program. -/
structure EvidenceProfile where
  definition : DefinitionBinding
  version : Nat
  sources : List DefinitionId
  kinds : List EvidenceKindDeclaration
  digestPolicies : List DigestPolicy
  cardinalities : List EvidenceCardinality
  correlationSlots : List CorrelationSlot
  deriving BEq, DecidableEq, Repr

/-- A total expression over literals and declared evidence fields. -/
inductive ObservationExpression where
  | literalText (value : String)
  | literalNatural (value : Nat)
  | field (reference : EvidenceFieldReference)
  | naturalRenderV1 (operand : ObservationExpression)
  | present (operand : ObservationExpression)
  | equals (left right : ObservationExpression)
  | all (operands : List ObservationExpression)
  | any (operands : List ObservationExpression)
  deriving BEq

deriving instance Repr for ObservationExpression

/-- One conditional projection from evidence into a model-trace coordinate. -/
structure Emit where
  definitionId : String
  sourceKindDefinitionId : DefinitionId
  outputDefinition : DefinitionBinding
  outputKind : PortableDefinitionKind
  coordinate : ModelCoordinate
  condition : ObservationExpression
  value : ObservationExpression
  deriving BEq, Repr

/-- A required predecessor relationship between two emits. -/
structure EmitOrdering where
  predecessorEmitDefinitionId : String
  successorEmitDefinitionId : String
  deriving BEq, DecidableEq, Repr

/-- The evidence profile and ordered projections that construct a model trace. -/
structure ObservationProgram where
  definition : DefinitionBinding
  source : SourceLocation
  mapping : DefinitionBinding
  mappingVersion : Nat
  profile : EvidenceProfile
  emits : List Emit
  ordering : List EmitOrdering
  deriving BEq, Repr

/-- One exact source-to-destination model-value mapping. -/
structure RenameExactEntry where
  source : ModelValue
  destination : ModelValue
  deriving BEq, DecidableEq, Repr

/-- One exact source-to-destination definition mapping for a semantic role. -/
structure DefinitionRenameEntry where
  source : DefinitionBinding
  kind : PortableDefinitionKind
  destination : DefinitionBinding
  deriving BEq, DecidableEq, Repr

/-- A positive portable bound paired with its closed wire unit. -/
structure PortableLimit where
  value : Nat
  unit : String
  deriving BEq, DecidableEq, Repr

/-- A bounded, exact implementation link between two checked definition spaces. -/
structure RenameExactLink where
  definition : DefinitionBinding
  source : SourceLocation
  sourceTarget : DefinitionBinding
  destinationTarget : DefinitionBinding
  entries : List RenameExactEntry
  definitionEntries : List DefinitionRenameEntry
  applicationLimit : PortableLimit
  deriving BEq, DecidableEq, Repr

/-- The semantic source of a portable property clause. -/
inductive ClauseProvenance where
  | transitionContract
  | inputOutput
  deriving BEq, DecidableEq, Repr

/-- A closed predicate over one model-trace field. -/
inductive PatternOperator where
  | equalsText (value : String)
  | naturalAtMost (bound : Nat)
  deriving BEq, DecidableEq, Repr

/-- A typed predicate applied to one field of the model trace. -/
structure Pattern where
  field : TraceField
  definition : DefinitionBinding
  operator : PatternOperator
  deriving BEq, DecidableEq, Repr

/-- One per-step implication checked independently against the model trace. -/
structure PropertyClause where
  definitionId : String
  provenance : ClauseProvenance
  trigger : Pattern
  required : Pattern
  deriving BEq, DecidableEq, Repr

/-- A named collection of portable clauses and their semantic requirements. -/
structure Property where
  definition : DefinitionBinding
  source : SourceLocation
  requirements : List DefinitionBinding
  clauses : List PropertyClause
  deriving BEq, DecidableEq, Repr

/-- The closed categories of incompleteness an evaluator may report. -/
inductive PortableKnownGapKind where
  | capabilityContract
  | input
  | interpretation
  | claim
  deriving BEq, DecidableEq, Repr

/-- One explicit limitation carried with the portable evaluation contract. -/
structure KnownGap where
  kind : PortableKnownGapKind
  code : String
  subject : String
  detail : String
  deriving BEq, DecidableEq, Repr

/-- The self-contained contract consumed by a portable evaluator. -/
structure Contract where
  versionMajor : Nat := 1
  versionMinor : Nat := 0
  contractId : String
  artifactChecksum : Option String := none
  experiment : Umpire.ArtifactBinding
  runtimeConfig : Umpire.ArtifactBinding
  test : DefinitionBinding
  query : DefinitionBinding
  limits : EvaluationLimits
  observation : ObservationProgram
  implementationLink : RenameExactLink
  properties : List Property
  knownGaps : List KnownGap
  provenance : List SourceLocation
  deriving BEq, Repr

/-- The closed comparison operators available to portable setup preconditions. -/
inductive PreconditionOperator where
  | equals
  | notEquals
  deriving BEq, DecidableEq, Repr

/-- One literal or declared role used by a portable setup precondition. -/
inductive ExecutionOperand where
  | literal (value : ModelValue)
  | role (definition : DefinitionBinding)
  | runtimeBindingSlot (definition : DefinitionBinding)
  deriving BEq, Repr

/-- One typed equality or inequality required before portable execution. -/
structure ExecutionPrecondition where
  definition : DefinitionBinding
  operator : PreconditionOperator
  left : ExecutionOperand
  right : ExecutionOperand
  deriving BEq, Repr

/-- One checked role and the model value selected for it. -/
structure PortableRoleBinding where
  role : DefinitionBinding
  value : ModelValue
  deriving BEq, Repr

/-- One role intentionally left symbolic in the portable plan. -/
structure PortableSymbolicRole where
  definition : DefinitionBinding
  kind : PortableDefinitionKind
  deriving BEq, Repr

/-- One runtime-filled value slot declared by the model compiler. -/
structure RuntimeBindingSlot where
  definition : DefinitionBinding
  valueKind : ObservationValueType
  deriving BEq, Repr

/-- One exact occurrence in the selected linear extension. -/
structure PortablePlannedOccurrence where
  definition : DefinitionBinding
  actionDefinitionId : DefinitionId
  position : Nat
  authoredDefinitionId : DefinitionId
  deriving BEq, Repr

/-- Expected model observations at one selected transition. -/
structure PortableExecutionCheckpoint where
  transition : Nat
  observations : List ModelValue
  deriving BEq, Repr

/-- Runtime bounds for one of the five closed execution phases. -/
structure PortableExecutionPhaseLimit where
  phase : Umpire.ExecutionPhase
  durationMilliseconds : Nat
  maxAttempts : Nat
  maxRecords : Nat
  maxBytes : Nat
  deriving BEq, Repr

/-- One participant program and its exact protocol and capability bindings. -/
structure PortableParticipantBinding where
  participant : DefinitionBinding
  protocol : DefinitionBinding
  protocolVersion : Nat
  program : DefinitionBinding
  capabilities : List DefinitionBinding
  deriving BEq, Repr

/-- Runtime bindings for the portable evidence profile, Observation, and mapping. -/
structure PortableObservationConfig where
  profile : DefinitionBinding
  program : DefinitionBinding
  mapping : DefinitionBinding
  deriving BEq, Repr

/-- The fixed runtime program selected by one model-compiled plan. -/
structure PortableRuntimeProgram where
  authorityProfile : DefinitionBinding
  config : DefinitionBinding
  participantBindings : List PortableParticipantBinding
  observationConfig : PortableObservationConfig
  phaseLimits : List PortableExecutionPhaseLimit
  termination : DefinitionBinding
  cleanup : DefinitionBinding
  authorityRequiredCapabilities : List DefinitionBinding
  deriving BEq, Repr

/-- The closed reason the model selected the projected Drive Plan. -/
inductive PlanSelectionReason where
  | satisfyingWitness
  | violatingCounterexample
  | behaviorSelection
  deriving BEq, Repr

/-- Exact expanded planning limits needed to reconstruct the selected Drive Plan. -/
structure PlanSearchLimits where
  maxSemanticTransitions : Nat
  maxSelectedActions : Nat
  maxCandidateEvaluations : Nat
  deriving BEq, Repr

/-- Exact planning counts needed to reconstruct the selected Drive Plan. -/
structure PlanExploredCounts where
  setups : Nat
  traces : Nat
  transitions : Nat
  propertyEvaluations : Nat
  deriving BEq, Repr

/-- Exact source identity of one projected model artifact. -/
structure PlanArtifactProvenance where
  sourceDefinitionIds : List DefinitionId
  sourceLocations : List SourceLocation
  deriving BEq, Repr

/-- Typed identity-bearing fields retained solely to reconstruct the exact executable artifacts. -/
structure PlanArtifactProjection where
  expandedLimits : PlanSearchLimits
  selectionReason : PlanSelectionReason
  explored : PlanExploredCounts
  experimentKnownGaps : List KnownGap
  experimentProvenance : PlanArtifactProvenance
  runtimeKnownGaps : List KnownGap
  runtimeProvenance : PlanArtifactProvenance
  experimentObservationRequirementDefinitionIds : List DefinitionId
  runtimeObservationConfig : PortableObservationConfig
  deriving BEq, Repr

/-- The complete selected execution program, without environment coordinates or credentials. -/
structure PortableExecutionProgram where
  setup : DefinitionBinding
  query : DefinitionBinding
  behavior : DefinitionBinding
  target : DefinitionBinding
  kernel : DefinitionBinding
  roleBindings : List PortableRoleBinding
  symbolicRoles : List PortableSymbolicRole
  runtimeBindingSlots : List RuntimeBindingSlot
  preconditions : List ExecutionPrecondition
  initialState : ModelValue
  requestedActions : List ModelValue
  modelOutcomes : List ModelValue
  resultingStates : List ModelValue
  occurrences : List PortablePlannedOccurrence
  selectedChoices : List ModelValue
  selectedVariants : List ModelValue
  requestedFaults : List ModelValue
  capabilityRequirements : List DefinitionBinding
  checkpoints : List PortableExecutionCheckpoint
  runtime : PortableRuntimeProgram
  artifactProjection : PlanArtifactProjection
  deriving BEq, Repr

/-- The two explicit trace projections admitted by the portable evaluator. -/
inductive TraceProjection where
  | directPlanTrace
  | renameExactLink (link : RenameExactLink)
  deriving BEq, Repr

/-- The exact finite verification program bundled with one portable execution. -/
structure PortableVerificationProgram where
  evidence : EvidenceProfile
  observation : ObservationProgram
  traceProjection : TraceProjection
  properties : List Property
  deriving BEq, Repr

/-- Structural resource limits for decoding one portable plan. -/
structure StructuralLimits where
  maxPlanBytes : Nat
  maxNestingDepth : Nat
  maxCollectionItems : Nat
  maxOperatorCount : Nat
  deriving BEq, Repr

/-- Resource limits for one bounded execution. -/
structure ExecutionLimits where
  maxActions : Nat
  maxFaults : Nat
  maxPhaseAttempts : Nat
  maxPhaseDurationMilliseconds : Nat
  maxTotalDurationMilliseconds : Nat
  deriving BEq, Repr

/-- Resource limits for the collected Evidence. -/
structure EvidenceLimits where
  maxRecords : Nat
  maxBytes : Nat
  maxSources : Nat
  deriving BEq, Repr

/-- Resource limits for portable Observation and Property evaluation. -/
structure PortableEvaluationLimits where
  maxExpressionDepth : Nat
  maxNatural : Nat
  maxWork : Nat
  deriving BEq, Repr

/-- Resource limits for diagnostics and the complete typed result. -/
structure OutputLimits where
  maxDiagnosticBytes : Nat
  maxResultBytes : Nat
  deriving BEq, Repr

/-- Independent resource ceilings for every portable plan stage. -/
structure PortableTestPlanLimits where
  structural : StructuralLimits
  execution : ExecutionLimits
  evidence : EvidenceLimits
  evaluation : PortableEvaluationLimits
  output : OutputLimits
  deriving BEq, Repr

/-- Whether an external verifier is required for complete model-bound success. -/
inductive ExternalVerificationObligationKind where
  | required
  | advisory
  deriving BEq, DecidableEq, Repr

/-- One unsupported check retained for a separately trusted verifier. -/
structure ExternalVerificationObligation where
  definition : DefinitionBinding
  kind : ExternalVerificationObligationKind
  source : SourceLocation
  statement : String
  deriving BEq, Repr

/-- The exact independently verifiable identity of a model compiler output. -/
structure ModelCompiledPlanProvenance where
  test : DefinitionBinding
  query : DefinitionBinding
  experiment : Umpire.ArtifactBinding
  runtimeConfig : Umpire.ArtifactBinding
  properties : List DefinitionBinding
  compilerContract : DefinitionBinding
  sources : List SourceLocation
  deriving BEq, Repr

/-- One model-compiled instance of the caller-neutral PortableTestPlan protobuf. -/
structure PortableTestPlan where
  versionMajor : Nat := 1
  versionMinor : Nat := 0
  planId : DefinitionId
  modelCompiled : ModelCompiledPlanProvenance
  execution : PortableExecutionProgram
  verification : PortableVerificationProgram
  limits : PortableTestPlanLimits
  knownGaps : List KnownGap
  externalObligations : List ExternalVerificationObligation
  deriving BEq, Repr

private def quote (value : String) : String := Lean.Json.compress (.str value)

private def array (items : List String) : String :=
  "[" ++ String.intercalate "," items ++ "]"

private def optionalField (name : String) (items : List α) (render : List α → String) : String :=
  if items.isEmpty then "" else ",\"" ++ name ++ "\":" ++ render items

private def definitionBindingJson (binding : DefinitionBinding) : String :=
  "{\"definitionId\":" ++ quote binding.definitionId.value ++
    ",\"behaviorFingerprint\":" ++ quote binding.behaviorFingerprint.render ++ "}"

private def artifactBindingJson (binding : Umpire.ArtifactBinding) : String :=
  "{\"formatVersion\":" ++ quote binding.formatVersion ++
    ",\"artifactChecksum\":" ++ quote binding.artifactChecksum.render ++
    ",\"behaviorFingerprint\":" ++ quote binding.behaviorFingerprint.render ++
    ",\"provenanceChecksum\":" ++ quote binding.provenanceChecksum.render ++ "}"

private def sourceJson (source : SourceLocation) : String :=
  "{\"path\":" ++ quote source.path ++
    ",\"line\":" ++ quote (toString source.line) ++
    ",\"column\":" ++ quote (toString source.column) ++
    ",\"provenance\":" ++ quote source.provenance ++ "}"

private def PortableDefinitionKind.protoName : PortableDefinitionKind → String
  | .setup => "DEFINITION_KIND_SETUP"
  | .state => "DEFINITION_KIND_STATE"
  | .action => "DEFINITION_KIND_ACTION"
  | .outcome => "DEFINITION_KIND_OUTCOME"
  | .observation => "DEFINITION_KIND_OBSERVATION"
  | .relation => "DEFINITION_KIND_RELATION"
  | .capability => "DEFINITION_KIND_CAPABILITY"

private def TraceField.protoName : TraceField → String
  | .initialState => "TRACE_FIELD_INITIAL_STATE"
  | .priorState => "TRACE_FIELD_PRIOR_STATE"
  | .selectedAction => "TRACE_FIELD_SELECTED_ACTION"
  | .modelOutcome => "TRACE_FIELD_MODEL_OUTCOME"
  | .resultingState => "TRACE_FIELD_RESULTING_STATE"
  | .observation => "TRACE_FIELD_OBSERVATION"

private def valueJson : PortableValue → String
  | .text value => "{\"text\":" ++ quote value ++ "}"
  | .natural value => "{\"natural\":" ++ quote (toString value) ++ "}"
  | .boolean value => "{\"boolValue\":" ++ (if value then "true" else "false") ++ "}"

private def modelValueJson (value : ModelValue) : String :=
  "{\"definition\":" ++ definitionBindingJson value.definition ++
    ",\"kind\":" ++ quote value.kind.protoName ++
    ",\"value\":" ++ valueJson value.value ++ "}"

private def coordinateJson (coordinate : ModelCoordinate) : String :=
  "{\"field\":" ++ quote coordinate.field.protoName ++
    (if coordinate.step == 0 then "" else ",\"step\":" ++ quote (toString coordinate.step)) ++
    (if coordinate.position == 0 then "" else
      ",\"position\":" ++ quote (toString coordinate.position)) ++ "}"

private def limitsJson (limits : EvaluationLimits) : String :=
  "{\"maxContractBytes\":" ++ quote (toString limits.maxContractBytes) ++
    ",\"maxInputBytes\":" ++ quote (toString limits.maxInputBytes) ++
    ",\"maxEvidenceRecords\":" ++ quote (toString limits.maxEvidenceRecords) ++
    ",\"maxExpressionDepth\":" ++ quote (toString limits.maxExpressionDepth) ++
    ",\"maxCollectionItems\":" ++ quote (toString limits.maxCollectionItems) ++
    ",\"maxNatural\":" ++ quote (toString limits.maxNatural) ++
    ",\"maxEvaluationWork\":" ++ quote (toString limits.maxEvaluationWork) ++
    ",\"maxDiagnosticBytes\":" ++ quote (toString limits.maxDiagnosticBytes) ++
    ",\"maxResultBytes\":" ++ quote (toString limits.maxResultBytes) ++
    ",\"maxTotalDurationMilliseconds\":" ++
      quote (toString limits.maxTotalDurationMilliseconds) ++
    ",\"maxOperatorCount\":" ++ quote (toString limits.maxOperatorCount) ++ "}"

private def valueTypeName : ObservationValueType → String
  | .text => "VALUE_KIND_TEXT"
  | .natural => "VALUE_KIND_NATURAL"
  | .boolean => "VALUE_KIND_BOOLEAN"

private def FieldDisposition.protoName : FieldDisposition → String
  | .retain => "FIELD_DISPOSITION_KIND_RETAIN"
  | .redact => "FIELD_DISPOSITION_KIND_REDACT"
  | .hash => "FIELD_DISPOSITION_KIND_HASH"
  | .reject => "FIELD_DISPOSITION_KIND_REJECT"

private def fieldDeclarationJson (field : EvidenceFieldDeclaration) : String :=
  "{\"fieldDefinitionId\":" ++ quote field.fieldDefinitionId.value ++
    ",\"valueKind\":" ++ quote (valueTypeName field.valueType) ++
    ",\"disposition\":" ++ quote field.disposition.protoName ++
    (field.digestPolicyDefinitionId.map fun id =>
      ",\"digestPolicyDefinitionId\":" ++ quote id.value).getD "" ++ "}"

private def kindDeclarationJson (kind : EvidenceKindDeclaration) : String :=
  "{\"kindDefinitionId\":" ++ quote kind.kindDefinitionId.value ++
    ",\"sourceDefinitionId\":" ++ quote kind.sourceDefinitionId.value ++
    optionalField "fields" kind.fields (array ∘ List.map fieldDeclarationJson) ++ "}"

private def digestPolicyJson (policy : DigestPolicy) : String :=
  "{\"definitionId\":" ++ quote policy.definitionId.value ++
    ",\"algorithm\":\"DIGEST_ALGORITHM_SYNTHETIC_DIGEST_V1\"}"

private def cardinalityJson (cardinality : EvidenceCardinality) : String :=
  "{\"kindDefinitionId\":" ++ quote cardinality.kindDefinitionId.value ++
    (if cardinality.minimum == 0 then "" else
      ",\"minimum\":" ++ quote (toString cardinality.minimum)) ++
    ",\"maximum\":" ++ quote (toString cardinality.maximum) ++ "}"

private def fieldReferenceJson (reference : EvidenceFieldReference) : String :=
  "{\"kindDefinitionId\":" ++ quote reference.kind.value ++
    ",\"fieldDefinitionId\":" ++ quote reference.field.value ++ "}"

private def CorrelationSlotKind.protoName : CorrelationSlotKind → String
  | .run => "CORRELATION_SLOT_KIND_RUN"
  | .workflow => "CORRELATION_SLOT_KIND_WORKFLOW"
  | .operation => "CORRELATION_SLOT_KIND_OPERATION"

private def correlationSlotJson (slot : CorrelationSlot) : String :=
  "{\"definitionId\":" ++ quote slot.definitionId.value ++
    ",\"kind\":" ++ quote slot.kind.protoName ++
    optionalField "fields" slot.fields (array ∘ List.map fieldReferenceJson) ++ "}"

private def evidenceProfileJson (profile : EvidenceProfile) : String :=
  "{\"definition\":" ++ definitionBindingJson profile.definition ++
    ",\"version\":" ++ toString profile.version ++
    optionalField "sources" profile.sources (array ∘ List.map fun id =>
      "{\"sourceDefinitionId\":" ++ quote id.value ++ "}") ++
    optionalField "kinds" profile.kinds (array ∘ List.map kindDeclarationJson) ++
    optionalField "digestPolicies" profile.digestPolicies (array ∘ List.map digestPolicyJson) ++
    optionalField "cardinalities" profile.cardinalities (array ∘ List.map cardinalityJson) ++
    optionalField "correlationSlots" profile.correlationSlots
      (array ∘ List.map correlationSlotJson) ++ "}"

private partial def expressionJson : ObservationExpression → String
  | .literalText value => "{\"literalText\":{\"value\":" ++ quote value ++ "}}"
  | .literalNatural value =>
      "{\"literalNatural\":{\"value\":" ++ quote (toString value) ++ "}}"
  | .field reference => "{\"field\":" ++ fieldReferenceJson reference ++ "}"
  | .naturalRenderV1 operand =>
      "{\"naturalRenderV1\":{\"operand\":" ++ expressionJson operand ++ "}}"
  | .present operand => "{\"present\":{\"operand\":" ++ expressionJson operand ++ "}}"
  | .equals left right =>
      "{\"equals\":{\"left\":" ++ expressionJson left ++
        ",\"right\":" ++ expressionJson right ++ "}}"
  | .all operands => "{\"all\":{\"operands\":" ++ array (operands.map expressionJson) ++ "}}"
  | .any operands => "{\"any\":{\"operands\":" ++ array (operands.map expressionJson) ++ "}}"

private def emitJson (emit : Emit) : String :=
  "{\"definitionId\":" ++ quote emit.definitionId ++
    ",\"sourceKindDefinitionId\":" ++ quote emit.sourceKindDefinitionId.value ++
    ",\"outputDefinition\":" ++ definitionBindingJson emit.outputDefinition ++
    ",\"outputKind\":" ++ quote emit.outputKind.protoName ++
    ",\"coordinate\":" ++ coordinateJson emit.coordinate ++
    ",\"condition\":" ++ expressionJson emit.condition ++
    ",\"value\":" ++ expressionJson emit.value ++ "}"

private def orderingJson (ordering : EmitOrdering) : String :=
  "{\"predecessorEmitDefinitionId\":" ++ quote ordering.predecessorEmitDefinitionId ++
    ",\"successorEmitDefinitionId\":" ++ quote ordering.successorEmitDefinitionId ++ "}"

private def observationJson (observation : ObservationProgram) : String :=
  "{\"definition\":" ++ definitionBindingJson observation.definition ++
    ",\"source\":" ++ sourceJson observation.source ++
    ",\"mapping\":" ++ definitionBindingJson observation.mapping ++
    ",\"mappingVersion\":" ++ toString observation.mappingVersion ++
    ",\"profile\":" ++ evidenceProfileJson observation.profile ++
    optionalField "emits" observation.emits (array ∘ List.map emitJson) ++
    optionalField "ordering" observation.ordering (array ∘ List.map orderingJson) ++ "}"

private def renameEntryJson (entry : RenameExactEntry) : String :=
  "{\"source\":" ++ modelValueJson entry.source ++
    ",\"destination\":" ++ modelValueJson entry.destination ++ "}"

private def definitionRenameEntryJson (entry : DefinitionRenameEntry) : String :=
  "{\"source\":" ++ definitionBindingJson entry.source ++
    ",\"kind\":" ++ quote entry.kind.protoName ++
    ",\"destination\":" ++ definitionBindingJson entry.destination ++ "}"

private def implementationLinkJson (link : RenameExactLink) : String :=
  "{\"definition\":" ++ definitionBindingJson link.definition ++
    ",\"source\":" ++ sourceJson link.source ++
    ",\"sourceTarget\":" ++ definitionBindingJson link.sourceTarget ++
    ",\"destinationTarget\":" ++ definitionBindingJson link.destinationTarget ++
    optionalField "entries" link.entries (array ∘ List.map renameEntryJson) ++
    optionalField "definitionEntries" link.definitionEntries
      (array ∘ List.map definitionRenameEntryJson) ++
    ",\"applicationLimit\":{\"value\":" ++ quote (toString link.applicationLimit.value) ++
    ",\"unit\":" ++ quote link.applicationLimit.unit ++ "}}"

private def ClauseProvenance.protoName : ClauseProvenance → String
  | .transitionContract => "PROPERTY_CLAUSE_PROVENANCE_TRANSITION_CONTRACT"
  | .inputOutput => "PROPERTY_CLAUSE_PROVENANCE_INPUT_OUTPUT"

private def patternJson (pattern : Pattern) : String :=
  let operator := match pattern.operator with
    | .equalsText value => ",\"equalsText\":{\"value\":" ++ quote value ++ "}"
    | .naturalAtMost bound =>
        ",\"naturalAtMost\":{\"bound\":" ++ quote (toString bound) ++ "}"
  "{\"field\":" ++ quote pattern.field.protoName ++
    ",\"definition\":" ++ definitionBindingJson pattern.definition ++ operator ++ "}"

private def clauseJson (clause : PropertyClause) : String :=
  "{\"definitionId\":" ++ quote clause.definitionId ++
    ",\"provenance\":" ++ quote clause.provenance.protoName ++
    ",\"perStepImplies\":{\"trigger\":" ++ patternJson clause.trigger ++
    ",\"required\":" ++ patternJson clause.required ++ "}}"

private def propertyJson (property : Property) : String :=
  "{\"definition\":" ++ definitionBindingJson property.definition ++
    ",\"source\":" ++ sourceJson property.source ++
    optionalField "requirements" property.requirements (array ∘ List.map definitionBindingJson) ++
    optionalField "clauses" property.clauses (array ∘ List.map clauseJson) ++ "}"

private def PortableKnownGapKind.protoName : PortableKnownGapKind → String
  | .capabilityContract => "KNOWN_GAP_KIND_CAPABILITY_CONTRACT"
  | .input => "KNOWN_GAP_KIND_INPUT"
  | .interpretation => "KNOWN_GAP_KIND_INTERPRETATION"
  | .claim => "KNOWN_GAP_KIND_CLAIM"

private def knownGapJson (gap : KnownGap) : String :=
  "{\"kind\":" ++ quote gap.kind.protoName ++
    ",\"code\":" ++ quote gap.code ++
    (if gap.subject == "" then "" else ",\"subject\":" ++ quote gap.subject) ++
    (if gap.detail == "" then "" else ",\"detail\":" ++ quote gap.detail) ++ "}"

/-- Render exact canonical ProtoJSON accepted by the structural Go packer. -/
def canonicalProtoJSON (contract : Contract) : String :=
  Umpire.Json.prettyBytes <|
    "{\"version\":{\"major\":" ++ toString contract.versionMajor ++
      (if contract.versionMinor == 0 then "" else
        ",\"minor\":" ++ toString contract.versionMinor) ++ "}" ++
      ",\"contractId\":" ++ quote contract.contractId ++
      (contract.artifactChecksum.map fun checksum =>
        ",\"artifactChecksum\":" ++ quote checksum).getD "" ++
      ",\"experiment\":" ++ artifactBindingJson contract.experiment ++
      ",\"runtimeConfig\":" ++ artifactBindingJson contract.runtimeConfig ++
      ",\"test\":" ++ definitionBindingJson contract.test ++
      ",\"query\":" ++ definitionBindingJson contract.query ++
      ",\"limits\":" ++ limitsJson contract.limits ++
      ",\"observation\":" ++ observationJson contract.observation ++
      ",\"implementationLink\":" ++ implementationLinkJson contract.implementationLink ++
      optionalField "properties" contract.properties (array ∘ List.map propertyJson) ++
      optionalField "knownGaps" contract.knownGaps (array ∘ List.map knownGapJson) ++
      optionalField "provenance" contract.provenance (array ∘ List.map sourceJson) ++ "}"

private def PortableDefinitionKind.portableProtoName : PortableDefinitionKind → String
  | .setup => "PORTABLE_DEFINITION_KIND_SETUP"
  | .state => "PORTABLE_DEFINITION_KIND_STATE"
  | .action => "PORTABLE_DEFINITION_KIND_ACTION"
  | .outcome => "PORTABLE_DEFINITION_KIND_OUTCOME"
  | .observation => "PORTABLE_DEFINITION_KIND_OBSERVATION"
  | .relation => "PORTABLE_DEFINITION_KIND_RELATION"
  | .capability => "PORTABLE_DEFINITION_KIND_CAPABILITY"

private def portableModelValueJson (value : ModelValue) : String :=
  "{\"definition\":" ++ definitionBindingJson value.definition ++
    ",\"kind\":" ++ quote value.kind.portableProtoName ++
    ",\"value\":" ++ valueJson value.value ++ "}"

private def PreconditionOperator.protoName : PreconditionOperator → String
  | .equals => "PRECONDITION_OPERATOR_EQUALS"
  | .notEquals => "PRECONDITION_OPERATOR_NOT_EQUALS"

private def executionOperandJson : ExecutionOperand → String
  | .literal value => "{\"literal\":" ++ portableModelValueJson value ++ "}"
  | .role definition => "{\"role\":" ++ definitionBindingJson definition ++ "}"
  | .runtimeBindingSlot definition =>
      "{\"runtimeBindingSlot\":" ++ definitionBindingJson definition ++ "}"

private def executionPreconditionJson (precondition : ExecutionPrecondition) : String :=
  "{\"definition\":" ++ definitionBindingJson precondition.definition ++
    ",\"operator\":" ++ quote precondition.operator.protoName ++
    ",\"left\":" ++ executionOperandJson precondition.left ++
    ",\"right\":" ++ executionOperandJson precondition.right ++ "}"

private def portableRoleBindingJson (binding : PortableRoleBinding) : String :=
  "{\"role\":" ++ definitionBindingJson binding.role ++
    ",\"value\":" ++ portableModelValueJson binding.value ++ "}"

private def portableSymbolicRoleJson (role : PortableSymbolicRole) : String :=
  "{\"definition\":" ++ definitionBindingJson role.definition ++
    ",\"kind\":" ++ quote role.kind.portableProtoName ++ "}"

private def portableValueKindName : ObservationValueType → String
  | .text => "PORTABLE_VALUE_KIND_TEXT"
  | .natural => "PORTABLE_VALUE_KIND_NATURAL"
  | .boolean => "PORTABLE_VALUE_KIND_BOOLEAN"

private def runtimeBindingSlotJson (slot : RuntimeBindingSlot) : String :=
  "{\"definition\":" ++ definitionBindingJson slot.definition ++
    ",\"valueKind\":" ++ quote (portableValueKindName slot.valueKind) ++ "}"

private def plannedOccurrenceJson (occurrence : PortablePlannedOccurrence) : String :=
  "{\"definition\":" ++ definitionBindingJson occurrence.definition ++
    ",\"actionDefinitionId\":" ++ quote occurrence.actionDefinitionId.value ++
    ",\"position\":" ++ quote (toString occurrence.position) ++
    ",\"authoredDefinitionId\":" ++ quote occurrence.authoredDefinitionId.value ++ "}"

private def executionCheckpointJson (checkpoint : PortableExecutionCheckpoint) : String :=
  "{\"transition\":" ++ quote (toString checkpoint.transition) ++
    optionalField "observations" checkpoint.observations
      (array ∘ List.map portableModelValueJson) ++ "}"

private def executionPhaseProtoName : Umpire.ExecutionPhase → String
  | .preparation => "EXECUTION_PHASE_PREPARATION"
  | .realization => "EXECUTION_PHASE_REALIZATION"
  | .observation => "EXECUTION_PHASE_OBSERVATION"
  | .isolation => "EXECUTION_PHASE_ISOLATION"
  | .cleanup => "EXECUTION_PHASE_CLEANUP"

private def executionPhaseLimitJson (limit : PortableExecutionPhaseLimit) : String :=
  "{\"phase\":" ++ quote (executionPhaseProtoName limit.phase) ++
    ",\"durationMilliseconds\":" ++ quote (toString limit.durationMilliseconds) ++
    ",\"maxAttempts\":" ++ quote (toString limit.maxAttempts) ++
    ",\"maxRecords\":" ++ quote (toString limit.maxRecords) ++
    ",\"maxBytes\":" ++ quote (toString limit.maxBytes) ++ "}"

private def participantBindingJson (participant : PortableParticipantBinding) : String :=
  "{\"participant\":" ++ definitionBindingJson participant.participant ++
    ",\"protocol\":" ++ definitionBindingJson participant.protocol ++
    ",\"protocolVersion\":" ++ quote (toString participant.protocolVersion) ++
    ",\"program\":" ++ definitionBindingJson participant.program ++
    optionalField "capabilities" participant.capabilities
      (array ∘ List.map definitionBindingJson) ++ "}"

private def observationConfigJson (config : PortableObservationConfig) : String :=
  "{\"profile\":" ++ definitionBindingJson config.profile ++
    ",\"program\":" ++ definitionBindingJson config.program ++
    ",\"mapping\":" ++ definitionBindingJson config.mapping ++ "}"

private def runtimeProgramJson (runtime : PortableRuntimeProgram) : String :=
  "{\"authorityProfile\":" ++ definitionBindingJson runtime.authorityProfile ++
    ",\"config\":" ++ definitionBindingJson runtime.config ++
    optionalField "participantBindings" runtime.participantBindings
      (array ∘ List.map participantBindingJson) ++
    ",\"observationConfig\":" ++ observationConfigJson runtime.observationConfig ++
    optionalField "phaseLimits" runtime.phaseLimits (array ∘ List.map executionPhaseLimitJson) ++
    ",\"termination\":{\"definition\":" ++ definitionBindingJson runtime.termination ++ "}" ++
    ",\"cleanup\":{\"definition\":" ++ definitionBindingJson runtime.cleanup ++ "}" ++
    optionalField "authorityRequiredCapabilities" runtime.authorityRequiredCapabilities
      (array ∘ List.map definitionBindingJson) ++ "}"

private def PlanSelectionReason.protoName : PlanSelectionReason → String
  | .satisfyingWitness => "PLAN_SELECTION_REASON_SATISFYING_WITNESS"
  | .violatingCounterexample => "PLAN_SELECTION_REASON_VIOLATING_COUNTEREXAMPLE"
  | .behaviorSelection => "PLAN_SELECTION_REASON_BEHAVIOR_SELECTION"

private def artifactProvenanceJson (provenance : PlanArtifactProvenance) : String :=
  let ids := provenance.sourceDefinitionIds.map (quote ∘ DefinitionId.value)
  "{\"sourceDefinitionIds\":" ++ array ids ++
    ",\"sourceLocations\":" ++ array (provenance.sourceLocations.map sourceJson) ++ "}"

private def artifactProjectionJson (projection : PlanArtifactProjection) : String :=
  "{\"expandedLimits\":{\"maxSemanticTransitions\":" ++
      quote (toString projection.expandedLimits.maxSemanticTransitions) ++
    ",\"maxSelectedActions\":" ++ quote (toString projection.expandedLimits.maxSelectedActions) ++
    ",\"maxCandidateEvaluations\":" ++
      quote (toString projection.expandedLimits.maxCandidateEvaluations) ++ "}" ++
    ",\"selectionReason\":" ++ quote projection.selectionReason.protoName ++
    ",\"explored\":{\"setups\":" ++ quote (toString projection.explored.setups) ++
    ",\"traces\":" ++ quote (toString projection.explored.traces) ++
    ",\"transitions\":" ++ quote (toString projection.explored.transitions) ++
    ",\"propertyEvaluations\":" ++ quote (toString projection.explored.propertyEvaluations) ++ "}" ++
    optionalField "experimentKnownGaps" projection.experimentKnownGaps
      (array ∘ List.map knownGapJson) ++
    ",\"experimentProvenance\":" ++ artifactProvenanceJson projection.experimentProvenance ++
    optionalField "runtimeKnownGaps" projection.runtimeKnownGaps
      (array ∘ List.map knownGapJson) ++
    ",\"runtimeProvenance\":" ++ artifactProvenanceJson projection.runtimeProvenance ++
    optionalField "experimentObservationRequirementDefinitionIds"
      projection.experimentObservationRequirementDefinitionIds
      (array ∘ List.map (quote ∘ DefinitionId.value)) ++
    ",\"runtimeObservationConfig\":" ++ observationConfigJson projection.runtimeObservationConfig ++ "}"

private def executionProgramJson (execution : PortableExecutionProgram) : String :=
  "{\"setup\":" ++ definitionBindingJson execution.setup ++
    ",\"query\":" ++ definitionBindingJson execution.query ++
    ",\"behavior\":" ++ definitionBindingJson execution.behavior ++
    ",\"target\":" ++ definitionBindingJson execution.target ++
    ",\"kernel\":" ++ definitionBindingJson execution.kernel ++
    optionalField "roleBindings" execution.roleBindings (array ∘ List.map portableRoleBindingJson) ++
    optionalField "symbolicRoles" execution.symbolicRoles (array ∘ List.map portableSymbolicRoleJson) ++
    optionalField "runtimeBindingSlots" execution.runtimeBindingSlots
      (array ∘ List.map runtimeBindingSlotJson) ++
    optionalField "preconditions" execution.preconditions
      (array ∘ List.map executionPreconditionJson) ++
    ",\"initialState\":" ++ portableModelValueJson execution.initialState ++
    optionalField "requestedActions" execution.requestedActions
      (array ∘ List.map portableModelValueJson) ++
    optionalField "modelOutcomes" execution.modelOutcomes
      (array ∘ List.map portableModelValueJson) ++
    optionalField "resultingStates" execution.resultingStates
      (array ∘ List.map portableModelValueJson) ++
    optionalField "occurrences" execution.occurrences (array ∘ List.map plannedOccurrenceJson) ++
    optionalField "selectedChoices" execution.selectedChoices
      (array ∘ List.map portableModelValueJson) ++
    optionalField "selectedVariants" execution.selectedVariants
      (array ∘ List.map portableModelValueJson) ++
    optionalField "requestedFaults" execution.requestedFaults
      (array ∘ List.map portableModelValueJson) ++
    optionalField "capabilityRequirements" execution.capabilityRequirements
      (array ∘ List.map definitionBindingJson) ++
    optionalField "checkpoints" execution.checkpoints (array ∘ List.map executionCheckpointJson) ++
    ",\"runtime\":" ++ runtimeProgramJson execution.runtime ++
    ",\"artifactProjection\":" ++ artifactProjectionJson execution.artifactProjection ++ "}"

private def traceProjectionJson : TraceProjection → String
  | .directPlanTrace => "\"directPlanTrace\":{}"
  | .renameExactLink link => "\"renameExactLink\":" ++ implementationLinkJson link

private def verificationProgramJson (verification : PortableVerificationProgram) : String :=
  "{\"evidence\":" ++ evidenceProfileJson verification.evidence ++
    ",\"observation\":" ++ observationJson verification.observation ++
    "," ++ traceProjectionJson verification.traceProjection ++
    optionalField "properties" verification.properties (array ∘ List.map propertyJson) ++
    ",\"decision\":{\"kind\":\"DECISION_POLICY_KIND_STRICT_V1\"}}"

private def planLimitsJson (limits : PortableTestPlanLimits) : String :=
  "{\"structural\":{\"maxPlanBytes\":" ++ quote (toString limits.structural.maxPlanBytes) ++
    ",\"maxNestingDepth\":" ++ quote (toString limits.structural.maxNestingDepth) ++
    ",\"maxCollectionItems\":" ++ quote (toString limits.structural.maxCollectionItems) ++
    ",\"maxOperatorCount\":" ++ quote (toString limits.structural.maxOperatorCount) ++ "}" ++
    ",\"execution\":{\"maxActions\":" ++ quote (toString limits.execution.maxActions) ++
    ",\"maxFaults\":" ++ quote (toString limits.execution.maxFaults) ++
    ",\"maxPhaseAttempts\":" ++ quote (toString limits.execution.maxPhaseAttempts) ++
    ",\"maxPhaseDurationMilliseconds\":" ++
      quote (toString limits.execution.maxPhaseDurationMilliseconds) ++
    ",\"maxTotalDurationMilliseconds\":" ++
      quote (toString limits.execution.maxTotalDurationMilliseconds) ++ "}" ++
    ",\"evidence\":{\"maxRecords\":" ++ quote (toString limits.evidence.maxRecords) ++
    ",\"maxBytes\":" ++ quote (toString limits.evidence.maxBytes) ++
    ",\"maxSources\":" ++ quote (toString limits.evidence.maxSources) ++ "}" ++
    ",\"evaluation\":{\"maxExpressionDepth\":" ++
      quote (toString limits.evaluation.maxExpressionDepth) ++
    ",\"maxNatural\":" ++ quote (toString limits.evaluation.maxNatural) ++
    ",\"maxWork\":" ++ quote (toString limits.evaluation.maxWork) ++ "}" ++
    ",\"output\":{\"maxDiagnosticBytes\":" ++ quote (toString limits.output.maxDiagnosticBytes) ++
    ",\"maxResultBytes\":" ++ quote (toString limits.output.maxResultBytes) ++ "}}"

private def ExternalVerificationObligationKind.protoName :
    ExternalVerificationObligationKind → String
  | .required => "EXTERNAL_VERIFICATION_OBLIGATION_KIND_REQUIRED"
  | .advisory => "EXTERNAL_VERIFICATION_OBLIGATION_KIND_ADVISORY"

private def externalObligationJson (obligation : ExternalVerificationObligation) : String :=
  "{\"definition\":" ++ definitionBindingJson obligation.definition ++
    ",\"kind\":" ++ quote obligation.kind.protoName ++
    ",\"source\":" ++ sourceJson obligation.source ++
    ",\"statement\":" ++ quote obligation.statement ++ "}"

private def modelCompiledJson (model : ModelCompiledPlanProvenance) : String :=
  "{\"test\":" ++ definitionBindingJson model.test ++
    ",\"query\":" ++ definitionBindingJson model.query ++
    ",\"experiment\":" ++ artifactBindingJson model.experiment ++
    ",\"runtimeConfig\":" ++ artifactBindingJson model.runtimeConfig ++
    optionalField "properties" model.properties (array ∘ List.map definitionBindingJson) ++
    ",\"compilerContract\":" ++ definitionBindingJson model.compilerContract ++
    optionalField "sources" model.sources (array ∘ List.map sourceJson) ++ "}"

/-- Render deterministic ProtoJSON for the shared PortableTestPlan packer and checksum admission. -/
def canonicalPortableTestPlanProtoJSON (plan : PortableTestPlan) : String :=
  Umpire.Json.prettyBytes <|
    "{\"version\":{\"major\":" ++ toString plan.versionMajor ++
      (if plan.versionMinor == 0 then "" else ",\"minor\":" ++ toString plan.versionMinor) ++ "}" ++
      ",\"planId\":" ++ quote plan.planId.value ++
      ",\"modelCompiled\":" ++ modelCompiledJson plan.modelCompiled ++
      ",\"execution\":" ++ executionProgramJson plan.execution ++
      ",\"verification\":" ++ verificationProgramJson plan.verification ++
      ",\"limits\":" ++ planLimitsJson plan.limits ++
      optionalField "knownGaps" plan.knownGaps (array ∘ List.map knownGapJson) ++
      optionalField "externalObligations" plan.externalObligations
        (array ∘ List.map externalObligationJson) ++ "}"

end Umpire.Artifact.PortableEvaluationContract
