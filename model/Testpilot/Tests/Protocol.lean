import Testpilot.Protocol

/-! Narrow-import checks for the generated Testpilot protocol boundary. -/

open temporal.server.api.testpilot.v1

namespace Testpilot.Tests.Protocol

private def programExpression : ProgramExpression :=
  { expression := some (.literal { value := some (.bool_value true) }) }

private def contractExpression : ContractExpression :=
  { expression := some (.literal { value := some (.bool_value true) }) }

#guard programExpression.expression.isSome
#guard contractExpression.expression.isSome

/--
error: Type mismatch
  contractExpression
has type
  ContractExpression
but is expected to have type
  ProgramExpression
-/
#guard_msgs (error, substring := true) in
example : ProgramExpression := contractExpression

/--
error: Type mismatch
  programExpression
has type
  ProgramExpression
but is expected to have type
  ContractExpression
-/
#guard_msgs (error, substring := true) in
example : ContractExpression := programExpression

end Testpilot.Tests.Protocol
