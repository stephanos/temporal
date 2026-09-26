import Temporal.Testpilot.CaseSupport

/-!
# The read observations the realization catalog binds

The third answer beside the generated history event kinds (`Temporal.Case.EventKind`) and the
Testpilot Run Event kinds: an observation a Case reads back through a unary RPC rather than out of
recorded history. Each binding names the RPC, the repeated field whose elements are read, the path
of the operation key inside one element and the fields the observation exposes; the template that
admits it emits the Program's declaration from here, so the name an `evidence` line writes and the
declaration the Case carries cannot drift apart.

`pendingAttempts` reads what no history event records: a retryable attempt failure leaves only the
pending operation's `attempt` behind, in `DescribeWorkflowExecution`. `scheduledEvent` reads the
scheduled event out of history as soon as it exists, ahead of the history read that closes the Run:
the verifier orders one operation's evidence across sources by the order it was lifted in, so the
evidence that opens the operation is lifted before any poll that follows it.
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

def getWorkflowExecutionHistoryMethod :=
  "/temporal.api.workflowservice.v1.WorkflowService/GetWorkflowExecutionHistory"

/-- The scheduled event, read out of history by its own event id. It exposes no field: the
Contract confirms the kind, and the event's attributes are read by the history read that follows. -/
def scheduledEvent : Binding := {
  name := "nexusOperationScheduled"
  method := getWorkflowExecutionHistoryMethod
  path := historyEvents
  operationKey := field "event_id"
  fields := [] }

def bindings : List Binding := [pendingAttempts, scheduledEvent]

/-- Every read observation an `evidence:` line may name. -/
def admitted : List String := bindings.map (·.name)

end Temporal.Case.ReadKind
