import Umpire.Artifact.Evidence
import Umpire.ImplementationLink.Application
import Umpire.Observation.Verdict

namespace Umpire

/-! Exact inert v2 transports for interpreted Evidence and one Run Evaluation Result. -/

/-- One Definition ID plus the checked behavior it denotes. -/
structure ArtifactDefinitionReference where
  definitionId : DefinitionId
  behaviorFingerprint : BehaviorFingerprint
  deriving BEq, DecidableEq, Repr

/-- Explicit transport projection of a Model coordinate. -/
structure ArtifactModelCoordinate where
  kind : String
  step : Option Nat
  position : Option Nat
  deriving BEq, DecidableEq, Ord, Repr

/-- One positive bound with its exact closed wire unit. -/
structure ArtifactLimit where
  value : Nat
  unit : String
  deriving BEq, DecidableEq, Ord, Repr

/-- One contiguous projected Model Trace step. -/
structure ArtifactModelTraceStep where
  position : Nat
  selectedAction : ModelValue
  modelOutcome : ModelValue
  resultingState : ModelValue
  observations : List ModelValue
  deriving BEq, DecidableEq, Repr

/-- The immutable Model Trace projection, without Evidence or evaluator constructors. -/
structure ArtifactModelTrace where
  traceId : String
  initialState : ModelValue
  steps : List ArtifactModelTraceStep
  deriving BEq, DecidableEq, Repr

/-- One projected vocabulary meaning. -/
structure ArtifactMeaningProvision where
  definitionId : DefinitionId
  kind : DefinitionKind
  canonicalBehavior : String
  deriving BEq, DecidableEq, Repr

/-- One exact raw-field identity. -/
structure ArtifactFieldReference where
  kindDefinitionId : DefinitionId
  fieldDefinitionId : DefinitionId
  deriving BEq, DecidableEq, Ord, Repr

/-- One persisted Observation disposition declaration. -/
structure ArtifactFieldDispositionRecord where
  field : ArtifactFieldReference
  disposition : String
  digestPolicyDefinitionId : Option DefinitionId
  deriving BEq, DecidableEq, Repr

/-- One persisted causal-order support row. -/
structure ArtifactEvidenceOrderingFact where
  factDefinitionId : DefinitionId
  kindDefinitionId : DefinitionId
  ordinal : Nat
  causalFactDefinitionIds : List DefinitionId
  deriving BEq, DecidableEq, Repr

/-- One persisted source-closure support row. -/
structure ArtifactEvidenceClosureFact where
  kindDefinitionId : DefinitionId
  lastOrdinal : Nat
  deriving BEq, DecidableEq, Repr

/-- The closed wire variants of AppliedDispositionEvidence. -/
structure ArtifactAppliedFieldDisposition where
  field : ArtifactFieldReference
  kind : String
  normalizedValue : Option String
  digestPolicyDefinitionId : Option DefinitionId
  digestToken : Option String
  deriving BEq, DecidableEq, Repr

/-- Why one already-established Model Fact is backed by exact Evidence. -/
structure ArtifactEvidenceLink where
  coordinate : ArtifactModelCoordinate
  mappingDefinitionId : DefinitionId
  mappingVersion : Nat
  mappingBehaviorFingerprint : BehaviorFingerprint
  profileDefinitionId : DefinitionId
  profileVersion : Nat
  evidenceDefinitionIds : List DefinitionId
  ruleDefinitionId : DefinitionId
  bindingDefinitionIds : List DefinitionId
  orderingSupport : List ArtifactEvidenceOrderingFact
  closureSupport : List ArtifactEvidenceClosureFact
  appliedDispositions : List ArtifactAppliedFieldDisposition
  appliedLimit : ArtifactLimit
  meaningBehaviorFingerprint : BehaviorFingerprint
  deriving BEq, DecidableEq, Repr

/-- Complete persisted projection of EvidenceBackedTrace without duplicate sibling links. -/
structure ArtifactEvidenceBackedModelTrace where
  traceId : String
  observationPlan : ArtifactDefinitionReference
  mappingDefinitionId : DefinitionId
  mappingVersion : Nat
  mappingBehaviorFingerprint : BehaviorFingerprint
  source : SourceLocation
  profileDefinitionId : DefinitionId
  profileVersion : Nat
  sourceClosed : Bool
  vocabulary : List ArtifactMeaningProvision
  appliedLimit : ArtifactLimit
  evidenceDefinitionIds : List DefinitionId
  trace : ArtifactModelTrace
  deriving BEq, DecidableEq, Repr

/-- Closed projection of one Observation diagnostic. -/
structure ArtifactObservationDiagnostic where
  kind : String
  observationPlanDefinitionId : DefinitionId
  relatedDefinitionIds : List DefinitionId
  appliedLimit : Option ArtifactLimit
  observedCount : Option Nat
  alternatives : List DefinitionId
  missingDiscriminatorDefinitionId : Option DefinitionId
  deriving BEq, DecidableEq, Repr

/-- Persisted output of Observation Evaluation; it never maps RawEvidence itself. -/
structure EvidenceArtifact where
  formatVersion : String
  runIdentity : DefinitionId
  behaviorFingerprint : BehaviorFingerprint
  experiment : ArtifactBinding
  runtimeConfiguration : ArtifactBinding
  run : ArtifactBinding
  rawEvidence : ArtifactBinding
  observationProgram : ArtifactDefinitionReference
  mapping : ArtifactDefinitionReference
  observationEvaluationStatus : String
  evidenceBackedModelTrace : Option ArtifactEvidenceBackedModelTrace
  evidenceLinks : List ArtifactEvidenceLink
  dispositions : List ArtifactFieldDispositionRecord
  diagnostics : List ArtifactObservationDiagnostic
  knownGaps : List KnownGap
  provenance : ArtifactProvenance
  provenanceChecksum : ArtifactChecksum
  artifactChecksum : ArtifactChecksum
  deriving BEq, DecidableEq, Repr

