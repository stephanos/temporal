import Temporal.Case.EventKind
import Temporal.Testpilot.CaseSupport

/-!
# What every realization template shares

The roles a Case addresses, the two workflow-service methods it invokes, the observations it
declares, and the history filter that makes a read wait for the close event. They live here rather
than beside each template so two templates cannot drift into naming the same thing differently --
`DeriveProfile` resolves a Case's bindings by the IDs the Program declares, so a divergence here is
a divergence in what a Case authorizes.
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

/-- `HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT`, read from the generated enum rather than spelled as a
number here. A read carrying it blocks until the workflow closes and returns only the closing
event, so it is what orders a later read after the work completed. -/
def closeEventFilter : ProgramExpression :=
  ProgramExpr.literal (Value.enumeration
    (Int32.ofInt Temporal.Api.Enums.V1.HistoryEventFilterType.historyEventFilterTypeCloseEvent.number))

end Temporal.Case.Support
