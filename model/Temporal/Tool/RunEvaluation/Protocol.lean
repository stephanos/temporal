import Temporal.Feature.Nexus.Experimental.CallerClosure
import Temporal.System.Execution.Nexus
import Temporal.System.Nexus.Observation
import Umpire.Artifact.Result
import Umpire.Json

/-!
The private checker protocol is one closed, bounded canonical v2 Generated View. It contains no
path, persisted artifact bytes, callback, environment option, or extension point.
-/

namespace Temporal.Tool.RunEvaluation.Protocol

open Umpire

def requestFormatVersion : String := "umpire-semantic-check-request/v2"
def responseFormatVersion : String := "umpire-semantic-check-response/v2"
def checkerIdentity : String := "temporal.nexus.caller-closure.run-evaluation"
def checkerVersion : Nat := 2
def maxBytes : Nat := 32 * 1024 * 1024

def checkerBehaviorFingerprint : BehaviorFingerprint :=
  behaviorFingerprintOf "temporal-nexus-caller-closure-run-evaluation-checker/v2"

/-- One exact compiled Definition identity admitted by the private protocol. -/
structure DefinitionReference where
  definitionId : DefinitionId
  behaviorFingerprint : BehaviorFingerprint
  deriving BEq, DecidableEq, Repr

/-- One complete Property identity as embedded in the compiled ExperimentSpec. -/
structure PropertyReference where
  definitionId : DefinitionId
  behaviorFingerprint : BehaviorFingerprint
  requirementDefinitionIds : List DefinitionId
  deriving BEq, DecidableEq, Repr

/-- The only direct checker request. Complex runtime projections remain JSON values until the
Temporal adapter validates their exact closed source schemas. -/
structure Request where
  formatVersion : String
  checkerIdentity : String
  checkerVersion : Nat
  checkerBehaviorFingerprint : BehaviorFingerprint
  experiment : ArtifactBinding
  runtimeConfiguration : ArtifactBinding
  run : ArtifactBinding
  rawEvidence : ArtifactBinding
  runIdentity : DefinitionId
  query : DefinitionReference
  properties : List PropertyReference
  observationProgram : DefinitionReference
  mapping : DefinitionReference
  phaseOutcomes : Lean.Json
  controlAttempts : Lean.Json
  sourceClosures : Lean.Json
  captureStatus : String
  sources : Lean.Json
  facts : Lean.Json
  runKnownGaps : Lean.Json
  rawEvidenceKnownGaps : Lean.Json
  deriving BEq

/-- The only direct checker response. All semantic projections are already-evaluated JSON values,
not a second semantic or persisted representation. -/
structure Response where
  formatVersion : String
  checkerIdentity : String
  checkerVersion : Nat
  checkerBehaviorFingerprint : BehaviorFingerprint
  experimentArtifactChecksum : ArtifactChecksum
  runtimeConfigurationArtifactChecksum : ArtifactChecksum
  runArtifactChecksum : ArtifactChecksum
  rawEvidenceArtifactChecksum : ArtifactChecksum
  experimentBehaviorFingerprint : BehaviorFingerprint
  runtimeConfigurationBehaviorFingerprint : BehaviorFingerprint
  runIdentity : DefinitionId
  observationEvaluationStatus : String
  evidenceBackedModelTrace : Lean.Json
  evidenceLinks : Lean.Json
  dispositions : Lean.Json
  diagnostics : Lean.Json
  observationKnownGaps : Lean.Json
  propertyVerdicts : Lean.Json
  querySummary : Lean.Json
  semanticStatus : String
  resultKnownGaps : Lean.Json
  evaluationOutcomeChecksum : Option ArtifactChecksum
  deriving BEq

inductive ErrorKind where
  | oversized
  | invalidUtf8
  | malformedJson
  | nonCanonical
  | wrongShape
  | invalidValue
  | closureDrift
  deriving BEq, DecidableEq, Ord, Repr

