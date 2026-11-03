//! Exhaustive activity API mirrors, independent of the Model's runtime state.
#![allow(dead_code, reason = "Mirror fields describe the API and are not instantiated")]
use crate::activity::{self, Client, Deadline, Worker};
use crate::foundation::dc;
use umpire_catalog::{
    Catalog,
    kinds::{
        Counter, Duration, Enum, Flag, Identifier, Message, OneOf, RawDuration, Secret, Text,
        Timestamp, Uuid,
    },
};
use umpire_temporal_api::temporal::api::{
    activity::v1 as a, common::v1 as c, enums::v1 as e, failure::v1 as f, taskqueue::v1 as q,
    workflowservice::v1 as w,
};
type ModelAction = <activity::World as umpire_dsl::Machine>::Action;
#[path = "api_transport.rs"]
mod transport;
use crate::foundation::service::{self, RpcRole};
pub use transport::{AnnotationResult, annotate};
use umpire_catalog::Binding;
use umpire_temporal_api::WorkflowServiceMethod;
/// The RPCs this feature claims; every other RPC is out of scope with its resource family.
fn classify(method: WorkflowServiceMethod) -> RpcRole<ModelAction> {
    match method {
        WorkflowServiceMethod::PollActivityTaskQueue => {
            RpcRole::Action(ModelAction::Worker(Worker::Poll(1)))
        }
        WorkflowServiceMethod::RecordActivityTaskHeartbeat => {
            RpcRole::Action(ModelAction::Worker(Worker::Heartbeat))
        }
        WorkflowServiceMethod::RespondActivityTaskCompleted => {
            RpcRole::Action(ModelAction::Worker(Worker::RespondCompleted))
        }
        WorkflowServiceMethod::RespondActivityTaskFailed => {
            RpcRole::Action(ModelAction::Worker(Worker::RespondFailed { retryable: false }))
        }
        WorkflowServiceMethod::RespondActivityTaskCanceled => {
            RpcRole::Observed("worker acknowledges client cancellation")
        }
        WorkflowServiceMethod::StartActivityExecution => {
            RpcRole::Observed("successful start establishes the Model initial state")
        }
        WorkflowServiceMethod::DescribeActivityExecution => {
            RpcRole::Observed("activity lifecycle phase")
        }
        WorkflowServiceMethod::PollActivityExecution => {
            RpcRole::Observed("SDK activity handle outcome")
        }
        WorkflowServiceMethod::RequestCancelActivityExecution => {
            RpcRole::Action(ModelAction::Client(Client::Cancel))
        }
        _ => RpcRole::OutOfScope(service::family(method)),
    }
}
/// Executable factories and exhaustive runtime bindings.
pub mod catalog {
    use super::{Binding, Client, Deadline, ModelAction, Worker, a, c, dc, f, q, w};
    use crate::foundation::api as foundation;
    pub use crate::foundation::enum_inventory::enum_inventory::*;
    use umpire_catalog::{AdmissionErrors, Catalog, ProbeCase};
    pub use umpire_temporal_api::RPC_BINDINGS;
    macro_rules! factories {
        (
            $($source:ident => $target:ident : [$($mirror:ident => $factory:ident),*
            $(,)?]);* $(;)?
        ) => {
            $($(#[doc = concat!("Admits the catalog for ", stringify!($mirror), ".")] pub
            fn $factory (limits : & dc::Limits) -> Result < Catalog <$target ::$mirror >,
            AdmissionErrors > { $source ::$mirror ::catalog(limits) })*)* #[doc =
            " Every declared primary and supplemental expectation."] pub fn
            expected_probe_cases(limits : & dc::Limits) -> Result < Vec < ProbeCase >,
            AdmissionErrors > { let mut cases = Vec::new(); $($(for catalog in $source
            ::$mirror ::catalogs(limits) ? { cases.extend(catalog.probe_cases()); })*)*
            cases.sort_by(| a, b | a.id.cmp(& b.id)); Ok(cases) }
        };
    }
    factories! {
        foundation => c : [WorkflowExecution => workflow_execution, WorkflowType =>
        workflow_type, Payloads => payloads, Payload => payload, ActivityType =>
        activity_type, RetryPolicy => retry_policy,]; super => a :
        [ActivityExecutionOutcome => activity_execution_outcome, ActivityExecutionInfo =>
        activity_execution_info,]; foundation => f : [ApplicationFailureInfo =>
        application_failure_info, Failure => failure, TimeoutFailureInfo =>
        timeout_failure_info, CanceledFailureInfo => canceled_failure_info,
        TerminatedFailureInfo => terminated_failure_info, ServerFailureInfo =>
        server_failure_info, ResetWorkflowFailureInfo => reset_workflow_failure_info,
        ActivityFailureInfo => activity_failure_info, ChildWorkflowExecutionFailureInfo
        => child_workflow_execution_failure_info, NexusOperationFailureInfo =>
        nexus_operation_failure_info, NexusHandlerFailureInfo =>
        nexus_handler_failure_info,]; foundation => q : [TaskQueue => task_queue,]; super
        => w : [PollActivityTaskQueueRequest => poll_activity_task_queue_request,
        PollActivityTaskQueueResponse => poll_activity_task_queue_response,
        RecordActivityTaskHeartbeatRequest => record_activity_task_heartbeat_request,
        RecordActivityTaskHeartbeatResponse => record_activity_task_heartbeat_response,
        RespondActivityTaskCompletedRequest => respond_activity_task_completed_request,
        RespondActivityTaskCompletedResponse => respond_activity_task_completed_response,
        RespondActivityTaskFailedRequest => respond_activity_task_failed_request,
        RespondActivityTaskFailedResponse => respond_activity_task_failed_response,
        RespondActivityTaskCanceledRequest => respond_activity_task_canceled_request,
        RespondActivityTaskCanceledResponse => respond_activity_task_canceled_response,
        StartActivityExecutionRequest => start_activity_execution_request,
        StartActivityExecutionResponse => start_activity_execution_response,
        DescribeActivityExecutionRequest => describe_activity_execution_request,
        DescribeActivityExecutionResponse => describe_activity_execution_response,
        PollActivityExecutionRequest => poll_activity_execution_request,
        PollActivityExecutionResponse => poll_activity_execution_response,
        RequestCancelActivityExecutionRequest =>
        request_cancel_activity_execution_request, RequestCancelActivityExecutionResponse
        => request_cancel_activity_execution_response,];
    }
    /// Binds every Model action, including explicitly environmental deadlines.
    pub fn bind(action: &ModelAction) -> Binding {
        umpire_temporal_api::bind_workflow_action!(
            action,
            ModelAction::Deadline(Deadline::Tick) | ModelAction::Worker(Worker::RespondCanceled(_)) => {
                if matches!(action, ModelAction::Deadline(Deadline::Tick)) {
                    Binding::Environmental("abstract deadline tick")
                } else {
                    Binding::Rpc("temporal.api.workflowservice.v1.WorkflowService.RespondActivityTaskCanceled")
                }
            }
        )
    }
}
/// UUID intent for request IDs accepted without UUID syntax by the server.
pub fn start_request_id_intent(
    limits: &dc::Limits,
) -> Result<Catalog<w::StartActivityExecutionRequest>, umpire_catalog::AdmissionErrors> {
    StartActivityExecutionRequest::__request_id_also_0(limits)
}
/// UUID intent for cancellation request IDs.
pub fn cancel_request_id_intent(
    limits: &dc::Limits,
) -> Result<Catalog<w::RequestCancelActivityExecutionRequest>, umpire_catalog::AdmissionErrors> {
    RequestCancelActivityExecutionRequest::__request_id_also_0(limits)
}
/// Worker-supplied failures require application failure information.
pub fn worker_failure_presence(
    limits: &dc::Limits,
) -> Result<Catalog<f::Failure>, umpire_catalog::AdmissionErrors> {
    crate::foundation::api::WorkerFailure::catalog(limits)
}
/// Source-reviewed dispositions keyed by complete contextual probe identity.
pub(crate) fn accepted_probe_gaps() -> Vec<(String, umpire_catalog::AcceptedGap)> {
    Vec::new()
}
umpire_temporal_api::workflow_feature_activity_catalog!();

/// The protocol's sticky-only normal-name condition, scoped to worker queue polling.
pub fn poll_normal_name(
    limits: &dc::Limits,
) -> Result<Catalog<w::PollActivityTaskQueueRequest>, umpire_catalog::AdmissionErrors> {
    PollActivityTaskQueueRequest::__task_queue_also_0(limits)
}

/// The public retry contract's ordering condition for explicit enabled start policies.
pub fn start_retry_order(
    limits: &dc::Limits,
) -> Result<Catalog<w::StartActivityExecutionRequest>, umpire_catalog::AdmissionErrors> {
    StartActivityExecutionRequest::__retry_policy_also_0(limits)
}
