import Umpire.Artifact.Runtime
import Umpire.Json
import Umpire.Observation.Language

/-!
The portable Evaluation Contract is a closed data vocabulary shared with the protobuf boundary.
This module owns only inert data and canonical ProtoJSON; selecting and specializing checked model
values remains the responsibility of a Temporal-owned compiler.
-/

namespace Umpire.Artifact.PortableEvaluationContract

structure DefinitionBinding where
  definitionId : DefinitionId
  behaviorFingerprint : BehaviorFingerprint
  deriving BEq, DecidableEq, Repr

inductive PortableDefinitionKind where
  | setup
  | state
  | action
  | outcome
  | observation
  | relation
  | capability
  deriving BEq, DecidableEq, Repr

inductive PortableValue where
  | text (value : String)
  | natural (value : Nat)
  | boolean (value : Bool)
  deriving BEq, DecidableEq, Repr

structure ModelValue where
  definition : DefinitionBinding
  kind : PortableDefinitionKind
  value : PortableValue
  deriving BEq, DecidableEq, Repr

inductive TraceField where
  | initialState
  | priorState
  | selectedAction
  | modelOutcome
  | resultingState
  | observation
  deriving BEq, DecidableEq, Repr

structure ModelCoordinate where
  field : TraceField
  step : Nat := 0
  position : Nat := 0
  deriving BEq, DecidableEq, Repr

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

inductive DigestAlgorithm where
  | syntheticDigestV1
  deriving BEq, DecidableEq, Repr

structure DigestPolicy where
  definitionId : DefinitionId
  algorithm : DigestAlgorithm
  deriving BEq, DecidableEq, Repr

inductive FieldDisposition where
  | retain
  | redact
  | hash
  | reject
  deriving BEq, DecidableEq, Repr

structure EvidenceFieldDeclaration where
  fieldDefinitionId : DefinitionId
  valueType : ObservationValueType
  disposition : FieldDisposition
  digestPolicyDefinitionId : Option DefinitionId := none
  deriving BEq, DecidableEq, Repr

structure EvidenceKindDeclaration where
  kindDefinitionId : DefinitionId
  sourceDefinitionId : DefinitionId
  fields : List EvidenceFieldDeclaration
  deriving BEq, DecidableEq, Repr

structure EvidenceCardinality where
  kindDefinitionId : DefinitionId
  minimum : Nat
  maximum : Nat
  deriving BEq, DecidableEq, Repr

inductive CorrelationSlotKind where
  | run
  | workflow
  | operation
  deriving BEq, DecidableEq, Repr

structure CorrelationSlot where
  definitionId : DefinitionId
  kind : CorrelationSlotKind
  fields : List EvidenceFieldReference
  deriving BEq, DecidableEq, Repr

structure EvidenceProfile where
  definition : DefinitionBinding
  version : Nat
  sources : List DefinitionId
  kinds : List EvidenceKindDeclaration
  digestPolicies : List DigestPolicy
  cardinalities : List EvidenceCardinality
  correlationSlots : List CorrelationSlot
  deriving BEq, DecidableEq, Repr

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

structure Emit where
  definitionId : String
  sourceKindDefinitionId : DefinitionId
  outputDefinition : DefinitionBinding
  outputKind : PortableDefinitionKind
  coordinate : ModelCoordinate
  condition : ObservationExpression
  value : ObservationExpression
  deriving BEq, Repr

structure EmitOrdering where
  predecessorEmitDefinitionId : String
  successorEmitDefinitionId : String
  deriving BEq, DecidableEq, Repr

structure ObservationProgram where
  definition : DefinitionBinding
  source : SourceLocation
  mapping : DefinitionBinding
  mappingVersion : Nat
  profile : EvidenceProfile
  emits : List Emit
  ordering : List EmitOrdering
  deriving BEq, Repr

structure RenameExactEntry where
  source : ModelValue
  destination : ModelValue
  deriving BEq, DecidableEq, Repr

structure DefinitionRenameEntry where
  source : DefinitionBinding
  kind : PortableDefinitionKind
  destination : DefinitionBinding
  deriving BEq, DecidableEq, Repr

structure PortableLimit where
  value : Nat
  unit : String
  deriving BEq, DecidableEq, Repr

structure RenameExactLink where
  definition : DefinitionBinding
  source : SourceLocation
  sourceTarget : DefinitionBinding
  destinationTarget : DefinitionBinding
  entries : List RenameExactEntry
  definitionEntries : List DefinitionRenameEntry
  applicationLimit : PortableLimit
  deriving BEq, DecidableEq, Repr

inductive ClauseProvenance where
  | transitionContract
  | inputOutput
  deriving BEq, DecidableEq, Repr

inductive PatternOperator where
  | equalsText (value : String)
  | naturalAtMost (bound : Nat)
  deriving BEq, DecidableEq, Repr

structure Pattern where
  field : TraceField
  definition : DefinitionBinding
  operator : PatternOperator
  deriving BEq, DecidableEq, Repr

structure PropertyClause where
  definitionId : String
  provenance : ClauseProvenance
  trigger : Pattern
  required : Pattern
  deriving BEq, DecidableEq, Repr

structure Property where
  definition : DefinitionBinding
  source : SourceLocation
  requirements : List DefinitionBinding
  clauses : List PropertyClause
  deriving BEq, DecidableEq, Repr

inductive PortableKnownGapKind where
  | capabilityContract
  | input
  | interpretation
  | claim
  deriving BEq, DecidableEq, Repr

structure KnownGap where
  kind : PortableKnownGapKind
  code : String
  subject : String
  detail : String
  deriving BEq, DecidableEq, Repr

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

end Umpire.Artifact.PortableEvaluationContract