structure Error where
  kind : ErrorKind
  field : String
  deriving BEq, DecidableEq, Repr

private def error (kind : ErrorKind) (field : String) : Error := { kind, field }

private def quote (value : String) : String := Lean.Json.compress (.str value)

private def array (values : List String) : String :=
  "[" ++ String.intercalate "," values ++ "]"

private def jsonValue (value : Lean.Json) : String := Lean.Json.compress value

private def referenceJson (reference : DefinitionReference) : String :=
  "{\"definitionId\":" ++ quote reference.definitionId.value ++
    ",\"behaviorFingerprint\":" ++ quote reference.behaviorFingerprint.render ++ "}"

private def propertyJson (property : PropertyReference) : String :=
  "{\"definitionId\":" ++ quote property.definitionId.value ++
    ",\"behaviorFingerprint\":" ++ quote property.behaviorFingerprint.render ++
    ",\"requirementDefinitionIds\":" ++
      array (property.requirementDefinitionIds.map (quote ∘ DefinitionId.value)) ++ "}"

private def optionalChecksumJson (checksum : Option ArtifactChecksum) : String :=
  checksum.map (quote ∘ ArtifactChecksum.render) |>.getD "null"

def canonicalResponseJson (response : Response) : String :=
  "{\"formatVersion\":" ++ quote response.formatVersion ++
    ",\"checkerIdentity\":" ++ quote response.checkerIdentity ++
    ",\"checkerVersion\":" ++ toString response.checkerVersion ++
    ",\"checkerBehaviorFingerprint\":" ++
      quote response.checkerBehaviorFingerprint.render ++
    ",\"experimentArtifactChecksum\":" ++
      quote response.experimentArtifactChecksum.render ++
    ",\"runtimeConfigurationArtifactChecksum\":" ++
      quote response.runtimeConfigurationArtifactChecksum.render ++
    ",\"runArtifactChecksum\":" ++ quote response.runArtifactChecksum.render ++
    ",\"rawEvidenceArtifactChecksum\":" ++
      quote response.rawEvidenceArtifactChecksum.render ++
    ",\"experimentBehaviorFingerprint\":" ++
      quote response.experimentBehaviorFingerprint.render ++
    ",\"runtimeConfigurationBehaviorFingerprint\":" ++
      quote response.runtimeConfigurationBehaviorFingerprint.render ++
    ",\"runIdentity\":" ++ quote response.runIdentity.value ++
    ",\"observationEvaluationStatus\":" ++ quote response.observationEvaluationStatus ++
    ",\"evidenceBackedModelTrace\":" ++ jsonValue response.evidenceBackedModelTrace ++
    ",\"evidenceLinks\":" ++ jsonValue response.evidenceLinks ++
    ",\"dispositions\":" ++ jsonValue response.dispositions ++
    ",\"diagnostics\":" ++ jsonValue response.diagnostics ++
    ",\"observationKnownGaps\":" ++ jsonValue response.observationKnownGaps ++
    ",\"propertyVerdicts\":" ++ jsonValue response.propertyVerdicts ++
    ",\"querySummary\":" ++ jsonValue response.querySummary ++
    ",\"semanticStatus\":" ++ quote response.semanticStatus ++
    ",\"resultKnownGaps\":" ++ jsonValue response.resultKnownGaps ++
    ",\"evaluationOutcomeChecksum\":" ++
      optionalChecksumJson response.evaluationOutcomeChecksum ++ "}"

def encodeResponse (response : Response) : Except Error ByteArray := do
  let bytes := (Json.prettyBytes (canonicalResponseJson response)).toUTF8
  if bytes.size > maxBytes then
    throw (error .oversized "response")
  pure bytes

