import Testpilot.Authoring
import Testpilot.ProtoJSON

open temporal.server.api.testpilot.v1
open Testpilot.Authoring

namespace Testpilot.Tests.ProtoJSON

private def assert (condition : Bool) (failure : String) : IO Unit := do
  unless condition do throw (IO.userError failure)

private def instructionLimits :=
  Program.instructionLimits (some 1000) (some 9223372036854775807)
private def runPath := Path.make #[Path.field "run_id"]

private def instructionExpression := Expr.all #[
  Expr.present (Expr.path Expr.run runPath),
  Expr.equal (Expr.literal (Value.bytes (ByteArray.mk #[0, 255])))
    (Expr.literal (Value.bytes (ByteArray.mk #[0, 255]))),
  Expr.compare .COMPARISON_OPERATOR_LESS_THAN
    (Expr.literal (Value.signedInteger (-9223372036854775808)))
    (Expr.literal (Value.unsignedInteger 18446744073709551615))
]

private def predicateExpression := Expr.any #[
  Expr.present (Expr.observation "result"),
  Expr.equal (Expr.runEvent .RUN_EVENT_FIELD_RUN_ID)
    (Expr.literal (Value.text "run")),
  Expr.equal (Expr.capture "captured")
    (Expr.literal (Value.messageValue {
      type_url := "type.googleapis.com/temporal.server.api.testpilot.v1.FormatVersion",
      value := ByteArray.mk #[8, 1, 16, 2]
    })),
  Expr.literal (Value.floatingPoint 1.5),
  Expr.literal (Value.enumeration 1),
  Expr.literal (Value.boolean false)
]

private def literalProgram : temporal.server.api.testpilot.v1.Program := Program.make "program"
  #[Program.role "endpoint" .ROLE_KIND_ENDPOINT]
  #[Program.valueSlot "slot" (Types.singular (Types.scalar .SCALAR_KIND_TEXT))]
  #[Program.observation "result" (Types.singular Types.any)]
  #[Program.controller "controller" #[Program.node "finish"
    (Program.finish instructionExpression) instructionLimits (guard := some instructionExpression)]]
  (Program.cleanup "cleanup" #[])

private def contract : Contract := Contract.contract "contract" #[
  Contract.rule "rule" .CONTRACT_RULE_KIND_BOUNDED_LIVENESS "open"
    #[Contract.state "open" .CONTRACT_STATE_STATUS_PENDING,
      Contract.state "done" .CONTRACT_STATE_STATUS_SATISFIED,
      Contract.state "late" .CONTRACT_STATE_STATUS_VIOLATED]
    #[Contract.transition "complete" "open" "done" #[.RUN_EVENT_KIND_RUN_CLOSED]
      predicateExpression .CONTRACT_SUPPORT_KIND_MATCHING_EVENT
      #[Contract.captureAssignment "captured" "result"]]
    (deadline := some (Contract.deadline (.elapsed_milliseconds 9223372036854775807) "late"))
    (captures := #[Contract.capture "captured" (Types.messageType
      "temporal.server.api.testpilot.v1.FormatVersion")])
]

