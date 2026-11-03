//! Every `WorkflowService` RPC by resource family, shared by all Temporal features.

use umpire_temporal_api::WorkflowServiceMethod;

/// A feature's relationship to one RPC.
pub enum RpcRole<A> {
    /// The RPC performs this Model action.
    Action(A),
    /// The RPC observes the Model without performing an action.
    Observed(&'static str),
    /// The feature does not claim the RPC; the reason is its resource family.
    OutOfScope(&'static str),
}

/// The resource family of every RPC, in descriptor order. A new upstream RPC fails the
/// build until it has a family; a feature reports the family of each RPC it does not claim.
#[allow(
    clippy::match_same_arms,
    clippy::too_many_lines,
    reason = "Descriptor admission requires RPC declaration order, so a family recurs between groups"
)]
pub const fn family(method: WorkflowServiceMethod) -> &'static str {
    match method {
        WorkflowServiceMethod::RegisterNamespace
        | WorkflowServiceMethod::DescribeNamespace
        | WorkflowServiceMethod::ListNamespaces
        | WorkflowServiceMethod::UpdateNamespace
        | WorkflowServiceMethod::DeprecateNamespace => "Namespace administration",
        WorkflowServiceMethod::StartWorkflowExecution
        | WorkflowServiceMethod::ExecuteMultiOperation
        | WorkflowServiceMethod::GetWorkflowExecutionHistory
        | WorkflowServiceMethod::GetWorkflowExecutionHistoryReverse
        | WorkflowServiceMethod::PollWorkflowTaskQueue
        | WorkflowServiceMethod::RespondWorkflowTaskCompleted
        | WorkflowServiceMethod::RespondWorkflowTaskFailed => "Workflow executions",
        WorkflowServiceMethod::PollActivityTaskQueue
        | WorkflowServiceMethod::RecordActivityTaskHeartbeat
        | WorkflowServiceMethod::RecordActivityTaskHeartbeatById
        | WorkflowServiceMethod::RespondActivityTaskCompleted
        | WorkflowServiceMethod::RespondActivityTaskCompletedById
        | WorkflowServiceMethod::RespondActivityTaskFailed
        | WorkflowServiceMethod::RespondActivityTaskFailedById
        | WorkflowServiceMethod::RespondActivityTaskCanceled
        | WorkflowServiceMethod::RespondActivityTaskCanceledById => "Activity tasks and executions",
        WorkflowServiceMethod::RequestCancelWorkflowExecution
        | WorkflowServiceMethod::SignalWorkflowExecution
        | WorkflowServiceMethod::SignalWithStartWorkflowExecution
        | WorkflowServiceMethod::ResetWorkflowExecution
        | WorkflowServiceMethod::TerminateWorkflowExecution
        | WorkflowServiceMethod::DeleteWorkflowExecution
        | WorkflowServiceMethod::ListOpenWorkflowExecutions
        | WorkflowServiceMethod::ListClosedWorkflowExecutions
        | WorkflowServiceMethod::ListWorkflowExecutions
        | WorkflowServiceMethod::ListArchivedWorkflowExecutions
        | WorkflowServiceMethod::ScanWorkflowExecutions
        | WorkflowServiceMethod::CountWorkflowExecutions => "Workflow executions",
        WorkflowServiceMethod::GetSearchAttributes => "Cluster and service information",
        WorkflowServiceMethod::RespondQueryTaskCompleted => "Workflow executions",
        WorkflowServiceMethod::ResetStickyTaskQueue => "Task queue administration",
        WorkflowServiceMethod::ShutdownWorker => "Worker infrastructure",
        WorkflowServiceMethod::QueryWorkflow | WorkflowServiceMethod::DescribeWorkflowExecution => {
            "Workflow executions"
        }
        WorkflowServiceMethod::DescribeTaskQueue => "Task queue administration",
        WorkflowServiceMethod::GetClusterInfo | WorkflowServiceMethod::GetSystemInfo => {
            "Cluster and service information"
        }
        WorkflowServiceMethod::ListTaskQueuePartitions => "Task queue administration",
        WorkflowServiceMethod::CreateSchedule
        | WorkflowServiceMethod::DescribeSchedule
        | WorkflowServiceMethod::UpdateSchedule
        | WorkflowServiceMethod::PatchSchedule
        | WorkflowServiceMethod::ListScheduleMatchingTimes
        | WorkflowServiceMethod::DeleteSchedule
        | WorkflowServiceMethod::ListSchedules
        | WorkflowServiceMethod::CountSchedules => "Schedules",
        WorkflowServiceMethod::UpdateWorkerBuildIdCompatibility
        | WorkflowServiceMethod::GetWorkerBuildIdCompatibility
        | WorkflowServiceMethod::UpdateWorkerVersioningRules
        | WorkflowServiceMethod::GetWorkerVersioningRules
        | WorkflowServiceMethod::GetWorkerTaskReachability
        | WorkflowServiceMethod::DescribeDeployment
        | WorkflowServiceMethod::DescribeWorkerDeploymentVersion
        | WorkflowServiceMethod::ListDeployments
        | WorkflowServiceMethod::GetDeploymentReachability
        | WorkflowServiceMethod::GetCurrentDeployment
        | WorkflowServiceMethod::SetCurrentDeployment
        | WorkflowServiceMethod::SetWorkerDeploymentCurrentVersion
        | WorkflowServiceMethod::DescribeWorkerDeployment
        | WorkflowServiceMethod::DeleteWorkerDeployment
        | WorkflowServiceMethod::DeleteWorkerDeploymentVersion
        | WorkflowServiceMethod::SetWorkerDeploymentRampingVersion
        | WorkflowServiceMethod::ListWorkerDeployments
        | WorkflowServiceMethod::CreateWorkerDeployment
        | WorkflowServiceMethod::CreateWorkerDeploymentVersion
        | WorkflowServiceMethod::UpdateWorkerDeploymentVersionComputeConfig
        | WorkflowServiceMethod::ValidateWorkerDeploymentVersionComputeConfig
        | WorkflowServiceMethod::UpdateWorkerDeploymentVersionMetadata
        | WorkflowServiceMethod::SetWorkerDeploymentManager => "Worker deployments and versioning",
        WorkflowServiceMethod::UpdateWorkflowExecution
        | WorkflowServiceMethod::PollWorkflowExecutionUpdate => "Workflow executions",
        WorkflowServiceMethod::StartBatchOperation
        | WorkflowServiceMethod::StopBatchOperation
        | WorkflowServiceMethod::DescribeBatchOperation
        | WorkflowServiceMethod::ListBatchOperations => "Batch operations",
        WorkflowServiceMethod::PollNexusTaskQueue
        | WorkflowServiceMethod::RespondNexusTaskCompleted
        | WorkflowServiceMethod::RespondNexusTaskFailed => "Nexus tasks and operations",
        WorkflowServiceMethod::UpdateActivityOptions => "Activity tasks and executions",
        WorkflowServiceMethod::UpdateWorkflowExecutionOptions => "Workflow executions",
        WorkflowServiceMethod::PauseActivity
        | WorkflowServiceMethod::UnpauseActivity
        | WorkflowServiceMethod::ResetActivity => "Activity tasks and executions",
        WorkflowServiceMethod::CreateWorkflowRule
        | WorkflowServiceMethod::DescribeWorkflowRule
        | WorkflowServiceMethod::DeleteWorkflowRule
        | WorkflowServiceMethod::ListWorkflowRules
        | WorkflowServiceMethod::TriggerWorkflowRule => "Workflow rules",
        WorkflowServiceMethod::RecordWorkerHeartbeat
        | WorkflowServiceMethod::ListWorkers
        | WorkflowServiceMethod::CountWorkers => "Worker infrastructure",
        WorkflowServiceMethod::UpdateTaskQueueConfig => "Task queue administration",
        WorkflowServiceMethod::FetchWorkerConfig
        | WorkflowServiceMethod::UpdateWorkerConfig
        | WorkflowServiceMethod::DescribeWorker => "Worker infrastructure",
        WorkflowServiceMethod::PauseWorkflowExecution
        | WorkflowServiceMethod::UnpauseWorkflowExecution => "Workflow executions",
        WorkflowServiceMethod::StartActivityExecution => "Activity tasks and executions",
        WorkflowServiceMethod::StartNexusOperationExecution => "Nexus tasks and operations",
        WorkflowServiceMethod::DescribeActivityExecution => "Activity tasks and executions",
        WorkflowServiceMethod::DescribeNexusOperationExecution => "Nexus tasks and operations",
        WorkflowServiceMethod::PollActivityExecution => "Activity tasks and executions",
        WorkflowServiceMethod::PollNexusOperationExecution => "Nexus tasks and operations",
        WorkflowServiceMethod::ListActivityExecutions => "Activity tasks and executions",
        WorkflowServiceMethod::ListNexusOperationExecutions => "Nexus tasks and operations",
        WorkflowServiceMethod::CountActivityExecutions => "Activity tasks and executions",
        WorkflowServiceMethod::CountNexusOperationExecutions => "Nexus tasks and operations",
        WorkflowServiceMethod::RequestCancelActivityExecution => "Activity tasks and executions",
        WorkflowServiceMethod::RequestCancelNexusOperationExecution => "Nexus tasks and operations",
        WorkflowServiceMethod::TerminateActivityExecution
        | WorkflowServiceMethod::DeleteActivityExecution
        | WorkflowServiceMethod::PauseActivityExecution
        | WorkflowServiceMethod::ResetActivityExecution
        | WorkflowServiceMethod::UnpauseActivityExecution
        | WorkflowServiceMethod::UpdateActivityExecutionOptions => "Activity tasks and executions",
        WorkflowServiceMethod::TerminateNexusOperationExecution
        | WorkflowServiceMethod::DeleteNexusOperationExecution => "Nexus tasks and operations",
        WorkflowServiceMethod::PollWorkflowExecutionTimeSkipping => "Workflow executions",
    }
}
