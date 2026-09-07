import Testpilot.Authoring

/-! Elaboration failures prove that recursive combinators retain their expression context. -/

open temporal.server.api.testpilot.v1
open Testpilot.Authoring

namespace Testpilot.Tests.AuthoringFailures

private def program := ProgramExpr.slot "slot"
private def contract := ContractExpr.observation "observation"
private def path := Path.make #[Path.field "value"]

/--
error: Application type mismatch: The argument
  contract
has type
  ContractExpression
but is expected to have type
  ProgramExpression
in the application
  ProgramExpr.path contract
-/
#guard_msgs (error, substring := true) in
example : ProgramExpression := ProgramExpr.path contract path

/--
error: Application type mismatch: The argument
  program
has type
  ProgramExpression
but is expected to have type
  ContractExpression
in the application
  ContractExpr.path program
-/
#guard_msgs (error, substring := true) in
example : ContractExpression := ContractExpr.path program path

/--
error: Application type mismatch: The argument
  contract
has type
  ContractExpression
but is expected to have type
  ProgramExpression
in the application
  ProgramExpr.equals program contract
-/
#guard_msgs (error, substring := true) in
example : ProgramExpression := ProgramExpr.equals program contract

/--
error: Application type mismatch: The argument
  program
has type
  ProgramExpression
but is expected to have type
  ContractExpression
in the application
  ContractExpr.equals contract program
-/
#guard_msgs (error, substring := true) in
example : ContractExpression := ContractExpr.equals contract program

/--
error: Application type mismatch: The argument
  contract
has type
  ContractExpression
but is expected to have type
  ProgramExpression
in the application
  ProgramExpr.negation contract
-/
#guard_msgs (error, substring := true) in
example : ProgramExpression := ProgramExpr.negation contract

/--
error: Application type mismatch: The argument
  program
has type
  ProgramExpression
but is expected to have type
  ContractExpression
in the application
  ContractExpr.negation program
-/
#guard_msgs (error, substring := true) in
example : ContractExpression := ContractExpr.negation program

/--
error: Application type mismatch: The argument
  contract
has type
  ContractExpression
but is expected to have type
  ProgramExpression
in the application
  List.cons contract
-/
#guard_msgs (error, substring := true) in
example : ProgramExpression := ProgramExpr.all #[program, contract]

/--
error: Application type mismatch: The argument
  program
has type
  ProgramExpression
but is expected to have type
  ContractExpression
in the application
  List.cons program
-/
#guard_msgs (error, substring := true) in
example : ContractExpression := ContractExpr.all #[contract, program]

/--
error: Application type mismatch: The argument
  contract
has type
  ContractExpression
but is expected to have type
  ProgramExpression
in the application
  List.cons contract
-/
#guard_msgs (error, substring := true) in
example : ProgramExpression := ProgramExpr.any #[program, contract]

/--
error: Application type mismatch: The argument
  program
has type
  ProgramExpression
but is expected to have type
  ContractExpression
in the application
  List.cons program
-/
#guard_msgs (error, substring := true) in
example : ContractExpression := ContractExpr.any #[contract, program]

end Testpilot.Tests.AuthoringFailures