def literalCase : Case := Testpilot.Authoring.case 1 "case" literalProgram contract
  (provenance "testpilot-tests" "1" (ByteArray.mk #[0, 255, 128]))

private def bindingProgram : temporal.server.api.testpilot.v1.Program := Program.make "program"
  #[Program.role "endpoint" .ROLE_KIND_ENDPOINT (resourceBindingId := "nexus.endpoint"),
    Program.role "worker" .ROLE_KIND_WORKER (namespaceBindingId := "namespace"),
    Program.role "queue" .ROLE_KIND_TASK_QUEUE
      (namespaceBindingId := "namespace") (resourceBindingId := "task.queue")]
  #[] #[] #[Program.controller "controller" #[Program.node "finish"
    (Program.finish (Expr.environment "")) instructionLimits]]
  (Program.cleanup "cleanup" #[])

def representativeCase : Case := Testpilot.Authoring.case 1 "binding-case" bindingProgram contract
  (provenance "testpilot-tests" "1" (ByteArray.mk #[0, 255, 128]))

private def unknownAnyCase : Case :=
  let expression := Expr.literal (Value.messageValue {
    type_url := "type.googleapis.com/example.Unknown"
    value := ByteArray.mk #[8, 1]
  })
  let unknownProgram := Program.make "program" #[] #[] #[]
    #[Program.controller "controller" #[Program.node "finish" (Program.finish expression)
      instructionLimits]] (Program.cleanup "cleanup" #[])
  Testpilot.Authoring.case 1 "unknown-any" unknownProgram contract (provenance "test" "1")

private def malformedAnyCase : Case :=
  let expression := Expr.literal (Value.messageValue {
    type_url := "type.googleapis.com/temporal.server.api.testpilot.v1.FormatVersion"
    value := ByteArray.mk #[255]
  })
  let malformedProgram := Program.make "program" #[] #[] #[]
    #[Program.controller "controller" #[Program.node "finish" (Program.finish expression)
      instructionLimits]] (Program.cleanup "cleanup" #[])
  Testpilot.Authoring.case 1 "malformed-any" malformedProgram contract (provenance "test" "1")

/-- An instruction guard that reads an Observation, which only a Contract predicate may read. The
one expression type lets Authoring build and render it; Go preparation rejects it. -/
private def contextMismatchCase : Case :=
  let mismatchProgram := Program.make "program" #[] #[]
    #[Program.observation "result" (Types.singular Types.any)]
    #[Program.controller "controller" #[Program.node "finish"
      (Program.finish (Expr.literal (Value.boolean true))) instructionLimits
      (guard := some (Expr.observation "result"))]]
    (Program.cleanup "cleanup" #[])
  Testpilot.Authoring.case 1 "context-mismatch" mismatchProgram contract (provenance "test" "1")

private def nestedExpression : Nat → Expression
  | 0 => Expr.literal (Value.boolean true)
  | depth + 1 => Expr.negate (nestedExpression depth)

private def recursionFailureCase : Case :=
  let deepProgram := Program.make "program" #[] #[] #[]
    #[Program.controller "controller" #[Program.node "finish"
      (Program.finish (nestedExpression 101)) instructionLimits]]
    (Program.cleanup "cleanup" #[])
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
  assert (first.contains "\"reference\":{\"environmentBindingId\":\"\"}")
    "present empty environment reference was dropped"
  assert (!first.contains "\"environment\":")
    "a Program declared the environment bindings preparation derives"
  assert (first.contains "\"namespaceBindingId\":\"namespace\"") "namespace binding was dropped"
  assert (first.contains "\"resourceBindingId\":\"task.queue\"") "resource binding was dropped"
  assert (first.contains "\"reference\":{\"runEvent\":{\"field\":\"RUN_EVENT_FIELD_RUN_ID\"}}")
    "Contract Run Event identity was dropped"
  assert (first.contains "\"maxAttempts\":\"9223372036854775807\"")
    "int64 upper bound was not rendered as a ProtoJSON string"
  assert (first.contains "\"elapsedMilliseconds\":\"9223372036854775807\"")
    "monitor deadline was dropped"
  assert (first.contains "AP+A") "opaque non-UTF-8 provenance bytes were not rendered"
  assert (first.contains "\"floatingPointValue\":1.5") "floating value was dropped"
  assert (first.contains "\"enumValue\":{\"number\":1}") "enum value was dropped"
  assert (first.contains "\"boolValue\":false") "present false oneof value was dropped"
  assert (first.contains "\"@type\":\"type.googleapis.com/temporal.server.api.testpilot.v1.FormatVersion\"")
    "resolved Any was dropped"
  let literalFirst ← render literalCase
  let literalSecond ← render literalCase
  assert (literalFirst == literalSecond) "literal Case 1.0 encoding changed nondeterministically"
  assert (literalFirst.contains "\"version\":{\"major\":1}") "literal Case 1.0 version changed"
  assert (!literalFirst.contains "bindingId") "resource-free Case 1.0 gained binding fields"
  assert (literalFirst.contains "\"reference\":{\"run\":{}}") "Program Run identity was dropped"
  assert (literalFirst.contains "AP8=") "expression bytes were not rendered"
  let mismatch ← render contextMismatchCase
  assert (mismatch.contains "\"guard\":{\"reference\":{\"observationId\":\"result\"}}")
    "an instruction guard reading an Observation was not rendered"
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
