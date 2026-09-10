import Testpilot.Authoring
import Testpilot.ProtoJSON

open temporal.server.api.testpilot.v1
open Testpilot.Authoring

namespace Testpilot.Tests.ProtoJSON

private def assert (condition : Bool) (failure : String) : IO Unit := do
  unless condition do throw (IO.userError failure)

private def limits := Program.limits 1 4 3 2 9223372036854775807 16 8 4 4096 8192 10000 1000
private def instructionLimits := Program.instructionLimits 1000 1 2 4096
private def runPath := Path.make #[Path.field "run_id"]

private def programExpr := ProgramExpr.all #[
  ProgramExpr.present (ProgramExpr.path ProgramExpr.run runPath),
  ProgramExpr.equals (ProgramExpr.literal (Value.bytes (ByteArray.mk #[0, 255])))
    (ProgramExpr.literal (Value.bytes (ByteArray.mk #[0, 255]))),
  ProgramExpr.compare .COMPARISON_OPERATOR_LESS_THAN
    (ProgramExpr.literal (Value.signedInteger (-9223372036854775808)))
    (ProgramExpr.literal (Value.unsignedInteger 18446744073709551615))
]

private def contractExpr := ContractExpr.any #[
  ContractExpr.present (ContractExpr.observation "result"),
  ContractExpr.equals (ContractExpr.runEvent .RUN_EVENT_FIELD_RUN_ID)
    (ContractExpr.literal (Value.text "run")),
  ContractExpr.equals (ContractExpr.capture "captured")
    (ContractExpr.literal (Value.messageValue {
      type_url := "type.googleapis.com/temporal.server.api.testpilot.v1.FormatVersion",
      value := ByteArray.mk #[8, 1, 16, 2]
    })),
  ContractExpr.literal (Value.floatingPoint 1.5),
  ContractExpr.literal (Value.enumeration 1),
  ContractExpr.literal (Value.boolean false)
]

private def literalProgram : temporal.server.api.testpilot.v1.Program := Program.make "program"
  #[Program.role "endpoint" .ROLE_KIND_ENDPOINT]
  #[Program.valueSlot "slot" (Types.singular (Types.scalar .SCALAR_KIND_TEXT))]
  #[Program.observation "result" (Types.singular Types.any)]
  #[Program.controller "controller" #[Program.node "finish"
    (Program.finish programExpr) instructionLimits (guard := some programExpr)]]
  (Program.cleanup "cleanup" #[])
  limits

private def contract : Contract := Contract.contract "contract" #[
  Contract.rule "rule" .CONTRACT_RULE_KIND_BOUNDED_LIVENESS "open"
    #[Contract.state "open" .CONTRACT_STATE_STATUS_NONTERMINAL,
      Contract.state "done" .CONTRACT_STATE_STATUS_SATISFIED,
      Contract.state "late" .CONTRACT_STATE_STATUS_VIOLATED]
    #[Contract.transition "complete" "open" "done" #[.RUN_EVENT_KIND_RUN_CLOSED]
      contractExpr .CONTRACT_SUPPORT_KIND_MATCHING_EVENT
      #[Contract.captureAssignment "captured" "result"]]
    (deadline := some (Contract.deadline 9223372036854775807 "late"))
    (captures := #[Contract.capture "captured" (Contract.messageCapture
      "temporal.server.api.testpilot.v1.FormatVersion")])
] (Contract.limits 1 3 1 16 32 64 1 1024)

def literalCase : Case := Testpilot.Authoring.case 1 "case" literalProgram contract
  (provenance "testpilot-tests" "1" (ByteArray.mk #[0, 255, 128]))

private def bindingProgram : temporal.server.api.testpilot.v1.Program := Program.make "program"
  #[Program.role "endpoint" .ROLE_KIND_ENDPOINT (resourceBindingId := "nexus.endpoint"),
    Program.role "worker" .ROLE_KIND_WORKER (namespaceBindingId := "namespace"),
    Program.role "queue" .ROLE_KIND_TASK_QUEUE
      (namespaceBindingId := "namespace") (resourceBindingId := "task.queue")]
  #[] #[] #[Program.controller "controller" #[Program.node "finish"
    (Program.finish (ProgramExpr.environment "")) instructionLimits]]
  (Program.cleanup "cleanup" #[]) limits
  (environment := #[Program.environment "namespace", Program.environment "task.queue",
    Program.environment "nexus.endpoint"])