/-- Exact projected checked-Target identity. -/
structure ArtifactImplementationTargetReference where
  definitionId : DefinitionId
  kind : DefinitionKind
  behaviorFingerprint : BehaviorFingerprint
  deriving BEq, DecidableEq, Repr

/-- Closed projection of one Implementation Link diagnostic. -/
structure ArtifactImplementationLinkDiagnostic where
  kind : String
  coordinate : Option ArtifactModelCoordinate
  relatedDefinitionIds : List DefinitionId
  sourceSetupBehaviorFingerprint : Option BehaviorFingerprint
  appliedLimit : Option ArtifactLimit
  observedCount : Option Nat
  knownGapCode : Option DefinitionId
  knownGapReason : Option String
  unsupportedVocabularyKind : Option DefinitionKind
  evidenceLinkBehaviorFingerprint : Option BehaviorFingerprint
  identity : BehaviorFingerprint
  deriving BEq, DecidableEq, Repr

/-- One full Implementation Link identity plus its already-produced diagnostic, if any. -/
structure ArtifactImplementationLinkRecord where
  definitionId : DefinitionId
  behaviorFingerprint : BehaviorFingerprint
  sourceTarget : ArtifactImplementationTargetReference
  destinationTarget : ArtifactImplementationTargetReference
  diagnostic : Option ArtifactImplementationLinkDiagnostic
  deriving BEq, DecidableEq, Repr

/-- Closed projection of one semantic verdict diagnostic. -/
structure ArtifactSemanticVerdictDiagnostic where
  kind : String
  relatedDefinitionIds : List DefinitionId
  observationDiagnostic : Option ArtifactObservationDiagnostic
  deriving BEq, DecidableEq, Repr

/-- One already-evaluated Property clause with exact coordinate Evidence Links. -/
structure ArtifactSemanticClauseVerdict where
  propertyDefinitionId : DefinitionId
  clauseDefinitionId : DefinitionId
  status : String
  coordinates : List ArtifactModelCoordinate
  queryLimits : QueryLimits
  propertyLimit : Option ArtifactLimit
  evidenceLimit : ArtifactLimit
  provenanceDefinitionIds : List DefinitionId
  evidenceLinks : List ArtifactEvidenceLink
  deriving BEq, DecidableEq, Repr

/-- One already-produced semantic Property verdict. -/
structure ArtifactPropertyVerdict where
  queryDefinitionId : DefinitionId
  propertyDefinitionId : DefinitionId
  propertyBehaviorFingerprint : BehaviorFingerprint
  traceId : Option String
  status : String
  queryLimits : QueryLimits
  evidenceLimit : Option ArtifactLimit
  provenanceDefinitionIds : List DefinitionId
  clauses : List ArtifactSemanticClauseVerdict
  diagnostic : Option ArtifactSemanticVerdictDiagnostic
  deriving BEq, DecidableEq, Repr

/-- Exact strict Query aggregation, including byte-identical embedded verdicts. -/
structure ArtifactQuerySummary where
  queryDefinitionId : DefinitionId
  status : String
  queryLimits : QueryLimits
  requiredPropertyDefinitionIds : List DefinitionId
  propertyVerdicts : List ArtifactPropertyVerdict
  missingPropertyDefinitionIds : List DefinitionId
  duplicatePropertyDefinitionIds : List DefinitionId
  unexpectedPropertyDefinitionIds : List DefinitionId
  divergentPropertyDefinitionIds : List DefinitionId
  wrongQueryResultDefinitionIds : List DefinitionId
  traceIds : List String
  deriving BEq, DecidableEq, Repr

/-- One Limit retained at its exact closed evaluation stage. -/
structure ArtifactStagedLimit where
  stage : String
  limit : ArtifactLimit
  deriving BEq, DecidableEq, Repr

/-- Inert Run Evaluation result; it performs neither evaluation nor Claim Assessment. -/
structure ResultArtifact where
  formatVersion : String
  runIdentity : DefinitionId
  behaviorFingerprint : BehaviorFingerprint
  experiment : ArtifactBinding
  runtimeConfiguration : ArtifactBinding
  run : ArtifactBinding
  rawEvidence : ArtifactBinding
  evidence : ArtifactBinding
  operationalStatus : String
  observationEvaluationStatus : String
  implementationLink : ArtifactImplementationLinkRecord
  implementationLinkStatus : String
  propertyVerdicts : List ArtifactPropertyVerdict
  querySummary : ArtifactQuerySummary
  semanticStatus : String
  limits : List ArtifactStagedLimit
  knownGaps : List KnownGap
  cleanupStatus : String
  evaluationOutcomeChecksum : Option ArtifactChecksum
  provenance : ArtifactProvenance
  provenanceChecksum : ArtifactChecksum
  artifactChecksum : ArtifactChecksum
  deriving BEq, DecidableEq, Repr

private def quoteResult (value : String) : String := Lean.Json.compress (.str value)

private def resultArray (items : List String) : String :=
  "[" ++ String.intercalate "," items ++ "]"

private def optionalResultJson (value : Option String) : String := value.getD "null"

private def optionalStringJson (value : Option String) : String :=
  optionalResultJson (value.map quoteResult)

private def optionalIdResultJson (value : Option DefinitionId) : String :=
  optionalResultJson (value.map (quoteResult ∘ DefinitionId.value))

private def optionalFingerprintResultJson (value : Option BehaviorFingerprint) : String :=
  optionalResultJson (value.map (quoteResult ∘ BehaviorFingerprint.render))

private def optionalChecksumResultJson (value : Option ArtifactChecksum) : String :=
  optionalResultJson (value.map (quoteResult ∘ ArtifactChecksum.render))

private def optionalNatResultJson (value : Option Nat) : String :=
  optionalResultJson (value.map toString)

