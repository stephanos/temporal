import Testpilot.Authoring
import Umpire.Core

/-!
Producer provenance carried inside a generated Testpilot Case.

The protobuf schema owns the Case, Program, Contract, and provenance row structures. Umpire retains
its producer-specific definitions, fingerprints, sources, Known Gaps, and Correlated Rule bindings
here and lowers them into the typed `CaseProvenance` rows, keeping the order the Producer lists
them. `Umpire.Case.Compiler` adds the Case-local name and model value fingerprint rows its
localization derives. Testpilot never reads the rows.
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
  ruleId : String
  propertyId : String
  propertyFingerprint : String
  projectionId : String
  projectionFingerprint : String
  source : SourceLocation
  deriving BEq, Repr

/-- One Case-local name the Program and Contract use, and the Definition ID it stands for. -/
structure LocalName where
  localName : String
  definitionId : String
  deriving BEq, DecidableEq, Repr

/-- One model value the Case spells differently from its canonical encoding: the local name of its
definition, its Case spelling, and the lowercase hexadecimal SHA-256 of the encoding. -/
structure ModelValueFingerprint where
  localName : String
  spelling : String
  fingerprint : String
  deriving BEq, DecidableEq, Repr

/-- One class of one action's input field a Case realized through the example its Model wrote for
it. A class with an example stands for several concrete values the Model does not count, so the
Case's Verdict is a claim about the class made through one of them, and the row says which. -/
structure AbstractionClaimRow where
  action : String
  field : String
  className : String
  exampleValue : String
  deriving BEq, DecidableEq, Repr

/-- Compiler and source provenance for one Case artifact. -/
structure Metadata where
  producerId : String
  producerVersion : String := ""
  definitions : List DefinitionBinding := []
  sources : List SourceLocation := []
  knownGaps : List KnownGap := []
  correlatedRules : List CorrelatedRuleBinding := []
  localNames : List LocalName := []
  modelValueFingerprints : List ModelValueFingerprint := []
  abstractionClaims : List AbstractionClaimRow := []
  deriving BEq, Repr

open temporal.server.api.testpilot.v1 (CaseProvenance)

private def definitionKind : DefinitionKind → temporal.server.api.testpilot.v1.DefinitionKind
  | .setup => .DEFINITION_KIND_SETUP
  | .state => .DEFINITION_KIND_STATE
  | .action => .DEFINITION_KIND_ACTION
  | .outcome => .DEFINITION_KIND_OUTCOME
  | .fact => .DEFINITION_KIND_FACT
  | .relation => .DEFINITION_KIND_RELATION
  | .capability => .DEFINITION_KIND_CAPABILITY
  | .property => .DEFINITION_KIND_PROPERTY
  | .query => .DEFINITION_KIND_QUERY
  | .scenario => .DEFINITION_KIND_SCENARIO
  | .target => .DEFINITION_KIND_TARGET
  | .compiler => .DEFINITION_KIND_COMPILER
  | .provider => .DEFINITION_KIND_PROVIDER
  | .law => .DEFINITION_KIND_LAW
  | .connector => .DEFINITION_KIND_CONNECTOR
  | .machine => .DEFINITION_KIND_MACHINE

private def gapKind : KnownGapKind → temporal.server.api.testpilot.v1.KnownGapKind
  | .capability => .KNOWN_GAP_KIND_CAPABILITY
  | .input => .KNOWN_GAP_KIND_INPUT
  | .interpretation => .KNOWN_GAP_KIND_INTERPRETATION
  | .claim => .KNOWN_GAP_KIND_CLAIM

/-- A source position narrowed to the protocol's `int32` line and column, or the source itself when
either does not fit, so no position wraps into a different one. -/
private def sourceLocation (source : Umpire.SourceLocation) :
    Except Umpire.SourceLocation temporal.server.api.testpilot.v1.SourceLocation :=
  if source.line ≤ 2147483647 && source.column ≤ 2147483647 then
    .ok { path := source.path, line := Int32.ofNat source.line,
          column := Int32.ofNat source.column, provenance := source.provenance }
  else
    .error source

/-- Lower the metadata into the typed provenance rows a generated Case carries, each list in the
order the Producer gave it. Testpilot never reads the rows. A source whose line or column does not
fit the protocol is returned as the error. -/
def make (metadata : Metadata) : Except Umpire.SourceLocation CaseProvenance := do
  let sources ← metadata.sources.mapM sourceLocation
  let correlatedRules ← metadata.correlatedRules.mapM fun rule => do
    let source ← sourceLocation rule.source
    pure ({ rule_id := rule.ruleId, property_id := rule.propertyId,
            property_fingerprint := rule.propertyFingerprint, projection_id := rule.projectionId,
            projection_fingerprint := rule.projectionFingerprint, source := some source } :
      temporal.server.api.testpilot.v1.CorrelatedRuleBinding)
  pure (Testpilot.Authoring.provenance metadata.producerId metadata.producerVersion
    (metadata.definitions.map fun definition =>
      { definition_id := definition.definitionId,
        behavior_fingerprint := definition.behaviorFingerprint,
        kind := definitionKind definition.kind }).toArray
    sources.toArray
    (metadata.knownGaps.map fun gap =>
      Testpilot.Authoring.knownGap (gapKind gap.kind) gap.code gap.subject gap.detail).toArray
    correlatedRules.toArray
    (metadata.localNames.map fun name =>
      ({ local_name := name.localName, definition_id := name.definitionId } :
        temporal.server.api.testpilot.v1.LocalName)).toArray
    (metadata.modelValueFingerprints.map fun value =>
      ({ local_name := value.localName, spelling := value.spelling,
         fingerprint := value.fingerprint } :
        temporal.server.api.testpilot.v1.ModelValueFingerprint)).toArray
    (metadata.abstractionClaims.map fun claim =>
      Testpilot.Authoring.abstractionClaim claim.action claim.field claim.className
        claim.exampleValue).toArray)

end Umpire.Provenance