def representativeCase : Case := Testpilot.Authoring.case 1 "binding-case" bindingProgram contract
  (provenance "testpilot-tests" "1" (ByteArray.mk #[0, 255, 128]))

private def unknownAnyCase : Case :=
  let expression := ProgramExpr.literal (Value.messageValue {
    type_url := "type.googleapis.com/example.Unknown"
    value := ByteArray.mk #[8, 1]
  })
  let unknownProgram := Program.make "program" #[] #[] #[]
    #[Program.controller "controller" #[Program.node "finish" (Program.finish expression)
      instructionLimits]] (Program.cleanup "cleanup" #[]) limits
  Testpilot.Authoring.case 1 "unknown-any" unknownProgram contract (provenance "test" "1")

private def malformedAnyCase : Case :=
  let expression := ProgramExpr.literal (Value.messageValue {
    type_url := "type.googleapis.com/temporal.server.api.testpilot.v1.FormatVersion"
    value := ByteArray.mk #[255]
  })
  let malformedProgram := Program.make "program" #[] #[] #[]
    #[Program.controller "controller" #[Program.node "finish" (Program.finish expression)
      instructionLimits]] (Program.cleanup "cleanup" #[]) limits
  Testpilot.Authoring.case 1 "malformed-any" malformedProgram contract (provenance "test" "1")

private def nestedExpression : Nat → ProgramExpression
  | 0 => ProgramExpr.literal (Value.boolean true)
  | depth + 1 => ProgramExpr.negation (nestedExpression depth)

private def recursionFailureCase : Case :=
  let deepProgram := Program.make "program" #[] #[] #[]
    #[Program.controller "controller" #[Program.node "finish"
      (Program.finish (nestedExpression 101)) instructionLimits]]
    (Program.cleanup "cleanup" #[]) limits
  Testpilot.Authoring.case 1 "recursion-failure" deepProgram contract (provenance "test" "1")

private def render (value : Case) : IO String := do
  match ← Testpilot.ProtoJSON.canonical value with
  | .ok text => pure text
  | .error error => throw (IO.userError (toString error))

private def tests : IO Unit := do
  let first ← render representativeCase
  let second ← render representativeCase
  assert (first == second) "equal Cases did not render deterministically"
  assert (first.contains "\"version\":{\"major\":1}") "Case 1.0 was dropped"
  assert (first.contains "\"environment\":{}") "present empty environment reference was dropped"
  assert (first.contains "\"environment\":[{\"bindingId\":\"namespace\"},{\"bindingId\":\"task.queue\"},{\"bindingId\":\"nexus.endpoint\"}]")
    "environment definitions changed order"
  assert (first.contains "\"namespaceBindingId\":\"namespace\"") "namespace binding was dropped"
  assert (first.contains "\"resourceBindingId\":\"task.queue\"") "resource binding was dropped"
  assert (first.contains "\"runEvent\":{\"field\":\"RUN_EVENT_FIELD_RUN_ID\"}")
    "Contract Run Event identity was dropped"
  assert (first.contains "\"maxAttempts\":\"9223372036854775807\"")
    "int64 upper bound was not rendered as a ProtoJSON string"
  assert (first.contains "\"elapsedMilliseconds\":\"9223372036854775807\"")
    "monitor deadline was dropped"
  assert (first.contains "AP+A") "opaque non-UTF-8 provenance bytes were not rendered"
  assert (first.contains "\"floatingPoint\":1.5") "floating value was dropped"
  assert (first.contains "\"enumValue\":{\"number\":1}") "enum value was dropped"
  assert (first.contains "\"boolValue\":false") "present false oneof value was dropped"
  assert (first.contains "\"@type\":\"type.googleapis.com/temporal.server.api.testpilot.v1.FormatVersion\"")
    "resolved Any was dropped"
  let literalFirst ← render literalCase
  let literalSecond ← render literalCase
  assert (literalFirst == literalSecond) "literal Case 1.0 encoding changed nondeterministically"
  assert (literalFirst.contains "\"version\":{\"major\":1}") "literal Case 1.0 version changed"
  assert (!literalFirst.contains "bindingId") "resource-free Case 1.0 gained binding fields"
  assert (literalFirst.contains "\"run\":{}") "Program Run identity was dropped"
  assert (literalFirst.contains "AP8=") "expression bytes were not rendered"
  match ← Testpilot.ProtoJSON.canonical unknownAnyCase with
  | .error (.protobuf (.unresolvedType _)) => pure ()
  | _ => throw (IO.userError "unknown Any type did not return unresolvedType")
  match ← Testpilot.ProtoJSON.canonical malformedAnyCase with
  | .error _ => pure ()
  | .ok _ => throw (IO.userError "malformed Any payload serialized successfully")
  match ← Testpilot.ProtoJSON.canonical recursionFailureCase with
  | .error (.protobuf (.recursionLimit _)) => pure ()
  | _ => throw (IO.userError "serialization recursion limit did not propagate")

#eval tests

end Testpilot.Tests.ProtoJSON
