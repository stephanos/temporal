import Testpilot.Authoring
import Umpire.Core
import Umpire.Json

/-!
Producer-owned identity bytes carried inside a generated Testpilot Case.

The protobuf schema owns the Case, Program, and Contract structures. Umpire retains only its
producer-specific definitions, fingerprints, sources, and Known Gaps, and encodes them into opaque
Testpilot provenance. Testpilot never reads the payload, so the field names, enum spellings, list
order, optional-field elision, indentation, and terminal newline below are the whole producer
contract.
-/

namespace Umpire.Provenance

/-- The closed categories of source-model incompleteness retained in Umpire provenance. -/
inductive KnownGapKind where
  | capability
  | input
  | interpretation
  | claim
  deriving BEq, DecidableEq, Repr

/-- The source definition classes retained in Case provenance. -/
inductive DefinitionKind where
  | setup
  | state
  | action
  | outcome
  | fact
  | relation
  | capability
  | property
  | query
  | scenario
  | target
  | compiler
  | provider
  | law
  | connector
  | machine
  deriving BEq, DecidableEq, Repr

/-- One source Definition ID and the behavior fingerprint used for this Case. -/
structure DefinitionBinding where
  definitionId : String
  behaviorFingerprint : String
  kind : DefinitionKind
  deriving BEq, DecidableEq, Repr

/-- One explicit coverage or portability gap retained by the compiler. -/
structure KnownGap where
  kind : KnownGapKind
  code : String
  subject : Option String := none
  detail : Option String := none
  deriving BEq, DecidableEq, Repr

/-- Exact source binding for one compiled correlated rule. -/
structure CorrelatedRuleBinding where
  clauseId : String
  propertyId : String
  propertyFingerprint : String
  projectionId : String
  projectionFingerprint : String
  source : SourceLocation
  deriving BEq, Repr

/-- Compiler and source provenance for one Case artifact. -/
structure Metadata where
  producerId : String
  producerVersion : String := ""
  definitions : List DefinitionBinding := []
  sources : List SourceLocation := []
  knownGaps : List KnownGap := []
  correlatedRules : List CorrelatedRuleBinding := []
  deriving BEq, Repr

open Umpire.CanonicalJson

private def definitionKind : DefinitionKind → String
  | .setup => "CASE_DEFINITION_KIND_SETUP"
  | .state => "CASE_DEFINITION_KIND_STATE"
  | .action => "CASE_DEFINITION_KIND_ACTION"
  | .outcome => "CASE_DEFINITION_KIND_OUTCOME"
  | .fact => "CASE_DEFINITION_KIND_FACT"
  | .relation => "CASE_DEFINITION_KIND_RELATION"
  | .capability => "CASE_DEFINITION_KIND_CAPABILITY"
  | .property => "CASE_DEFINITION_KIND_PROPERTY"
  | .query => "CASE_DEFINITION_KIND_QUERY"
  | .scenario => "CASE_DEFINITION_KIND_SCENARIO"
  | .target => "CASE_DEFINITION_KIND_TARGET"
  | .compiler => "CASE_DEFINITION_KIND_COMPILER"
  | .provider => "CASE_DEFINITION_KIND_PROVIDER"
  | .law => "CASE_DEFINITION_KIND_LAW"
  | .connector => "CASE_DEFINITION_KIND_CONNECTOR"
  | .machine => "CASE_DEFINITION_KIND_MACHINE"

private def gapKind : KnownGapKind → String
  | .capability => "CASE_KNOWN_GAP_KIND_CAPABILITY"
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
def producerData (metadata : Metadata) : ByteArray := (CanonicalJson.object ([
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
 ] ++ if metadata.correlatedRules.isEmpty then [] else [
  ("correlatedRules", .array (metadata.correlatedRules.map fun rule => .object [
    ("clauseId", .string rule.clauseId),
    ("propertyId", .string rule.propertyId),
    ("propertyFingerprint", .string rule.propertyFingerprint),
    ("projectionId", .string rule.projectionId),
    ("projectionFingerprint", .string rule.projectionFingerprint),
    ("source", sourceLocation rule.source)
  ]))
 ])).prettyBytes.toUTF8

/-- Construct generated Testpilot provenance while leaving its payload entirely Umpire-owned. -/
def make (metadata : Metadata) : temporal.server.api.testpilot.v1.CaseProvenance :=
  Testpilot.Authoring.provenance metadata.producerId metadata.producerVersion
    (producerData metadata)

end Umpire.Provenance
