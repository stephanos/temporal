import Testpilot.Authoring
import Umpire.Provenance
import Umpire.Case.Coverage
import Umpire.Case.LocalNames
import Umpire.KnownGap

/-!
Umpire producer assembly for generated Testpilot Cases.

Producers lower their checked semantics into generated Program and Contract values before this
boundary. A checked field Property's monitor rule arrives as `.monitor`, derived by
`Umpire.Case.Projection.lower` together with the request coverage it implies, and a correlated
Property arrives as `.correlated` from `Umpire.Case.Correlated.lower`; the compiler admits both side
by side. The compiler validates source-bound property rows, admits the requested whole-Case field
and clause coverage, preserves unsupported-lowering diagnostics, renames the Program and Contract to
Case-local names and spellings (`Umpire.Case.LocalNames`), attaches exact Umpire provenance rows,
and returns the generated Case without introducing a parallel protocol representation.
-/

namespace Umpire

/-- Convert one checked planning Known Gap to the exact Umpire provenance row vocabulary. -/
def KnownGap.toProvenanceGap (gap : KnownGap) : Provenance.KnownGap := {
  kind := match gap.kind with
    | .capability => .capability
    | .input => .input
    | .interpretation => .interpretation
    | .claim => .claim
  code := gap.code.value
  subject := gap.subject.map DefinitionId.value
  detail := gap.detail
}

/-- Convert checked planning Known Gaps to exact provenance rows in one order-preserving pass. -/
def KnownGapSet.toProvenanceGaps (gaps : KnownGapSet) : List Provenance.KnownGap :=
  gaps.toList.map KnownGap.toProvenanceGap

end Umpire

namespace Umpire.Case.Compiler

open Umpire
open Umpire.Case
open temporal.server.api.testpilot.v1

/-- A stable failure for a checked construct outside the generated Testpilot vocabulary. -/
structure Error where
  sourceDefinitionId : String
  source : SourceLocation
  construct : String
  deriving BEq, DecidableEq, Repr

/-- One checked property already lowered to a generated rule, or rejected with its source. -/
inductive ContractLowering where
  | monitor (sourceDefinition : Provenance.DefinitionBinding) (rule : ContractRule)
  | correlated (sourceDefinition : Provenance.DefinitionBinding) (capability : CorrelatedContract)
      (rules : List Provenance.CorrelatedRuleBinding)
  | unsupported
      (sourceDefinition : Provenance.DefinitionBinding)
      (source : SourceLocation)
      (construct : String)

/-- Complete checked producer input whose wire-shaped fields are generated protocol values. -/
structure Input where
  version : FormatVersion
  caseId : String
  producerId : String
  producerVersion : String := ""
  definitions : List Provenance.DefinitionBinding
  sources : List SourceLocation
  knownGaps : List Provenance.KnownGap
  program : Program
  contractId : String
  properties : List ContractLowering
  /-- The modeled input fields and requested clauses this Case must cover. A Case that requests
  none keeps its exact existing meaning. -/
  coverage : Coverage.Request := {}

private def lowerProperty : ContractLowering → Except Error (Option ContractRule)
  | .monitor sourceDefinition rule =>
      if sourceDefinition.kind != .property then
        .error {
          sourceDefinitionId := sourceDefinition.definitionId
          source := { path := "" }
          construct := "property.definition-kind"
        }
      else
        .ok (some rule)
  | .correlated sourceDefinition _ _ =>
      if sourceDefinition.kind != .property then
        .error {
          sourceDefinitionId := sourceDefinition.definitionId
          source := { path := "" }
          construct := "property.definition-kind" }
      else .ok none
  | .unsupported sourceDefinition source construct =>
      .error { sourceDefinitionId := sourceDefinition.definitionId, source, construct }

/-- A Case-local renaming that would merge two Definition IDs or two encodings, naming both. -/
private def localNameError : LocalNames.Error → Error
  | .sharedName localName first second =>
      Error.mk first { path := "" } s!"local-name {localName} names {first} and {second}"
  | .splitDefinition definitionId first second =>
      Error.mk definitionId { path := "" } s!"local-name {definitionId} is named {first} and {second}"
  | .ambiguousSpelling definitionId spelling first second =>
      Error.mk definitionId { path := "" }
        s!"model-value-spelling {spelling} of {definitionId} spells {first} and {second}"

/-- Assemble generated values into a Case while preserving Umpire provenance and typed rejection. -/
def compile (input : Input) : Except Error temporal.server.api.testpilot.v1.Case := do
  let lowered ← input.properties.mapM lowerProperty
  let rules := lowered.filterMap id
  let correlatedProperties := input.properties.filterMap fun property => match property with
    | .correlated binding capability bindings => some (binding, capability, bindings)
    | _ => none
  if correlatedProperties.length > 1 then
    throw {
      sourceDefinitionId := input.contractId
      source := { path := "" }
      construct := "multiple correlated projections" }
  let capability := correlatedProperties.head?.map (·.2.1)
  let correlatedRules := correlatedProperties.flatMap (·.2.2)
  -- A requested mapping that this Case does not construct or lower rejects the whole Case here,
  -- before any Program or Contract could reach a Driver.
  (Coverage.check input.program correlatedRules input.coverage input.caseId).mapError fun failure =>
    Error.mk failure.subject { path := "" } failure.reason
  let localized ← (LocalNames.localize input.program rules.toArray capability).mapError
    localNameError
  let metadata : Provenance.Metadata := {
    producerId := input.producerId
    producerVersion := input.producerVersion
    definitions := input.definitions
    sources := input.sources
    knownGaps := input.knownGaps
    correlatedRules
    localNames := localized.localNames
    modelValueFingerprints := localized.modelValueFingerprints
  }
  let provenance ← (Provenance.make metadata).mapError fun source =>
    Error.mk input.caseId source "provenance.source-position"
  pure (Testpilot.Authoring.case input.version.major input.caseId localized.program
    (Testpilot.Authoring.Contract.contract input.contractId localized.rules localized.capability)
    provenance input.version.minor)

end Umpire.Case.Compiler