private def artifactLimitJson (limit : ArtifactLimit) : String :=
  "{\"value\":" ++ toString limit.value ++ ",\"unit\":" ++ quoteResult limit.unit ++ "}"

private def optionalArtifactLimitJson (limit : Option ArtifactLimit) : String :=
  optionalResultJson (limit.map artifactLimitJson)

private def coreLimitJson (limit : Limit) : String :=
  "{\"value\":" ++ toString limit.value ++ ",\"unit\":" ++ quoteResult limit.unit.name ++ "}"

private def queryLimitsJson (limits : QueryLimits) : String :=
  "{\"behavior\":{\"transitions\":" ++ coreLimitJson limits.behavior.transitions ++
    ",\"selectedActions\":" ++ coreLimitJson limits.behavior.selectedActions ++ "}" ++
    ",\"search\":" ++ coreLimitJson limits.search ++ "}"

private def definitionReferenceJson (reference : ArtifactDefinitionReference) : String :=
  "{\"definitionId\":" ++ quoteResult reference.definitionId.value ++
    ",\"behaviorFingerprint\":" ++ quoteResult reference.behaviorFingerprint.render ++ "}"

private def modelCoordinateJson (coordinate : ArtifactModelCoordinate) : String :=
  "{\"kind\":" ++ quoteResult coordinate.kind ++
    ",\"step\":" ++ optionalNatResultJson coordinate.step ++
    ",\"position\":" ++ optionalNatResultJson coordinate.position ++ "}"

private def modelValueResultJson (value : ModelValue) : String :=
  "{\"definitionId\":" ++ quoteResult value.definitionId.value ++
    ",\"value\":" ++ quoteResult value.value ++ "}"

private def modelTraceStepJson (step : ArtifactModelTraceStep) : String :=
  "{\"position\":" ++ toString step.position ++
    ",\"selectedAction\":" ++ modelValueResultJson step.selectedAction ++
    ",\"modelOutcome\":" ++ modelValueResultJson step.modelOutcome ++
    ",\"resultingState\":" ++ modelValueResultJson step.resultingState ++
    ",\"observations\":" ++ resultArray (step.observations.map modelValueResultJson) ++ "}"

private def modelTraceJson (trace : ArtifactModelTrace) : String :=
  "{\"traceId\":" ++ quoteResult trace.traceId ++
    ",\"initialState\":" ++ modelValueResultJson trace.initialState ++
    ",\"steps\":" ++ resultArray (trace.steps.map modelTraceStepJson) ++ "}"

private def sourceLocationResultJson (source : SourceLocation) : String :=
  "{\"path\":" ++ quoteResult source.path ++
    ",\"line\":" ++ toString source.line ++
    ",\"column\":" ++ toString source.column ++
    ",\"provenance\":" ++ quoteResult source.provenance ++ "}"

private def meaningProvisionJson (meaning : ArtifactMeaningProvision) : String :=
  "{\"definitionId\":" ++ quoteResult meaning.definitionId.value ++
    ",\"kind\":" ++ quoteResult meaning.kind.name ++
    ",\"canonicalBehavior\":" ++ quoteResult meaning.canonicalBehavior ++ "}"

private def fieldReferenceJson (field : ArtifactFieldReference) : String :=
  "{\"kindDefinitionId\":" ++ quoteResult field.kindDefinitionId.value ++
    ",\"fieldDefinitionId\":" ++ quoteResult field.fieldDefinitionId.value ++ "}"

private def fieldDispositionRecordJson (record : ArtifactFieldDispositionRecord) : String :=
  "{\"field\":" ++ fieldReferenceJson record.field ++
    ",\"disposition\":" ++ quoteResult record.disposition ++
    ",\"digestPolicyDefinitionId\":" ++
      optionalIdResultJson record.digestPolicyDefinitionId ++ "}"

private def evidenceOrderingFactJson (fact : ArtifactEvidenceOrderingFact) : String :=
  "{\"factDefinitionId\":" ++ quoteResult fact.factDefinitionId.value ++
    ",\"kindDefinitionId\":" ++ quoteResult fact.kindDefinitionId.value ++
    ",\"ordinal\":" ++ toString fact.ordinal ++
    ",\"causalFactDefinitionIds\":" ++ resultArray
      (fact.causalFactDefinitionIds.map (quoteResult ∘ DefinitionId.value)) ++ "}"

private def evidenceClosureFactJson (fact : ArtifactEvidenceClosureFact) : String :=
  "{\"kindDefinitionId\":" ++ quoteResult fact.kindDefinitionId.value ++
    ",\"lastOrdinal\":" ++ toString fact.lastOrdinal ++ "}"

private def appliedFieldDispositionJson (disposition : ArtifactAppliedFieldDisposition) : String :=
  "{\"field\":" ++ fieldReferenceJson disposition.field ++
    ",\"kind\":" ++ quoteResult disposition.kind ++
    ",\"normalizedValue\":" ++ optionalStringJson disposition.normalizedValue ++
    ",\"digestPolicyDefinitionId\":" ++
      optionalIdResultJson disposition.digestPolicyDefinitionId ++
    ",\"digestToken\":" ++ optionalStringJson disposition.digestToken ++ "}"

