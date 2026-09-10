import Testpilot.Authoring
import Umpire.Provenance
import Umpire.Case.Coverage
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
  | monitor (sourceDefinition : Provenance.DefinitionBinding) (rule : ContractRuleDefinition)
  | correlated (sourceDefinition : Provenance.DefinitionBinding) (capability : CorrelatedContract)
      (clauses : List Provenance.CorrelatedRuleBinding)
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
  contractLimits : ContractLimits
  /-- The modeled input fields and requested clauses this Case must cover. A Case that requests
  none keeps its exact existing meaning. -/
  coverage : Coverage.Request := {}

private def lowerProperty : ContractLowering → Except Error (Option ContractRuleDefinition)
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

/-- Assemble generated values into a Case while preserving Umpire provenance and typed rejection. -/
def compile (input : Input) : Except Error temporal.server.api.testpilot.v1.Case := do
  let lowered ← input.properties.mapM lowerProperty
  let rules := lowered.filterMap id
  let correlatedProperties := input.properties.filterMap fun property => match property with
    | .correlated binding capability clauses => some (binding, capability, clauses)
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
  let metadata : Provenance.Metadata := {
    producerId := input.producerId
    producerVersion := input.producerVersion
    definitions := input.definitions
    sources := input.sources
    knownGaps := input.knownGaps
    correlatedRules
  }
  pure (Testpilot.Authoring.case input.version.major input.caseId input.program
    { Testpilot.Authoring.Contract.contract input.contractId rules.toArray input.contractLimits with
      «correlated» := capability }
    (Provenance.make metadata) input.version.minor)

end Umpire.Case.Compiler