private def canonicalRequestJson (request : Request) : String :=
  "{\"formatVersion\":" ++ quote request.formatVersion ++
    ",\"checkerIdentity\":" ++ quote request.checkerIdentity ++
    ",\"checkerVersion\":" ++ toString request.checkerVersion ++
    ",\"checkerBehaviorFingerprint\":" ++
      quote request.checkerBehaviorFingerprint.render ++
    ",\"experiment\":" ++ request.experiment.canonicalJson ++
    ",\"runtimeConfiguration\":" ++ request.runtimeConfiguration.canonicalJson ++
    ",\"run\":" ++ request.run.canonicalJson ++
    ",\"rawEvidence\":" ++ request.rawEvidence.canonicalJson ++
    ",\"runIdentity\":" ++ quote request.runIdentity.value ++
    ",\"query\":" ++ referenceJson request.query ++
    ",\"properties\":" ++ array (request.properties.map propertyJson) ++
    ",\"observationProgram\":" ++ referenceJson request.observationProgram ++
    ",\"mapping\":" ++ referenceJson request.mapping ++
    ",\"phaseOutcomes\":" ++ jsonValue request.phaseOutcomes ++
    ",\"controlAttempts\":" ++ jsonValue request.controlAttempts ++
    ",\"sourceClosures\":" ++ jsonValue request.sourceClosures ++
    ",\"captureStatus\":" ++ quote request.captureStatus ++
    ",\"sources\":" ++ jsonValue request.sources ++
    ",\"facts\":" ++ jsonValue request.facts ++
    ",\"runKnownGaps\":" ++ jsonValue request.runKnownGaps ++
    ",\"rawEvidenceKnownGaps\":" ++ jsonValue request.rawEvidenceKnownGaps ++ "}"

def encodeRequest (request : Request) : Except Error ByteArray := do
  let bytes := (Json.prettyBytes (canonicalRequestJson request)).toUTF8
  if bytes.size > maxBytes then
    throw (error .oversized "request")
  pure bytes

private def compactAux : List Char → Bool → Bool → List Char → List Char
  | [], _, _, output => output.reverse
  | character :: rest, insideString, escaped, output =>
      if insideString then
        if escaped then
          compactAux rest true false (character :: output)
        else if character == '\\' then
          compactAux rest true true (character :: output)
        else if character == '"' then
          compactAux rest false false (character :: output)
        else
          compactAux rest true false (character :: output)
      else if character == '"' then
        compactAux rest true false (character :: output)
      else if character.isWhitespace then
        compactAux rest false false output
      else
        compactAux rest false false (character :: output)

private def compact (value : String) : String :=
  String.ofList (compactAux value.toList false false [])

private def canonicalText (bytes : ByteArray) : Except Error String := do
  if bytes.size > maxBytes then
    throw (error .oversized "request")
  let text ← match String.fromUTF8? bytes with
    | some value => pure value
    | none => throw (error .invalidUtf8 "request")
  if !(text.endsWith "\n") || ((text.dropEnd 1).toString.endsWith "\n") then
    throw (error .nonCanonical "request")
  let body := (text.dropEnd 1).toString
  if Json.pretty (compact body) != body then
    throw (error .nonCanonical "request")
  pure text

private def parseJson (text : String) : Except Error Lean.Json :=
  match Lean.Json.parse text with
  | .ok value => pure value
  | .error _ => throw (error .malformedJson "request")

private def objectHasExactly (json : Lean.Json) (fields : List String) : Bool :=
  match json.getObj? with
  | .error _ => false
  | .ok object =>
      object.toList.length == fields.length &&
        fields.all fun field => (json.getObjVal? field).isOk

private def getValue (json : Lean.Json) (field : String) : Except Error Lean.Json :=
  match json.getObjVal? field with
  | .ok value => pure value
  | .error _ => throw (error .wrongShape field)

private def getString (json : Lean.Json) (field : String) : Except Error String := do
  let value ← getValue json field
  match value.getStr? with
  | .ok string => pure string
  | .error _ => throw (error .wrongShape field)

private def getNat (json : Lean.Json) (field : String) : Except Error Nat := do
  let value ← getValue json field
  match value.getNat? with
  | .ok natural => pure natural
  | .error _ => throw (error .wrongShape field)

