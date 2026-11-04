//! Activity-owned probe baselines: registered action fillers plus bootstrap/read fillers.
#![expect(
    clippy::unnecessary_wraps,
    reason = "Activity baselines implement the declared refusable filler signature"
)]

use crate::activity_realization::{ActivityIdentity, FillInputs, Realization, activity};
use crate::probe_fillers::{Baselines, Filler};
use crate::probes::{Result, UUID, application, duration};
use activity::{Client, Worker};
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
            StartActivityExecutionRequest,
        },
    },
    umpire::temporal::v1::ProbeRequest,
};

type Action = <activity::World as umpire_dsl::Machine>::Action;

/// Bootstrap and read baselines outside the Model action graph.
static FILLERS: &[Filler<ProbeRequest>] = &[
    Filler { method: "StartActivityExecution", fill: fill_start },
    Filler { method: "DescribeActivityExecution", fill: fill_describe },
    Filler { method: "PollActivityExecution", fill: fill_poll_execution },
];

/// Action baselines fill through the registered action filler with probe-selected inputs.
const ACTIONS: &[&str] = &[
    "RequestCancelActivityExecution",
    "PollActivityTaskQueue",
    "RespondActivityTaskCompleted",
    "RespondActivityTaskFailed",
    "RespondActivityTaskCanceled",
];

/// The admitted bootstrap/read baseline registry consumed by the probe inventory and its fills.
fn baselines() -> Result<Baselines<'static, ProbeRequest>> {
    Ok(Baselines::new(FILLERS).map_err(|refusal| refusal.message)?)
}

/// Every probe baseline method: admitted bootstrap/read fillers and registered actions.
pub(crate) fn methods() -> Result<std::collections::BTreeSet<&'static str>> {
    Ok(baselines()?.methods().chain(ACTIONS.iter().copied()).collect())
}

pub(crate) fn fill(name: &str, slots: &ProbeRequest, context: &str) -> Result<Vec<u8>> {
    if ACTIONS.contains(&name) {
        return Ok(registered(name, slots, context).map_err(|refusal| refusal.message)?.value);
    }
    Ok(baselines()?.fill(name, slots, context).map_err(|refusal| refusal.message)?.value)
}

fn registered(
    name: &str,
    slots: &ProbeRequest,
    context: &str,
) -> std::result::Result<Any, Refusal> {
    const CALLER: &str = "umpire-probe";
    let identity = ActivityIdentity::task(
        slots.namespace.clone(),
        slots.activity_id.clone(),
        slots.run_id.clone(),
        slots.task_queue.clone(),
        slots.task_token.clone(),
        CALLER.into(),
    );
    let worker = |details| FillInputs::Worker { attempt: 1, details };
    let (action, inputs) = match name {
        "RequestCancelActivityExecution" => {
            (Client::Cancel.into(), FillInputs::Cancel { request_id: UUID.into() })
        }
        "PollActivityTaskQueue" => (Worker::Poll(1).into(), worker(None)),
        "RespondActivityTaskCompleted" => {
            (Worker::RespondCompleted.into(), FillInputs::Completed { attempt: 1, result: None })
        }
        "RespondActivityTaskFailed" => (
            Worker::RespondFailed { retryable: true }.into(),
            FillInputs::Failed { attempt: 1, failure: Box::new(failure(context)) },
        ),
        "RespondActivityTaskCanceled" => (Worker::RespondCanceled(1).into(), worker(None)),
        _ => return Err(Refusal::unsupported("unknown prerequisite method")),
    };
    let action: Action = action;
    crate::probe_fillers::registered(
        &Realization::default(),
        &format!("/temporal.api.workflowservice.v1.WorkflowService/{name}"),
        &action,
        &activity::World::new(1, None, None, None, None),
        &inputs,
        &identity,
    )
}

/// A retryable probe application failure with the context's declared cause.
fn failure(context: &str) -> Failure {
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
    failure
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
