import Umpire.Case.Compiler
import Umpire.Case.ScopedTests

namespace Umpire.Case.CompilerTests

open Umpire
open Umpire.Case
open Umpire.Case.Compiler
open Testpilot.Authoring
open temporal.server.api.testpilot.v1

private def property : CaseDefinitionBinding := {
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

private def rule (id : String := "example.rule") : ContractRuleDefinition :=
  Monitor.rule id .CONTRACT_RULE_KIND_SAFETY "satisfied"
    #[Monitor.state "satisfied" .CONTRACT_STATE_STATUS_SATISFIED] #[]

private def program : Program :=
  Program.make "example.program" #[] #[] #[] #[] (Program.cleanup "cleanup" #[])
    (Program.limits 1 1 1 1 1 16 4 1 1024 1024 1000 100)

private def contractLimits : ContractLimits :=
  Monitor.limits 2 2 1 4 4 64 1 64

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
  contractLimits
}

private def planningGaps : KnownGapSet :=
  (KnownGapSet.checkCanonical [
    { kind := .capabilityContract, code := DefinitionId.of "example.gap.capability" },
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
  input with knownGaps := planningGaps.toCaseKnownGaps
}

private def expectedProducerData := String.intercalate "\n" [
  "{",
  "  \"definitions\": [",
  "    {",
  "      \"definitionId\": \"example.property\",",
  "      \"behaviorFingerprint\": \"example-property/v1\",",
  "      \"kind\": \"CASE_DEFINITION_KIND_PROPERTY\"",
  "    }",
  "  ],",
  "  \"sources\": [",
  "    {",
  "      \"path\": \"Example/Case.lean\",",
  "      \"line\": \"11\",",
  "      \"column\": \"3\",",
  "      \"provenance\": \"checked-model\"",
  "    }",
  "  ],",
  "  \"knownGaps\": [",
  "    {",
  "      \"kind\": \"CASE_KNOWN_GAP_KIND_CAPABILITY_CONTRACT\",",
  "      \"code\": \"example.gap.capability\"",
  "    },",
  "    {",
  "      \"kind\": \"CASE_KNOWN_GAP_KIND_INPUT\",",
  "      \"code\": \"example.gap.input\",",
  "      \"subject\": \"example.target\"",
  "    },",
  "    {",
  "      \"kind\": \"CASE_KNOWN_GAP_KIND_INTERPRETATION\",",
  "      \"code\": \"example.gap.interpretation\",",
  "      \"detail\": \"Interpretation remains model-owned.\"",
  "    },",
  "    {",
  "      \"kind\": \"CASE_KNOWN_GAP_KIND_CLAIM\",",
  "      \"code\": \"example.gap.claim\",",
  "      \"subject\": \"example.property\",",
  "      \"detail\": \"Claim requires runtime evidence.\"",
  "    }",
  "  ]",
  "}"
] ++ "\n"

/-! The single checked conversion pass retains every row field in the compiled Case. -/
#guard match compile inputWithPlanningGaps with
  | .ok output => output.provenance.map (·.producer_data) == some expectedProducerData.toUTF8
  | .error _ => false

#guard match compile { input with
    version := { major := 1, minor := 2 }
    properties := [.monitor property (rule "first"), .monitor property (rule "second")]
  } with
  | .ok output =>
      output.case_id == input.caseId &&
      output.version.map (fun version => (version.major, version.minor)) == some (1, 2) &&
      output.program.map (·.program_id) == some input.program.program_id &&
      output.contract.map (fun contract => contract.rules.map (·.rule_id)) ==
        some #["first", "second"] &&
      (output.contract.bind (·.limits) |>.map (·.max_rules)) ==
        some input.contractLimits.max_rules
  | .error _ => false

private def rejectsAs (sourceDefinitionId : String) (expectedSource : SourceLocation)
    (construct : String) (result : Except LoweringError temporal.server.api.testpilot.v1.Case) : Bool :=
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
