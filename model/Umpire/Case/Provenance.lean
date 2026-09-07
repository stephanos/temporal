import Testpilot.Authoring
import Umpire.Case
import Umpire.Json

/-!
Umpire-owned provenance encoding for generated Testpilot Cases.

Testpilot treats these bytes as opaque. The field names, enum spellings, list order, optional-field
elision, indentation, and terminal newline remain the producer contract established by the original
Umpire Case encoder.
-/

namespace Umpire.Case.Provenance

open Umpire.CanonicalJson

private def definitionKind : CaseDefinitionKind → String
  | .setup => "CASE_DEFINITION_KIND_SETUP"
  | .state => "CASE_DEFINITION_KIND_STATE"
  | .action => "CASE_DEFINITION_KIND_ACTION"
  | .outcome => "CASE_DEFINITION_KIND_OUTCOME"
  | .observation => "CASE_DEFINITION_KIND_OBSERVATION"
  | .relation => "CASE_DEFINITION_KIND_RELATION"
  | .capability => "CASE_DEFINITION_KIND_CAPABILITY"
  | .property => "CASE_DEFINITION_KIND_PROPERTY"
  | .query => "CASE_DEFINITION_KIND_QUERY"
  | .behavior => "CASE_DEFINITION_KIND_BEHAVIOR"
  | .target => "CASE_DEFINITION_KIND_TARGET"
  | .compiler => "CASE_DEFINITION_KIND_COMPILER"
  | .provider => "CASE_DEFINITION_KIND_PROVIDER"
  | .law => "CASE_DEFINITION_KIND_LAW"
  | .connector => "CASE_DEFINITION_KIND_CONNECTOR"
  | .kernel => "CASE_DEFINITION_KIND_KERNEL"
  | .experimentSpace => "CASE_DEFINITION_KIND_EXPERIMENT_SPACE"
  | .variationAxis => "CASE_DEFINITION_KIND_VARIATION_AXIS"
  | .choice => "CASE_DEFINITION_KIND_CHOICE"
  | .fault => "CASE_DEFINITION_KIND_FAULT"
  | .coverageGoal => "CASE_DEFINITION_KIND_COVERAGE_GOAL"

private def gapKind : CaseKnownGapKind → String
  | .capabilityContract => "CASE_KNOWN_GAP_KIND_CAPABILITY_CONTRACT"
  | .input => "CASE_KNOWN_GAP_KIND_INPUT"
  | .interpretation => "CASE_KNOWN_GAP_KIND_INTERPRETATION"
  | .claim => "CASE_KNOWN_GAP_KIND_CLAIM"

private def sourceLocation (source : Umpire.SourceLocation) : CanonicalJson := .object [
  ("path", .string source.path),
  ("line", .string (toString source.line)),
  ("column", .string (toString source.column)),
  ("provenance", .string source.provenance)
]

/-- Encode the exact deterministic Umpire payload carried opaquely by Testpilot provenance. -/
def producerData (metadata : CaseMetadata) : ByteArray := (CanonicalJson.object [
  ("definitions", .array (metadata.definitions.map fun definition => .object [
    ("definitionId", .string definition.definitionId),
    ("behaviorFingerprint", .string definition.behaviorFingerprint),
    ("kind", .string (definitionKind definition.kind))
  ])),
  ("sources", .array (metadata.sources.map sourceLocation)),
  ("knownGaps", .array (metadata.knownGaps.map fun gap => .object ([
    ("kind", .string (gapKind gap.kind)),
    ("code", .string gap.code)
  ] ++ gap.subject.toList.map (fun subject => ("subject", .string subject)) ++
    gap.detail.toList.map (fun detail => ("detail", .string detail)))))
]).prettyBytes.toUTF8

/-- Construct generated Testpilot provenance while leaving its payload entirely Umpire-owned. -/
def make (metadata : CaseMetadata) : temporal.server.api.testpilot.v1.CaseProvenance :=
  Testpilot.Authoring.provenance metadata.producerId metadata.producerVersion
    (producerData metadata)

end Umpire.Case.Provenance
