import Temporal.Case.EventKind
import Temporal.Testpilot.CaseSupport

/-!
# What every realization shares

The roles a Case addresses, the two workflow-service methods it invokes, the observations it
declares, and the history filter that makes a read wait for the close event. They live here rather
than beside each realization so two realizations cannot drift into naming the same thing
differently -- `DeriveProfile` resolves a Case's bindings by the IDs the Program declares, so a
divergence here is a divergence in what a Case authorizes.
-/

namespace Temporal.Case.Support

open Umpire
open Temporal.Testpilot.CaseSupport
open Testpilot.Authoring
open temporal.server.api.testpilot.v1

/-! ### Roles a Case addresses -/

def workflowServiceRole := "temporal.workflow-service"
def workerRole := "temporal.worker"
def taskQueueRole := "temporal.task-queue"
/-- The queue the Nexus handler polls, its own so a fault that stops the handler's worker leaves the
caller workflow's running. -/
def handlerTaskQueueRole := "temporal.handler-task-queue"
def nexusEndpointRole := "temporal.nexus-endpoint"

/-! ### Methods a Case invokes -/

def startWorkflowMethod :=
  "/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution"
def getHistoryMethod :=
  "/temporal.api.workflowservice.v1.WorkflowService/GetWorkflowExecutionHistory"

/-! ### Observations a Case declares -/

def historyObservation := "history-event"
def correlatedObservation := "correlated-evidence"

def correlatedEvidenceType : ValueType :=
  Types.singular (Types.messageType "temporal.server.api.testpilot.v1.CorrelatedEvidence")

/-- `HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT`. The generated Lean enum carries numbers only, so the
name is spelled here, and preparation rejects it unless the request field's enum declares it. A read
carrying it blocks until the workflow closes and returns only the closing event, so it is what
orders a later read after the work completed. -/
def closeEventFilter : Expression :=
  Expr.literal (Value.enumeration "HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT")

end Temporal.Case.Support