private def evidenceLinkJson (link : ArtifactEvidenceLink) : String :=
  "{\"coordinate\":" ++ modelCoordinateJson link.coordinate ++
    ",\"mappingDefinitionId\":" ++ quoteResult link.mappingDefinitionId.value ++
    ",\"mappingVersion\":" ++ toString link.mappingVersion ++
    ",\"mappingBehaviorFingerprint\":" ++ quoteResult link.mappingBehaviorFingerprint.render ++
    ",\"profileDefinitionId\":" ++ quoteResult link.profileDefinitionId.value ++
    ",\"profileVersion\":" ++ toString link.profileVersion ++
    ",\"evidenceDefinitionIds\":" ++ resultArray
      (link.evidenceDefinitionIds.map (quoteResult ∘ DefinitionId.value)) ++
    ",\"ruleDefinitionId\":" ++ quoteResult link.ruleDefinitionId.value ++
    ",\"bindingDefinitionIds\":" ++ resultArray
      (link.bindingDefinitionIds.map (quoteResult ∘ DefinitionId.value)) ++
    ",\"orderingSupport\":" ++ resultArray (link.orderingSupport.map evidenceOrderingFactJson) ++
    ",\"closureSupport\":" ++ resultArray (link.closureSupport.map evidenceClosureFactJson) ++
    ",\"appliedDispositions\":" ++ resultArray
      (link.appliedDispositions.map appliedFieldDispositionJson) ++
    ",\"appliedLimit\":" ++ artifactLimitJson link.appliedLimit ++
    ",\"meaningBehaviorFingerprint\":" ++ quoteResult link.meaningBehaviorFingerprint.render ++ "}"

private def evidenceBackedModelTraceJson (trace : ArtifactEvidenceBackedModelTrace) : String :=
  "{\"traceId\":" ++ quoteResult trace.traceId ++
    ",\"observationPlan\":" ++ definitionReferenceJson trace.observationPlan ++
    ",\"mappingDefinitionId\":" ++ quoteResult trace.mappingDefinitionId.value ++
    ",\"mappingVersion\":" ++ toString trace.mappingVersion ++
    ",\"mappingBehaviorFingerprint\":" ++ quoteResult trace.mappingBehaviorFingerprint.render ++
    ",\"source\":" ++ sourceLocationResultJson trace.source ++
    ",\"profileDefinitionId\":" ++ quoteResult trace.profileDefinitionId.value ++
    ",\"profileVersion\":" ++ toString trace.profileVersion ++
    ",\"sourceClosed\":" ++ (if trace.sourceClosed then "true" else "false") ++
    ",\"vocabulary\":" ++ resultArray (trace.vocabulary.map meaningProvisionJson) ++
    ",\"appliedLimit\":" ++ artifactLimitJson trace.appliedLimit ++
    ",\"evidenceDefinitionIds\":" ++ resultArray
      (trace.evidenceDefinitionIds.map (quoteResult ∘ DefinitionId.value)) ++
    ",\"trace\":" ++ modelTraceJson trace.trace ++ "}"

private def optionalEvidenceBackedModelTraceJson
    (trace : Option ArtifactEvidenceBackedModelTrace) : String :=
  optionalResultJson (trace.map evidenceBackedModelTraceJson)

private def observationDiagnosticJson (diagnostic : ArtifactObservationDiagnostic) : String :=
  "{\"kind\":" ++ quoteResult diagnostic.kind ++
    ",\"observationPlanDefinitionId\":" ++
      quoteResult diagnostic.observationPlanDefinitionId.value ++
    ",\"relatedDefinitionIds\":" ++ resultArray
      (diagnostic.relatedDefinitionIds.map (quoteResult ∘ DefinitionId.value)) ++
    ",\"appliedLimit\":" ++ optionalArtifactLimitJson diagnostic.appliedLimit ++
    ",\"observedCount\":" ++ optionalNatResultJson diagnostic.observedCount ++
    ",\"alternatives\":" ++ resultArray
      (diagnostic.alternatives.map (quoteResult ∘ DefinitionId.value)) ++
    ",\"missingDiscriminatorDefinitionId\":" ++
      optionalIdResultJson diagnostic.missingDiscriminatorDefinitionId ++ "}"

private def evidenceArtifactContentJson (evidence : EvidenceArtifact) : String :=
  "{\"formatVersion\":" ++ quoteResult evidence.formatVersion ++
    ",\"runIdentity\":" ++ quoteResult evidence.runIdentity.value ++
    ",\"behaviorFingerprint\":" ++ quoteResult evidence.behaviorFingerprint.render ++
    ",\"experiment\":" ++ evidence.experiment.canonicalJson ++
    ",\"runtimeConfiguration\":" ++ evidence.runtimeConfiguration.canonicalJson ++
    ",\"run\":" ++ evidence.run.canonicalJson ++
    ",\"rawEvidence\":" ++ evidence.rawEvidence.canonicalJson ++
    ",\"observationProgram\":" ++ definitionReferenceJson evidence.observationProgram ++
    ",\"mapping\":" ++ definitionReferenceJson evidence.mapping ++
    ",\"observationEvaluationStatus\":" ++ quoteResult evidence.observationEvaluationStatus ++
    ",\"evidenceBackedModelTrace\":" ++
      optionalEvidenceBackedModelTraceJson evidence.evidenceBackedModelTrace ++
    ",\"evidenceLinks\":" ++ resultArray (evidence.evidenceLinks.map evidenceLinkJson) ++
    ",\"dispositions\":" ++ resultArray
      (evidence.dispositions.map fieldDispositionRecordJson) ++
    ",\"diagnostics\":" ++ resultArray (evidence.diagnostics.map observationDiagnosticJson) ++
    ",\"knownGaps\":" ++ resultArray (evidence.knownGaps.map canonicalKnownGapJson) ++
    ",\"provenance\":" ++ evidence.provenance.canonicalJson ++
    ",\"provenanceChecksum\":" ++ quoteResult evidence.provenanceChecksum.render ++ "}"

def EvidenceArtifact.expectedArtifactChecksum (evidence : EvidenceArtifact) : ArtifactChecksum :=
  evidenceChecksumOf (Json.prettyBytes (evidenceArtifactContentJson evidence))

def EvidenceArtifact.seal (evidence : EvidenceArtifact) : EvidenceArtifact :=
  let withProvenance := { evidence with provenanceChecksum := evidence.provenance.expectedChecksum }
  { withProvenance with artifactChecksum := withProvenance.expectedArtifactChecksum }

def EvidenceArtifact.hasValidChecksums (evidence : EvidenceArtifact) : Bool :=
  evidence.provenanceChecksum == evidence.provenance.expectedChecksum &&
    evidence.artifactChecksum == evidence.expectedArtifactChecksum