private def parseChecksum (value field : String) : Except Error ArtifactChecksum :=
  match ArtifactChecksum.parse? value with
  | some checksum => pure checksum
  | none => throw (error .invalidValue field)

private def parseFingerprint (value field : String) : Except Error BehaviorFingerprint :=
  match BehaviorFingerprint.parse? value with
  | some fingerprint => pure fingerprint
  | none => throw (error .invalidValue field)

private def parseId (value field : String) : Except Error DefinitionId := do
  let definitionId := DefinitionId.of value
  if !definitionId.isNamespaced then
    throw (error .invalidValue field)
  pure definitionId

private def parseBinding (json : Lean.Json) (field : String) : Except Error ArtifactBinding := do
  if !objectHasExactly json
      ["formatVersion", "artifactChecksum", "behaviorFingerprint", "provenanceChecksum"] then
    throw (error .wrongShape field)
  pure {
    formatVersion := ← getString json "formatVersion"
    artifactChecksum := ← parseChecksum (← getString json "artifactChecksum") field
    behaviorFingerprint := ←
      parseFingerprint (← getString json "behaviorFingerprint") field
    provenanceChecksum := ← parseChecksum (← getString json "provenanceChecksum") field
  }

private def parseReference (json : Lean.Json) (field : String) : Except Error DefinitionReference := do
  if !objectHasExactly json ["definitionId", "behaviorFingerprint"] then
    throw (error .wrongShape field)
  pure {
    definitionId := ← parseId (← getString json "definitionId") field
    behaviorFingerprint := ←
      parseFingerprint (← getString json "behaviorFingerprint") field
  }

private def stringArray (json : Lean.Json) (field : String) : Except Error (List String) := do
  let values ← match json.getArr? with
    | .ok values => pure values
    | .error _ => throw (error .wrongShape field)
  values.toList.mapM fun value =>
    match value.getStr? with
    | .ok string => pure string
    | .error _ => throw (error .wrongShape field)

private def parseProperty (json : Lean.Json) : Except Error PropertyReference := do
  if !objectHasExactly json
      ["definitionId", "behaviorFingerprint", "requirementDefinitionIds"] then
    throw (error .wrongShape "properties")
  let requirements ← stringArray (← getValue json "requirementDefinitionIds") "properties"
  pure {
    definitionId := ← parseId (← getString json "definitionId") "properties"
    behaviorFingerprint := ←
      parseFingerprint (← getString json "behaviorFingerprint") "properties"
    requirementDefinitionIds := ← requirements.mapM fun value => parseId value "properties"
  }

private def parseProperties (json : Lean.Json) : Except Error (List PropertyReference) := do
  let values ← match json.getArr? with
    | .ok values => pure values
    | .error _ => throw (error .wrongShape "properties")
  values.toList.mapM parseProperty

def expectedExperimentBinding : ArtifactBinding :=
  Temporal.System.Execution.Nexus.experimentBinding

def expectedRuntimeConfigurationBinding : ArtifactBinding :=
  (Temporal.System.Execution.Nexus.runtimeConfigurationFor
    Temporal.Feature.Nexus.Experimental.CallerClosure.compiledArtifact).artifactBinding

def expectedQuery : DefinitionReference := {
  definitionId := Temporal.Feature.Nexus.Experimental.CallerClosure.exactActionQuery.id
  behaviorFingerprint :=
    Temporal.Feature.Nexus.Experimental.CallerClosure.exactActionQuery.behaviorFingerprint
}

def expectedProperties : List PropertyReference :=
  Temporal.Feature.Nexus.Experimental.CallerClosure.compiledArtifact.properties.map fun property => {
    definitionId := property.definitionId
    behaviorFingerprint := property.behaviorFingerprint
    requirementDefinitionIds := property.requirementDefinitionIds
  }

