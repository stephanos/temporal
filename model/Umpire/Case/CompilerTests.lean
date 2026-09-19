import Umpire.Case.Compiler
import Umpire.Case.CorrelatedTests

namespace Umpire.Case.CompilerTests

open Umpire
open Umpire.Case
open Umpire.Case.Compiler
open Testpilot.Authoring
open temporal.server.api.testpilot.v1

private def property : Provenance.DefinitionBinding := {
  definitionId := "example.property"
  behaviorFingerprint := "example-property/v1"
  kind := .property
}

private def source : SourceLocation := {
  path := "Example/Case.lean"
  line := 11
  column := 3
  provenance := "checked-model"
}

private def rule (id : String := "example.rule") : ContractRule :=
  Contract.rule id .CONTRACT_RULE_KIND_SAFETY "satisfied"
    #[Contract.state "satisfied" .CONTRACT_STATE_STATUS_SATISFIED] #[]

private def program : Program :=
  Program.make "example.program" #[] #[] #[] #[] (Program.cleanup "cleanup" #[])

private def input : Input := {
  version := { major := 1 }
  caseId := "example.case"
  producerId := "umpire.case.compiler"
  definitions := [property]
  sources := [source]
  knownGaps := []
  program
  contractId := "example.contract"
  properties := [.monitor property rule]
}

private def planningGaps : KnownGapSet :=
  (KnownGapSet.checkCanonical [
    { kind := .capability, code := DefinitionId.of "example.gap.capability" },
    {
      kind := .input
      code := DefinitionId.of "example.gap.input"
      subject := some (DefinitionId.of "example.target")
    },
    {
      kind := .interpretation
      code := DefinitionId.of "example.gap.interpretation"
      detail := some "Interpretation remains model-owned."
    },
    {
      kind := .claim
      code := DefinitionId.of "example.gap.claim"
      subject := some (DefinitionId.of "example.property")
      detail := some "Claim requires runtime evidence."
    }
  ]).toOption.get (by native_decide)

private def inputWithPlanningGaps : Input := {
  input with knownGaps := planningGaps.toProvenanceGaps
}

private def sourceRow (source : temporal.server.api.testpilot.v1.SourceLocation) :=
  (source.path, source.line, source.column, source.provenance)

private def gapRow (gap : temporal.server.api.testpilot.v1.KnownGap) :=
  (gap.kind, gap.code, gap.subject_presence.map (fun | .subject subject => subject),
    gap.detail_presence.map (fun | .detail detail => detail))

/-- Every field of every provenance row, so the comparison pins each one. -/
private def provenanceRows (provenance : CaseProvenance) :=
  (provenance.definitions.toList.map (fun definition =>
      (definition.definition_id, definition.behavior_fingerprint, definition.kind)),
   provenance.sources.toList.map sourceRow,
   provenance.known_gaps.toList.map gapRow)

/-- The rows the input lowers to, in the input's order. -/
private def expectedProvenance :=
  ([("example.property", "example-property/v1", DefinitionKind.DEFINITION_KIND_PROPERTY)],
   [("Example/Case.lean", (11 : Int32), (3 : Int32), "checked-model")],
   [(KnownGapKind.KNOWN_GAP_KIND_CAPABILITY, "example.gap.capability", (none : Option String),
       (none : Option String)),
    (.KNOWN_GAP_KIND_INPUT, "example.gap.input", some "example.target", none),
    (.KNOWN_GAP_KIND_INTERPRETATION, "example.gap.interpretation", none,
       some "Interpretation remains model-owned."),
    (.KNOWN_GAP_KIND_CLAIM, "example.gap.claim", some "example.property",
       some "Claim requires runtime evidence.")])

/-! The single checked conversion pass retains every row field in the compiled Case. -/
#guard match compile inputWithPlanningGaps with
  | .ok output => output.provenance.map provenanceRows == some expectedProvenance
  | .error _ => false

-- A source position lowers to the protocol's int32 line and column only when it fits; the next
-- line rejects with the source rather than wrapping into another position.
#guard match compile { input with sources := [{ source with line := 2147483647, column := 2147483647 }] } with
  | .ok output => output.provenance.map (·.sources.toList.map sourceRow) ==
      some [("Example/Case.lean", (2147483647 : Int32), (2147483647 : Int32), "checked-model")]
  | .error _ => false

#guard match compile { input with sources := [{ source with line := 2147483648 }] } with
  | .error error => error.construct == "provenance.source-position" && error.source.line == 2147483648
  | .ok _ => false

#guard match compile { input with sources := [{ source with column := 2147483648 }] } with
  | .error error => error.construct == "provenance.source-position" && error.source.column == 2147483648
  | .ok _ => false

#guard match compile { input with
    version := { major := 1, minor := 2 }
    properties := [.monitor property (rule "first"), .monitor property (rule "second")]
  } with
  | .ok output =>
      output.case_id == input.caseId &&
      output.version.map (fun version => (version.major, version.minor)) == some (1, 2) &&
      output.program.map (·.program_id) == some input.program.program_id &&
      output.contract.map (fun contract => contract.rules.map (·.rule_id)) ==
        some #["first", "second"]
  | .error _ => false

private def rejectsAs (sourceDefinitionId : String) (expectedSource : SourceLocation)
    (construct : String) (result : Except Error temporal.server.api.testpilot.v1.Case) : Bool :=
  match result with
  | .error failure =>
      failure.sourceDefinitionId == sourceDefinitionId && failure.source == expectedSource &&
        failure.construct == construct
  | .ok _ => false

#guard rejectsAs property.definitionId { path := "" } "property.definition-kind"
  (compile { input with properties := [.monitor { property with kind := .query } rule] })

private def unsupported := ContractLowering.unsupported
  property source "property.temporal-unbounded"

private def unsupportedGuardedTemporal := ContractLowering.unsupported
  property source "property.guarded-eventually-within"

#guard rejectsAs property.definitionId source "property.temporal-unbounded"
  (compile { input with properties := [unsupported] })

/- The current lowering boundary preserves a guarded temporal rejection as checked source data. -/
#guard rejectsAs property.definitionId source "property.guarded-eventually-within"
  (compile { input with properties := [unsupportedGuardedTemporal] })

end Umpire.Case.CompilerTests