def canonicalEvidenceArtifactJson (evidence : EvidenceArtifact) : String :=
  let content := evidenceArtifactContentJson evidence
  Json.pretty ((content.dropEnd 1).toString ++
    ",\"artifactChecksum\":" ++ quoteResult evidence.artifactChecksum.render ++ "}")

def canonicalEvidenceArtifactBytes (evidence : EvidenceArtifact) : String :=
  canonicalEvidenceArtifactJson evidence ++ "\n"

private def implementationTargetReferenceJson
    (target : ArtifactImplementationTargetReference) : String :=
  "{\"definitionId\":" ++ quoteResult target.definitionId.value ++
    ",\"kind\":" ++ quoteResult target.kind.name ++
    ",\"behaviorFingerprint\":" ++ quoteResult target.behaviorFingerprint.render ++ "}"

private def optionalModelCoordinateJson (coordinate : Option ArtifactModelCoordinate) : String :=
  optionalResultJson (coordinate.map modelCoordinateJson)

private def implementationLinkDiagnosticJson
    (diagnostic : ArtifactImplementationLinkDiagnostic) : String :=
  "{\"kind\":" ++ quoteResult diagnostic.kind ++
    ",\"coordinate\":" ++ optionalModelCoordinateJson diagnostic.coordinate ++
    ",\"relatedDefinitionIds\":" ++ resultArray
      (diagnostic.relatedDefinitionIds.map (quoteResult ∘ DefinitionId.value)) ++
    ",\"sourceSetupBehaviorFingerprint\":" ++
      optionalFingerprintResultJson diagnostic.sourceSetupBehaviorFingerprint ++
    ",\"appliedLimit\":" ++ optionalArtifactLimitJson diagnostic.appliedLimit ++
    ",\"observedCount\":" ++ optionalNatResultJson diagnostic.observedCount ++
    ",\"knownGapCode\":" ++ optionalIdResultJson diagnostic.knownGapCode ++
    ",\"knownGapReason\":" ++ optionalStringJson diagnostic.knownGapReason ++
    ",\"unsupportedVocabularyKind\":" ++ optionalStringJson
      (diagnostic.unsupportedVocabularyKind.map DefinitionKind.name) ++
    ",\"evidenceLinkBehaviorFingerprint\":" ++
      optionalFingerprintResultJson diagnostic.evidenceLinkBehaviorFingerprint ++
    ",\"identity\":" ++ quoteResult diagnostic.identity.render ++ "}"

private def optionalImplementationLinkDiagnosticJson
    (diagnostic : Option ArtifactImplementationLinkDiagnostic) : String :=
  optionalResultJson (diagnostic.map implementationLinkDiagnosticJson)

private def implementationLinkRecordJson (record : ArtifactImplementationLinkRecord) : String :=
  "{\"definitionId\":" ++ quoteResult record.definitionId.value ++
    ",\"behaviorFingerprint\":" ++ quoteResult record.behaviorFingerprint.render ++
    ",\"sourceTarget\":" ++ implementationTargetReferenceJson record.sourceTarget ++
    ",\"destinationTarget\":" ++ implementationTargetReferenceJson record.destinationTarget ++
    ",\"diagnostic\":" ++ optionalImplementationLinkDiagnosticJson record.diagnostic ++ "}"

private def semanticVerdictDiagnosticJson
    (diagnostic : ArtifactSemanticVerdictDiagnostic) : String :=
  "{\"kind\":" ++ quoteResult diagnostic.kind ++
    ",\"relatedDefinitionIds\":" ++ resultArray
      (diagnostic.relatedDefinitionIds.map (quoteResult ∘ DefinitionId.value)) ++
    ",\"observationDiagnostic\":" ++ optionalResultJson
      (diagnostic.observationDiagnostic.map observationDiagnosticJson) ++ "}"

private def optionalSemanticVerdictDiagnosticJson
    (diagnostic : Option ArtifactSemanticVerdictDiagnostic) : String :=
  optionalResultJson (diagnostic.map semanticVerdictDiagnosticJson)

private def semanticClauseVerdictJson (verdict : ArtifactSemanticClauseVerdict) : String :=
  "{\"propertyDefinitionId\":" ++ quoteResult verdict.propertyDefinitionId.value ++
    ",\"clauseDefinitionId\":" ++ quoteResult verdict.clauseDefinitionId.value ++
    ",\"status\":" ++ quoteResult verdict.status ++
    ",\"coordinates\":" ++ resultArray (verdict.coordinates.map modelCoordinateJson) ++
    ",\"queryLimits\":" ++ queryLimitsJson verdict.queryLimits ++
    ",\"propertyLimit\":" ++ optionalArtifactLimitJson verdict.propertyLimit ++
    ",\"evidenceLimit\":" ++ artifactLimitJson verdict.evidenceLimit ++
    ",\"provenanceDefinitionIds\":" ++ resultArray
      (verdict.provenanceDefinitionIds.map (quoteResult ∘ DefinitionId.value)) ++
    ",\"evidenceLinks\":" ++ resultArray (verdict.evidenceLinks.map evidenceLinkJson) ++ "}"

private def propertyVerdictJson (verdict : ArtifactPropertyVerdict) : String :=
  "{\"queryDefinitionId\":" ++ quoteResult verdict.queryDefinitionId.value ++
    ",\"propertyDefinitionId\":" ++ quoteResult verdict.propertyDefinitionId.value ++
    ",\"propertyBehaviorFingerprint\":" ++ quoteResult verdict.propertyBehaviorFingerprint.render ++
    ",\"traceId\":" ++ optionalStringJson verdict.traceId ++
    ",\"status\":" ++ quoteResult verdict.status ++
    ",\"queryLimits\":" ++ queryLimitsJson verdict.queryLimits ++
    ",\"evidenceLimit\":" ++ optionalResultJson (verdict.evidenceLimit.map artifactLimitJson) ++
    ",\"provenanceDefinitionIds\":" ++ resultArray
      (verdict.provenanceDefinitionIds.map (quoteResult ∘ DefinitionId.value)) ++
    ",\"clauses\":" ++ resultArray (verdict.clauses.map semanticClauseVerdictJson) ++
    ",\"diagnostic\":" ++ optionalSemanticVerdictDiagnosticJson verdict.diagnostic ++ "}"

