import Temporal.Testpilot.CaseSupport

/-!
# The read observations the realization catalog binds

The third answer beside the generated history event kinds (`Temporal.Case.EventKind`) and the
Testpilot Run Event kinds: an observation a Case reads back through a unary RPC rather than out of
recorded history. Each binding names the RPC, the repeated field whose elements are read, the path
of the operation key inside one element and the fields the observation exposes; the template that
admits it emits the Program's declaration from here, so the name an `evidence` line writes and the
declaration the Case carries cannot drift apart.

`pendingAttempts` is the one binding today: a retryable attempt failure writes no history event, and
only `DescribeWorkflowExecution` shows the pending operation's `attempt`.
-/

namespace Temporal.Case.ReadKind

open Testpilot.Authoring
open Temporal.Testpilot.CaseSupport

/-- One read observation: where it is read from and what it exposes. -/
structure Binding where
  name : String
  /-- The full method name, `/package.Service/Method`. -/
  method : String
  /-- The repeated field of the response whose elements are read. -/
  path : String
  /-- The operation key inside one element. -/
  operationKey : String
  /-- The fields one element exposes, each with its path inside the element. -/
  fields : List (String × String)

def describeWorkflowExecutionMethod :=
  "/temporal.api.workflowservice.v1.WorkflowService/DescribeWorkflowExecution"

def pendingAttempts : Binding := {
  name := "pendingAttempts"
  method := describeWorkflowExecutionMethod
  path := field "pending_nexus_operations"
  operationKey := field "scheduled_event_id"
  fields := [("attempts", field "attempt")] }

def bindings : List Binding := [pendingAttempts]

/-- Every read observation an `evidence:` line may name. -/
def admitted : List String := bindings.map (·.name)

end Temporal.Case.ReadKind
