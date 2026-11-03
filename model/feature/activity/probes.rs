//! Activity-owned baseline policy retained by the probe migration adapter.
#![expect(
    clippy::unnecessary_wraps,
    reason = "Activity baselines implement the declared refusable filler signature"
)]

use crate::probe_fillers::{Baselines, Filler};
use crate::probes::{Result, UUID, application, duration};
use umpire_runtime::{AnyMessage as _, prost_types::Any, protocol::Refusal};
use umpire_temporal_api::{
    temporal::api::{
        common::v1::{ActivityType, RetryPolicy},
        failure::v1::{
            ActivityFailureInfo, ChildWorkflowExecutionFailureInfo, Failure,
            NexusHandlerFailureInfo, TimeoutFailureInfo, failure::FailureInfo,
        },
        taskqueue::v1::TaskQueue,
        workflowservice::v1::{
            DescribeActivityExecutionRequest, PollActivityExecutionRequest,
            PollActivityTaskQueueRequest, RequestCancelActivityExecutionRequest,
            RespondActivityTaskCanceledRequest, RespondActivityTaskCompletedRequest,
            RespondActivityTaskFailedRequest, StartActivityExecutionRequest,
        },
    },
    umpire::temporal::v1::ProbeRequest,
};

static FILLERS: &[Filler<ProbeRequest>] = &[
    Filler { method: "StartActivityExecution", fill: fill_start },
    Filler { method: "DescribeActivityExecution", fill: fill_describe },
    Filler { method: "PollActivityExecution", fill: fill_poll_execution },
    Filler { method: "RequestCancelActivityExecution", fill: fill_cancel },
    Filler { method: "PollActivityTaskQueue", fill: fill_poll_queue },
    Filler { method: "RespondActivityTaskCompleted", fill: fill_complete },
    Filler { method: "RespondActivityTaskFailed", fill: fill_fail },
    Filler { method: "RespondActivityTaskCanceled", fill: fill_canceled },
];

/// The admitted activity baseline registry consumed by the probe inventory and its fills.
pub(crate) fn baselines() -> Result<Baselines<'static, ProbeRequest>> {
    Ok(Baselines::new(FILLERS).map_err(|refusal| refusal.message)?)
}

pub(crate) fn fill(name: &str, slots: &ProbeRequest, context: &str) -> Result<Vec<u8>> {
    Ok(baselines()?.fill(name, slots, context).map_err(|refusal| refusal.message)?.value)
}

fn fill_start(slots: &ProbeRequest, _context: &str) -> std::result::Result<Any, Refusal> {
    let queue = TaskQueue { name: slots.task_queue.clone(), ..Default::default() };
    Ok(Any::pack(&StartActivityExecutionRequest {
        namespace: slots.namespace.clone(),
        activity_id: slots.activity_id.clone(),
        request_id: UUID.into(),
        identity: "umpire-probe".into(),
        activity_type: Some(ActivityType { name: "ModelSlice".into() }),
        task_queue: Some(queue),
        schedule_to_close_timeout: Some(duration(60)),
        start_to_close_timeout: Some(duration(20)),
        schedule_to_start_timeout: Some(duration(30)),
        heartbeat_timeout: Some(duration(10)),
        retry_policy: Some(RetryPolicy {
            maximum_attempts: 2,
            initial_interval: Some(duration(1)),
            maximum_interval: Some(duration(10)),
            backoff_coefficient: 2.0,
            ..Default::default()
        }),
        ..Default::default()
    }))
}

fn fill_describe(slots: &ProbeRequest, _context: &str) -> std::result::Result<Any, Refusal> {
    Ok(Any::pack(&DescribeActivityExecutionRequest {
        namespace: slots.namespace.clone(),
        activity_id: slots.activity_id.clone(),
        run_id: slots.run_id.clone(),
        ..Default::default()
    }))
}

fn fill_poll_execution(slots: &ProbeRequest, _context: &str) -> std::result::Result<Any, Refusal> {
    Ok(Any::pack(&PollActivityExecutionRequest {
        namespace: slots.namespace.clone(),
        activity_id: slots.activity_id.clone(),
        run_id: slots.run_id.clone(),
    }))
}

fn fill_cancel(slots: &ProbeRequest, _context: &str) -> std::result::Result<Any, Refusal> {
    Ok(Any::pack(&RequestCancelActivityExecutionRequest {
        namespace: slots.namespace.clone(),
        activity_id: slots.activity_id.clone(),
        run_id: slots.run_id.clone(),
        request_id: UUID.into(),
        identity: "umpire-probe".into(),
        ..Default::default()
    }))
}

fn fill_poll_queue(slots: &ProbeRequest, _context: &str) -> std::result::Result<Any, Refusal> {
    let queue = TaskQueue { name: slots.task_queue.clone(), ..Default::default() };
    Ok(Any::pack(&PollActivityTaskQueueRequest {
        namespace: slots.namespace.clone(),
        task_queue: Some(queue),
        identity: "umpire-probe".into(),
        ..Default::default()
    }))
}

fn fill_complete(slots: &ProbeRequest, _context: &str) -> std::result::Result<Any, Refusal> {
    Ok(Any::pack(&RespondActivityTaskCompletedRequest {
        namespace: slots.namespace.clone(),
        task_token: slots.task_token.clone(),
        identity: "umpire-probe".into(),
        ..Default::default()
    }))
}

fn fill_fail(slots: &ProbeRequest, context: &str) -> std::result::Result<Any, Refusal> {
    Ok(Any::pack(&{
        let mut failure = application();
        let info = match context {
            "failed.cause.activity" => {
                Some(FailureInfo::ActivityFailureInfo(ActivityFailureInfo::default()))
            }
            "failed.cause.child" => Some(FailureInfo::ChildWorkflowExecutionFailureInfo(
                ChildWorkflowExecutionFailureInfo::default(),
            )),
            "failed.cause.timeout" => {
                Some(FailureInfo::TimeoutFailureInfo(TimeoutFailureInfo::default()))
            }
            "failed.cause.nexus" => {
                Some(FailureInfo::NexusHandlerFailureInfo(NexusHandlerFailureInfo::default()))
            }
            _ => None,
        };
        if info.is_some() {
            failure.cause = Some(Box::new(Failure { failure_info: info, ..Default::default() }));
        }
        RespondActivityTaskFailedRequest {
            namespace: slots.namespace.clone(),
            task_token: slots.task_token.clone(),
            identity: "umpire-probe".into(),
            failure: Some(failure),
            ..Default::default()
        }
    }))
}

fn fill_canceled(slots: &ProbeRequest, _context: &str) -> std::result::Result<Any, Refusal> {
    Ok(Any::pack(&RespondActivityTaskCanceledRequest {
        namespace: slots.namespace.clone(),
        task_token: slots.task_token.clone(),
        identity: "umpire-probe".into(),
        ..Default::default()
    }))
}