private def querySummaryJson (summary : ArtifactQuerySummary) : String :=
  "{\"queryDefinitionId\":" ++ quoteResult summary.queryDefinitionId.value ++
    ",\"status\":" ++ quoteResult summary.status ++
    ",\"queryLimits\":" ++ queryLimitsJson summary.queryLimits ++
    ",\"requiredPropertyDefinitionIds\":" ++ resultArray
      (summary.requiredPropertyDefinitionIds.map (quoteResult ∘ DefinitionId.value)) ++
    ",\"propertyVerdicts\":" ++ resultArray (summary.propertyVerdicts.map propertyVerdictJson) ++
    ",\"missingPropertyDefinitionIds\":" ++ resultArray
      (summary.missingPropertyDefinitionIds.map (quoteResult ∘ DefinitionId.value)) ++
    ",\"duplicatePropertyDefinitionIds\":" ++ resultArray
      (summary.duplicatePropertyDefinitionIds.map (quoteResult ∘ DefinitionId.value)) ++
    ",\"unexpectedPropertyDefinitionIds\":" ++ resultArray
      (summary.unexpectedPropertyDefinitionIds.map (quoteResult ∘ DefinitionId.value)) ++
    ",\"divergentPropertyDefinitionIds\":" ++ resultArray
      (summary.divergentPropertyDefinitionIds.map (quoteResult ∘ DefinitionId.value)) ++
    ",\"wrongQueryResultDefinitionIds\":" ++ resultArray
      (summary.wrongQueryResultDefinitionIds.map (quoteResult ∘ DefinitionId.value)) ++
    ",\"traceIds\":" ++ resultArray (summary.traceIds.map quoteResult) ++ "}"

private def stagedLimitJson (staged : ArtifactStagedLimit) : String :=
  "{\"stage\":" ++ quoteResult staged.stage ++
    ",\"limit\":" ++ artifactLimitJson staged.limit ++ "}"

private def resultArtifactContentJson (result : ResultArtifact) : String :=
  "{\"formatVersion\":" ++ quoteResult result.formatVersion ++
    ",\"runIdentity\":" ++ quoteResult result.runIdentity.value ++
    ",\"behaviorFingerprint\":" ++ quoteResult result.behaviorFingerprint.render ++
    ",\"experiment\":" ++ result.experiment.canonicalJson ++
    ",\"runtimeConfiguration\":" ++ result.runtimeConfiguration.canonicalJson ++
    ",\"run\":" ++ result.run.canonicalJson ++
    ",\"rawEvidence\":" ++ result.rawEvidence.canonicalJson ++
    ",\"evidence\":" ++ result.evidence.canonicalJson ++
    ",\"operationalStatus\":" ++ quoteResult result.operationalStatus ++
    ",\"observationEvaluationStatus\":" ++ quoteResult result.observationEvaluationStatus ++
    ",\"implementationLink\":" ++ implementationLinkRecordJson result.implementationLink ++
    ",\"implementationLinkStatus\":" ++ quoteResult result.implementationLinkStatus ++
    ",\"propertyVerdicts\":" ++ resultArray (result.propertyVerdicts.map propertyVerdictJson) ++
    ",\"querySummary\":" ++ querySummaryJson result.querySummary ++
    ",\"semanticStatus\":" ++ quoteResult result.semanticStatus ++
    ",\"limits\":" ++ resultArray (result.limits.map stagedLimitJson) ++
    ",\"knownGaps\":" ++ resultArray (result.knownGaps.map canonicalKnownGapJson) ++
    ",\"cleanupStatus\":" ++ quoteResult result.cleanupStatus ++
    ",\"evaluationOutcomeChecksum\":" ++
      optionalChecksumResultJson result.evaluationOutcomeChecksum ++
    ",\"provenance\":" ++ result.provenance.canonicalJson ++
    ",\"provenanceChecksum\":" ++ quoteResult result.provenanceChecksum.render ++ "}"

def ResultArtifact.expectedArtifactChecksum (result : ResultArtifact) : ArtifactChecksum :=
  resultChecksumOf (Json.prettyBytes (resultArtifactContentJson result))

def ResultArtifact.seal (result : ResultArtifact) : ResultArtifact :=
  let withProvenance := { result with provenanceChecksum := result.provenance.expectedChecksum }
  { withProvenance with artifactChecksum := withProvenance.expectedArtifactChecksum }

def ResultArtifact.hasValidChecksums (result : ResultArtifact) : Bool :=
  result.provenanceChecksum == result.provenance.expectedChecksum &&
    result.artifactChecksum == result.expectedArtifactChecksum

def canonicalResultArtifactJson (result : ResultArtifact) : String :=
  let content := resultArtifactContentJson result
  Json.pretty ((content.dropEnd 1).toString ++
    ",\"artifactChecksum\":" ++ quoteResult result.artifactChecksum.render ++ "}")

def canonicalResultArtifactBytes (result : ResultArtifact) : String :=
  canonicalResultArtifactJson result ++ "\n"

private def portablePropertyResultJson (property : PortableProperty) : String :=
  "{\"definitionId\":" ++ quoteResult property.definitionId.value ++
    ",\"behaviorFingerprint\":" ++ quoteResult property.behaviorFingerprint.render ++
    ",\"requirementDefinitionIds\":" ++ resultArray
      (property.requirementDefinitionIds.map (quoteResult ∘ DefinitionId.value)) ++ "}"

