//! Frozen baseline constructors from before entity-lens extraction.
use super::*;
use umpire_temporal_api::temporal::api::{
    common::v1::{ActivityType, RetryPolicy},
    failure::v1::{
        ActivityFailureInfo, ChildWorkflowExecutionFailureInfo, NexusHandlerFailureInfo,
        TimeoutFailureInfo,
    },
    workflowservice::v1::{
        DescribeActivityExecutionRequest, PollActivityExecutionRequest,
        PollActivityTaskQueueRequest, RequestCancelActivityExecutionRequest,
        RespondActivityTaskCanceledRequest, RespondActivityTaskCompletedRequest,
        RespondActivityTaskFailedRequest,
    },
};

fn check_slots(name: &str, slots: &ProbeRequest) -> Result<()> {
    if slots.namespace.is_empty() || slots.activity_id.is_empty() || slots.task_queue.is_empty() {
        return Err("missing namespace/activity/queue prerequisite".into());
    }
    if matches!(
        name,
        "DescribeActivityExecution" | "PollActivityExecution" | "RequestCancelActivityExecution"
    ) && slots.run_id.is_empty()
    {
        return Err("missing run prerequisite".into());
    }
    if name.starts_with("RespondActivityTask") && slots.task_token.is_empty() {
        return Err("missing live task token prerequisite".into());
    }
    Ok(())
}

fn baseline(name: &str, slots: &ProbeRequest, context: &str) -> Result<Vec<u8>> {
    check_slots(name, slots)?;
    let queue = TaskQueue { name: slots.task_queue.clone(), ..Default::default() };
    Ok(match name {
        "StartActivityExecution" => (StartActivityExecutionRequest {
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
        })
        .encode_to_vec(),
        "DescribeActivityExecution" => (DescribeActivityExecutionRequest {
            namespace: slots.namespace.clone(),
            activity_id: slots.activity_id.clone(),
            run_id: slots.run_id.clone(),
            ..Default::default()
        })
        .encode_to_vec(),
        "PollActivityExecution" => (PollActivityExecutionRequest {
            namespace: slots.namespace.clone(),
            activity_id: slots.activity_id.clone(),
            run_id: slots.run_id.clone(),
        })
        .encode_to_vec(),
        "RequestCancelActivityExecution" => (RequestCancelActivityExecutionRequest {
            namespace: slots.namespace.clone(),
            activity_id: slots.activity_id.clone(),
            run_id: slots.run_id.clone(),
            request_id: UUID.into(),
            identity: "umpire-probe".into(),
            ..Default::default()
        })
        .encode_to_vec(),
        "PollActivityTaskQueue" => (PollActivityTaskQueueRequest {
            namespace: slots.namespace.clone(),
            task_queue: Some(queue),
            identity: "umpire-probe".into(),
            ..Default::default()
        })
        .encode_to_vec(),
        "RespondActivityTaskCompleted" => (RespondActivityTaskCompletedRequest {
            namespace: slots.namespace.clone(),
            task_token: slots.task_token.clone(),
            identity: "umpire-probe".into(),
            ..Default::default()
        })
        .encode_to_vec(),
        "RespondActivityTaskFailed" => {
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
                failure.cause =
                    Some(Box::new(Failure { failure_info: info, ..Default::default() }));
            }
            (RespondActivityTaskFailedRequest {
                namespace: slots.namespace.clone(),
                task_token: slots.task_token.clone(),
                identity: "umpire-probe".into(),
                failure: Some(failure),
                ..Default::default()
            })
            .encode_to_vec()
        }
        "RespondActivityTaskCanceled" => (RespondActivityTaskCanceledRequest {
            namespace: slots.namespace.clone(),
            task_token: slots.task_token.clone(),
            identity: "umpire-probe".into(),
            ..Default::default()
        })
        .encode_to_vec(),
        _ => return Err("unknown prerequisite method".into()),
    })
}

#[test]
fn registered_probe_baselines_preserve_every_contextual_case() -> Result<()> {
    let slots = ProbeRequest {
        namespace: "namespace".into(),
        activity_id: "activity".into(),
        task_queue: "queue".into(),
        run_id: UUID.into(),
        task_token: vec![8, 1],
        ..Default::default()
    };
    for entry in inventory()? {
        assert_eq!(
            baseline(entry.method, &slots, entry.context)?,
            super::baseline(entry.method, &slots, entry.context)?,
            "{}",
            entry.id()
        );
    }
    for method in [
        "StartActivityExecution",
        "RespondActivityTaskCompleted",
        "RespondActivityTaskCanceled",
        "RequestCancelActivityExecution",
    ] {
        assert_eq!(baseline(method, &slots, "")?, super::baseline(method, &slots, "")?, "{method}");
    }
    for slots in [
        ProbeRequest::default(),
        ProbeRequest { run_id: String::new(), ..slots.clone() },
        ProbeRequest { task_token: Vec::new(), ..slots.clone() },
    ] {
        for method in
            ["StartActivityExecution", "DescribeActivityExecution", "RespondActivityTaskCompleted"]
        {
            let before = baseline(method, &slots, "").map_err(|error| error.to_string());
            let after = super::baseline(method, &slots, "").map_err(|error| error.to_string());
            assert_eq!(before, after, "{method}");
        }
    }
    Ok(())
}