def expectedObservationProgram : DefinitionReference := {
  definitionId :=
    Temporal.System.Execution.Nexus.canonicalObservationProgramDefinition.reference.definitionId
  behaviorFingerprint :=
    Temporal.System.Execution.Nexus.canonicalObservationProgramDefinition.reference.behaviorFingerprint
}

def expectedMapping : DefinitionReference := {
  definitionId := Temporal.System.Nexus.Observation.checkedPlan.id
  behaviorFingerprint := Temporal.System.Nexus.Observation.checkedPlan.behaviorFingerprint
}

private def validateClosure (request : Request) : Except Error Unit := do
  if request.formatVersion != requestFormatVersion ||
      request.checkerIdentity != checkerIdentity || request.checkerVersion != checkerVersion ||
      request.checkerBehaviorFingerprint != checkerBehaviorFingerprint then
    throw (error .closureDrift "checker")
  if request.experiment != expectedExperimentBinding then
    throw (error .closureDrift "experiment")
  if request.runtimeConfiguration != expectedRuntimeConfigurationBinding then
    throw (error .closureDrift "runtimeConfiguration")
  if request.run.formatVersion != "umpire-experiment-run/v2" ||
      request.rawEvidence.formatVersion != "umpire-raw-evidence/v2" then
    throw (error .closureDrift "run")
  if request.query != expectedQuery || request.properties != expectedProperties ||
      request.observationProgram != expectedObservationProgram ||
      request.mapping != expectedMapping then
    throw (error .closureDrift "semantics")
  if request.captureStatus != "closed" && request.captureStatus != "partial" &&
      request.captureStatus != "failed" then
    throw (error .invalidValue "captureStatus")

def decodeRequest (bytes : ByteArray) : Except Error Request := do
  let text ← canonicalText bytes
  let json ← parseJson text
  let fields := [
    "formatVersion", "checkerIdentity", "checkerVersion", "checkerBehaviorFingerprint",
    "experiment", "runtimeConfiguration", "run", "rawEvidence", "runIdentity", "query",
    "properties", "observationProgram", "mapping", "phaseOutcomes", "controlAttempts",
    "sourceClosures", "captureStatus", "sources", "facts", "runKnownGaps",
    "rawEvidenceKnownGaps"
  ]
  if !objectHasExactly json fields then
    throw (error .wrongShape "request")
  let request : Request := {
    formatVersion := ← getString json "formatVersion"
    checkerIdentity := ← getString json "checkerIdentity"
    checkerVersion := ← getNat json "checkerVersion"
    checkerBehaviorFingerprint := ←
      parseFingerprint (← getString json "checkerBehaviorFingerprint")
        "checkerBehaviorFingerprint"
    experiment := ← parseBinding (← getValue json "experiment") "experiment"
    runtimeConfiguration := ←
      parseBinding (← getValue json "runtimeConfiguration") "runtimeConfiguration"
    run := ← parseBinding (← getValue json "run") "run"
    rawEvidence := ← parseBinding (← getValue json "rawEvidence") "rawEvidence"
    runIdentity := ← parseId (← getString json "runIdentity") "runIdentity"
    query := ← parseReference (← getValue json "query") "query"
    properties := ← parseProperties (← getValue json "properties")
    observationProgram := ←
      parseReference (← getValue json "observationProgram") "observationProgram"
    mapping := ← parseReference (← getValue json "mapping") "mapping"
    phaseOutcomes := ← getValue json "phaseOutcomes"
    controlAttempts := ← getValue json "controlAttempts"
    sourceClosures := ← getValue json "sourceClosures"
    captureStatus := ← getString json "captureStatus"
    sources := ← getValue json "sources"
    facts := ← getValue json "facts"
    runKnownGaps := ← getValue json "runKnownGaps"
    rawEvidenceKnownGaps := ← getValue json "rawEvidenceKnownGaps"
  }
  validateClosure request
  if (← encodeRequest request) != bytes then
    throw (error .nonCanonical "request")
  pure request

end Temporal.Tool.RunEvaluation.Protocol