private def evaluationOutcomeJson
    (plan : DrivePlan)
    (trace : ArtifactEvidenceBackedModelTrace)
    (evidenceLinks : List ArtifactEvidenceLink)
    (observationProgram mapping : ArtifactDefinitionReference)
    (implementationLink : ArtifactImplementationLinkRecord)
    (querySummary : ArtifactQuerySummary)
    (properties : List PortableProperty)
    (propertyVerdicts : List ArtifactPropertyVerdict)
    (limits : List ArtifactStagedLimit) : String :=
  "{\"plan\":" ++ canonicalDrivePlanJson plan ++
    ",\"evidenceBackedModelTrace\":" ++ evidenceBackedModelTraceJson trace ++
    ",\"evidenceLinks\":" ++ resultArray (evidenceLinks.map evidenceLinkJson) ++
    ",\"observationProgram\":" ++ definitionReferenceJson observationProgram ++
    ",\"mapping\":" ++ definitionReferenceJson mapping ++
    ",\"implementationLink\":" ++ implementationLinkRecordJson implementationLink ++
    ",\"querySummary\":" ++ querySummaryJson querySummary ++
    ",\"properties\":" ++ resultArray (properties.map portablePropertyResultJson) ++
    ",\"propertyVerdicts\":" ++ resultArray (propertyVerdicts.map propertyVerdictJson) ++
    ",\"limits\":" ++ resultArray (limits.map stagedLimitJson) ++ "}"

/-- Recompute only the frozen stable Generated View; no evaluator is run here. -/
def ResultArtifact.expectedEvaluationOutcomeChecksum
    (result : ResultArtifact)
    (evidence : EvidenceArtifact)
    (experiment : ExperimentSpec) : Option ArtifactChecksum :=
  if result.semanticStatus == "satisfied" || result.semanticStatus == "violated" then do
    let trace ← evidence.evidenceBackedModelTrace
    some <| evaluationOutcomeChecksumOf <| Json.prettyBytes <|
      evaluationOutcomeJson experiment.plan trace evidence.evidenceLinks evidence.observationProgram
        evidence.mapping result.implementationLink result.querySummary experiment.properties
        result.propertyVerdicts result.limits
  else none

private def implementationLinkFailureStatus? (kind : String) : Option String :=
  if ["stale-source-target", "stale-destination-target", "behavior-fingerprint-drift",
      "source-setup-mismatch", "non-authoritative-source-initial",
      "non-authoritative-source-step", "invalid-coordinate"].contains kind then
    some "invalid"
  else if ["absent-coordinate", "limit-reached"].contains kind then
    some "unknown"
  else if ["duplicate-coordinate", "contradictory-coordinate", "multiple-mappings",
      "evidence-link-mismatch"].contains kind then
    some "conflict"
  else if ["known-gap", "unsupported-vocabulary"].contains kind then
    some "unsupported"
  else none

private def coordinateIdentityName (coordinate : ArtifactModelCoordinate) : Option String :=
  match coordinate.kind, coordinate.step, coordinate.position with
  | "initial-state", none, none => some "initial-state"
  | kind, some step, none =>
      if step > 0 && ["selected-action", "model-outcome", "resulting-state"].contains kind then
        some (kind ++ ":" ++ toString step)
      else none
  | "observation", some step, some position =>
      if step > 0 && position > 0 then some ("observation:" ++ toString step ++ ":" ++ toString position)
      else none
  | _, _, _ => none

private def implementationDiagnosticTargetIdentityJson
    (target : ArtifactImplementationTargetReference) : String :=
  "{\"id\":" ++ quoteResult target.definitionId.value ++
    ",\"kind\":" ++ quoteResult target.kind.name ++
    ",\"behaviorFingerprint\":" ++ quoteResult target.behaviorFingerprint.render ++ "}"

private def implementationDiagnosticIdentityJson
    (record : ArtifactImplementationLinkRecord)
    (diagnostic : ArtifactImplementationLinkDiagnostic)
    (status : String) : String :=
  "{\"implementationLinkId\":" ++ quoteResult record.definitionId.value ++
    ",\"implementationLinkBehaviorFingerprint\":" ++ quoteResult record.behaviorFingerprint.render ++
    ",\"sourceTarget\":" ++ implementationDiagnosticTargetIdentityJson record.sourceTarget ++
    ",\"destinationTarget\":" ++ implementationDiagnosticTargetIdentityJson record.destinationTarget ++
    ",\"kind\":" ++ quoteResult diagnostic.kind ++
    ",\"status\":" ++ quoteResult status ++
    ",\"coordinate\":" ++ optionalStringJson
      (diagnostic.coordinate.bind coordinateIdentityName) ++
    ",\"relatedDefinitionIds\":" ++ resultArray
      (diagnostic.relatedDefinitionIds.map (quoteResult ∘ DefinitionId.value)) ++
    ",\"sourceSetupBehaviorFingerprint\":" ++
      optionalFingerprintResultJson diagnostic.sourceSetupBehaviorFingerprint ++
    ",\"appliedLimit\":" ++ optionalArtifactLimitJson diagnostic.appliedLimit ++
    ",\"observedCount\":" ++ optionalNatResultJson diagnostic.observedCount ++
    ",\"knownGapCode\":" ++ optionalIdResultJson diagnostic.knownGapCode ++
    ",\"knownGapReason\":" ++ optionalStringJson diagnostic.knownGapReason ++
    ",\"unsupportedVocabularyKind\":" ++ optionalStringJson
      (diagnostic.unsupportedVocabularyKind.map DefinitionKind.name) ++
    ",\"evidenceLinkBehaviorFingerprint\":" ++
      optionalFingerprintResultJson diagnostic.evidenceLinkBehaviorFingerprint ++ "}"

