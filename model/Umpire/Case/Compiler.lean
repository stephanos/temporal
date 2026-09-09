import Testpilot.Authoring
import Umpire.Case
import Umpire.Case.Coverage
import Umpire.Case.Provenance
import Umpire.KnownGap

/-!
Umpire producer assembly for generated Testpilot Cases.

Producers lower their checked semantics into generated Program and Contract values before this
boundary. The compiler validates source-bound property rows, admits the requested whole-Case field
and clause coverage, preserves unsupported-lowering diagnostics, attaches exact Umpire provenance,
and returns the generated Case without introducing a parallel protocol representation.
-/

namespace Umpire

/-- Convert one checked planning Known Gap to the exact Umpire provenance row vocabulary. -/
def KnownGap.toCaseKnownGap (gap : KnownGap) : Case.CaseKnownGap := {
  kind := match gap.kind with
    | .capabilityContract => .capabilityContract
    | .input => .input
    | .interpretation => .interpretation
    | .claim => .claim
  code := gap.code.value
  subject := gap.subject.map DefinitionId.value
  detail := gap.detail
}

/-- Convert checked planning Known Gaps to exact provenance rows in one order-preserving pass. -/
def KnownGapSet.toCaseKnownGaps (gaps : KnownGapSet) : List Case.CaseKnownGap :=
  gaps.toList.map KnownGap.toCaseKnownGap

end Umpire

namespace Umpire.Case.Compiler

open Umpire
open Umpire.Case
open temporal.server.api.testpilot.v1

/-- A stable failure for a checked construct outside the generated Testpilot vocabulary. -/
structure LoweringError where
  sourceDefinitionId : String
  source : SourceLocation
  construct : String
  deriving BEq, DecidableEq, Repr

/-- One checked property already lowered to a generated rule, or rejected with its source. -/
inductive ContractLowering where
  | monitor (sourceDefinition : CaseDefinitionBinding) (rule : ContractRuleDefinition)
  | scoped (sourceDefinition : CaseDefinitionBinding) (capability : ScopedContract)
      (clauses : List CaseScopedClauseBinding)
  | unsupported
      (sourceDefinition : CaseDefinitionBinding)
      (source : SourceLocation)
      (construct : String)

/-- Complete checked producer input whose wire-shaped fields are generated protocol values. -/
structure Input where
  version : FormatVersion
  caseId : String
  producerId : String
  producerVersion : String := ""
  definitions : List CaseDefinitionBinding
  sources : List SourceLocation
  knownGaps : List CaseKnownGap
  program : Program
  contractId : String
  properties : List ContractLowering
  contractLimits : ContractLimits
  /-- The modeled input fields and requested clauses this Case must cover. A Case that requests
  none keeps its exact existing meaning. -/
  coverage : Coverage.Request := {}

private def lowerProperty : ContractLowering → Except LoweringError (Option ContractRuleDefinition)
  | .monitor sourceDefinition rule =>
      if sourceDefinition.kind != .property then
        .error {
          sourceDefinitionId := sourceDefinition.definitionId
          source := { path := "" }
          construct := "property.definition-kind"
        }
      else
        .ok (some rule)
  | .scoped sourceDefinition _ _ =>
      if sourceDefinition.kind != .property then
        .error {
          sourceDefinitionId := sourceDefinition.definitionId
          source := { path := "" }
          construct := "property.definition-kind" }
      else .ok none
  | .unsupported sourceDefinition source construct =>
      .error { sourceDefinitionId := sourceDefinition.definitionId, source, construct }

/-- Assemble generated values into a Case while preserving Umpire provenance and typed rejection. -/
def compile (input : Input) : Except LoweringError temporal.server.api.testpilot.v1.Case := do
  let lowered ← input.properties.mapM lowerProperty
  let rules := lowered.filterMap id
  let scopedProperties := input.properties.filterMap fun property => match property with
    | .scoped binding capability clauses => some (binding, capability, clauses)
    | _ => none
  if scopedProperties.length > 1 then
    throw {
      sourceDefinitionId := input.contractId
      source := { path := "" }
      construct := "multiple scoped projections" }
  let capability := scopedProperties.head?.map (·.2.1)
  let scopedClauses := scopedProperties.flatMap (·.2.2)
  -- A requested mapping that this Case does not construct or lower rejects the whole Case here,
  -- before any Program or Contract could reach a Driver.
  (Coverage.check input.program scopedClauses input.coverage input.caseId).mapError fun failure =>
    LoweringError.mk failure.subject { path := "" } failure.reason
  let metadata : CaseMetadata := {
    producerId := input.producerId
    producerVersion := input.producerVersion
    definitions := input.definitions
    sources := input.sources
    knownGaps := input.knownGaps
    scopedClauses
  }
  pure (Testpilot.Authoring.case input.version.major input.caseId input.program
    { Testpilot.Authoring.Monitor.contract input.contractId rules.toArray input.contractLimits with
      «scoped» := capability }
    (Provenance.make metadata) input.version.minor)

end Umpire.Case.Compiler