/-- Fingerprint only the frozen pretty diagnostic projection; no link application occurs. -/
def ArtifactImplementationLinkRecord.expectedDiagnosticIdentity
    (record : ArtifactImplementationLinkRecord) : Option BehaviorFingerprint := do
  let diagnostic ← record.diagnostic
  let status ← implementationLinkFailureStatus? diagnostic.kind
  some <| behaviorFingerprintOf <| Json.prettyBytes <|
    implementationDiagnosticIdentityJson record diagnostic status

def RawEvidence.artifactBinding (evidence : RawEvidence) : ArtifactBinding := {
  formatVersion := evidence.formatVersion
  artifactChecksum := evidence.artifactChecksum
  behaviorFingerprint := evidence.behaviorFingerprint
  provenanceChecksum := evidence.provenanceChecksum
}

def EvidenceArtifact.artifactBinding (evidence : EvidenceArtifact) : ArtifactBinding := {
  formatVersion := evidence.formatVersion
  artifactChecksum := evidence.artifactChecksum
  behaviorFingerprint := evidence.behaviorFingerprint
  provenanceChecksum := evidence.provenanceChecksum
}

private def observationStatusMatrixValid (evidence : EvidenceArtifact) : Bool :=
  if evidence.observationEvaluationStatus == "accepted" then
    evidence.evidenceBackedModelTrace.isSome && evidence.evidenceLinks != [] && evidence.diagnostics == []
  else if ["unknown", "conflict", "unsupported"].contains evidence.observationEvaluationStatus then
    evidence.evidenceBackedModelTrace.isNone && evidence.evidenceLinks == [] && evidence.diagnostics.length == 1
  else false

/-- Structural transport validation only; it does not interpret RawEvidence. -/
def EvidenceArtifact.isValidTransport (evidence : EvidenceArtifact) : Bool :=
  evidence.formatVersion == "umpire-evidence/v2" && evidence.runIdentity.isNamespaced &&
    evidence.experiment.formatVersion == "umpire-experiment/v2" &&
    evidence.runtimeConfiguration.formatVersion == "umpire-runtime-configuration/v2" &&
    evidence.run.formatVersion == "umpire-experiment-run/v2" &&
    evidence.rawEvidence.formatVersion == "umpire-raw-evidence/v2" &&
    observationStatusMatrixValid evidence && (validateKnownGaps evidence.knownGaps).isOk &&
    evidence.provenance.isValidTransport && evidence.hasValidChecksums

def EvidenceArtifact.closes
    (evidence : EvidenceArtifact)
    (experiment : ExperimentSpec)
    (configuration : RuntimeConfiguration)
    (run : ExperimentRun)
    (rawEvidence : RawEvidence) : Bool :=
  rawEvidence.closes experiment configuration run &&
    evidence.experiment == experiment.artifactBinding &&
    evidence.runtimeConfiguration == configuration.artifactBinding &&
    evidence.run == run.artifactBinding && evidence.rawEvidence == rawEvidence.artifactBinding &&
    evidence.runIdentity == run.runIdentity &&
    evidence.observationProgram.definitionId == configuration.observation.programDefinitionId &&
    evidence.observationProgram.behaviorFingerprint ==
      configuration.observation.programBehaviorFingerprint &&
    evidence.mapping.definitionId == configuration.observation.mappingDefinitionId &&
    evidence.mapping.behaviorFingerprint == configuration.observation.mappingBehaviorFingerprint

private def resultStatusMatrixValid (result : ResultArtifact) : Bool :=
  let resolved := result.semanticStatus == "satisfied" || result.semanticStatus == "violated"
  (resolved == result.evaluationOutcomeChecksum.isSome) &&
    result.semanticStatus == result.querySummary.status &&
    result.propertyVerdicts == result.querySummary.propertyVerdicts &&
    if result.observationEvaluationStatus != "accepted" then
      result.implementationLinkStatus == "not-evaluated" && result.propertyVerdicts == [] &&
        result.semanticStatus == "incomplete"
    else if result.implementationLinkStatus != "applied" then
      result.propertyVerdicts == [] && result.semanticStatus == "incomplete"
    else result.propertyVerdicts != []

/-- Structural Result validation keeps every stage status independent. -/
def ResultArtifact.isValidTransport (result : ResultArtifact) : Bool :=
  result.formatVersion == "umpire-result/v2" && result.runIdentity.isNamespaced &&
    ["succeeded", "failed", "incomplete"].contains result.operationalStatus &&
    ["accepted", "unknown", "conflict", "unsupported"].contains
      result.observationEvaluationStatus &&
    ["applied", "invalid", "unknown", "conflict", "unsupported", "not-evaluated"].contains
      result.implementationLinkStatus &&
    ["satisfied", "violated", "incomplete"].contains result.semanticStatus &&
    ["complete", "incomplete", "failed"].contains result.cleanupStatus &&
    resultStatusMatrixValid result && (validateKnownGaps result.knownGaps).isOk &&
    result.provenance.isValidTransport && result.hasValidChecksums

def ResultArtifact.closes
    (result : ResultArtifact)
    (experiment : ExperimentSpec)
    (configuration : RuntimeConfiguration)
    (run : ExperimentRun)
    (rawEvidence : RawEvidence)
    (evidence : EvidenceArtifact) : Bool :=
  evidence.closes experiment configuration run rawEvidence &&
    result.experiment == experiment.artifactBinding &&
    result.runtimeConfiguration == configuration.artifactBinding &&
    result.run == run.artifactBinding && result.rawEvidence == rawEvidence.artifactBinding &&
    result.evidence == evidence.artifactBinding && result.runIdentity == run.runIdentity &&
    result.operationalStatus == run.operationalStatus.name &&
    result.observationEvaluationStatus == evidence.observationEvaluationStatus &&
    result.cleanupStatus == run.cleanup.status.name &&
    (match result.evaluationOutcomeChecksum with
      | none => result.expectedEvaluationOutcomeChecksum evidence experiment |>.isNone
      | some checksum => result.expectedEvaluationOutcomeChecksum evidence experiment == some checksum)

end Umpire
