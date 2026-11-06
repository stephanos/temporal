//! Explicit typed occurrences and witness policies for Activity strict probes.
use crate::activity_api::{self as api, catalog};
use crate::foundation::dc::Limits;
use crate::probes::Result;
use umpire_catalog::contextual_probe::{
    BaselineSelector, Context, Mutation, Registry, TypedBinding,
};
use umpire_catalog::{
    AsciiOverflow, Duration, GenerationError, ProbeCase, Violation, WitnessProfile,
};
use umpire_temporal_api::temporal::api::{failure::v1 as f, workflowservice::v1 as w};

fn text<L>(case: &ProbeCase) -> Mutation<L> {
    if case.violation.constraint == "string.max_bytes" {
        Mutation::Profile(WitnessProfile::AsciiMaxBytes(AsciiOverflow {
            byte: b'x',
            max_bytes: 16 * 1024 * 1024,
        }))
    } else {
        Mutation::Default
    }
}
fn time<L>(case: &ProbeCase) -> Mutation<L> {
    match case.violation.constraint.as_str() {
        "duration.valid" => {
            Mutation::Profile(WitnessProfile::InvalidDuration(Duration { seconds: 1, nanos: -1 }))
        }
        "duration.gte" => {
            Mutation::Profile(WitnessProfile::DurationGte(Duration { seconds: -1, nanos: 0 }))
        }
        _ => Mutation::Default,
    }
}
fn combined<L>(case: &ProbeCase) -> Mutation<L> {
    if case.violation.constraint == "string.max_bytes" { text(case) } else { time(case) }
}
fn default<L>(_: &ProbeCase) -> Mutation<L> {
    Mutation::Default
}
#[expect(
    clippy::unnecessary_wraps,
    reason = "Context::typed requires a fallible companion callback"
)]
fn none(_: &ProbeCase) -> Result<Vec<Violation>> {
    Ok(Vec::new())
}
fn is_request_id_overflow(case: &ProbeCase) -> bool {
    case.violation.constraint == "string.max_bytes" && case.violation.path.ends_with(".request_id")
}
fn start_companions(case: &ProbeCase) -> Result<Vec<Violation>> {
    if !is_request_id_overflow(case) {
        return Ok(Vec::new());
    }
    Ok(api::start_request_id_intent(&Limits::default())?
        .probe_cases()
        .into_iter()
        .map(|c| c.violation)
        .collect())
}
fn cancel_companions(case: &ProbeCase) -> Result<Vec<Violation>> {
    if !is_request_id_overflow(case) {
        return Ok(Vec::new());
    }
    Ok(api::cancel_request_id_intent(&Limits::default())?
        .probe_cases()
        .into_iter()
        .map(|c| c.violation)
        .collect())
}

macro_rules! root {
    ($contexts:ident; $id:literal, $method:ident, $pre:literal, $factory:path, $profile:ident, $companions:ident) => {{
        let mut binding = TypedBinding::identity(|| Ok($factory(&Limits::default())?));
        binding.mutation = $profile;
        $contexts.push(Context::typed(
            $id,
            concat!("/temporal.api.workflowservice.v1.WorkflowService/", stringify!($method)),
            $pre,
            binding,
            $companions,
        )?);
    }};
}
macro_rules! optional {
    ($contexts:ident; $id:literal, $method:ident, $pre:literal, $root:ty, $factory:path, $member:ident, $profile:ident) => {
        $contexts.push(Context::typed(
            $id,
            concat!("/temporal.api.workflowservice.v1.WorkflowService/", stringify!($method)),
            $pre,
            TypedBinding {
                catalog: || Ok($factory(&Limits::default())?),
                project: |r: &$root| r.$member.clone().ok_or(GenerationError::UnsupportedType),
                replace: |r: &mut $root, leaf| {
                    *r.$member.as_mut().ok_or(GenerationError::UnsupportedType)? = leaf;
                    Ok(())
                },
                mutation: $profile,
            },
            none,
        )?);
    };
}
macro_rules! variant {
    ($contexts:ident; $id:literal, $factory:path, $variant:ident, $profile:ident, $node:ident, [$($ancestor:ident),*]) => {
        $contexts.push(Context::typed($id, "/temporal.api.workflowservice.v1.WorkflowService/RespondActivityTaskFailed", "token",
            TypedBinding {
                catalog: || Ok($factory(&Limits::default())?),
                project: |r: &w::RespondActivityTaskFailedRequest| {
                    let $node = r.failure.as_ref().ok_or(GenerationError::UnsupportedType)?;
                    $(let $node = $node.$ancestor.as_ref().ok_or(GenerationError::UnsupportedType)?;)*
                    let info = &$node.failure_info;
                    match info.as_ref() { Some(f::failure::FailureInfo::$variant(v)) => Ok(v.clone()), _ => Err(GenerationError::UnsupportedType) }
                },
                replace: |r: &mut w::RespondActivityTaskFailedRequest, leaf| {
                    let $node = r.failure.as_mut().ok_or(GenerationError::UnsupportedType)?;
                    $(let $node = $node.$ancestor.as_mut().ok_or(GenerationError::UnsupportedType)?;)*
                    let info = &mut $node.failure_info;
                    match info.as_mut() { Some(f::failure::FailureInfo::$variant(v)) => { *v = leaf; Ok(()) }, _ => Err(GenerationError::UnsupportedType) }
                },
                mutation: $profile,
            }, none)?);
    };
}

fn start_contexts(contexts: &mut Vec<Context>) -> Result<()> {
    root!(contexts;
        "start",
        StartActivityExecution,
        "start",
        catalog::start_activity_execution_request,
        combined,
        start_companions
    );
    optional!(contexts;
        "start.type",
        StartActivityExecution,
        "start",
        w::StartActivityExecutionRequest,
        catalog::activity_type,
        activity_type,
        text
    );
    optional!(contexts;
        "start.queue",
        StartActivityExecution,
        "start",
        w::StartActivityExecutionRequest,
        catalog::task_queue,
        task_queue,
        default
    );
    root!(contexts;
        "start.retry_order",
        StartActivityExecution,
        "start",
        api::start_retry_order,
        default,
        none
    );
    optional!(contexts;
        "start.retry",
        StartActivityExecution,
        "start",
        w::StartActivityExecutionRequest,
        catalog::retry_policy,
        retry_policy,
        time
    );
    root!(contexts;
        "start.request_id",
        StartActivityExecution,
        "start",
        api::start_request_id_intent,
        default,
        none
    );
    Ok(())
}

fn client_contexts(contexts: &mut Vec<Context>) -> Result<()> {
    root!(contexts;
        "describe",
        DescribeActivityExecution,
        "scheduled",
        catalog::describe_activity_execution_request,
        text,
        none
    );
    root!(contexts;
        "poll_execution",
        PollActivityExecution,
        "completed",
        catalog::poll_activity_execution_request,
        text,
        none
    );
    root!(contexts;
        "cancel",
        RequestCancelActivityExecution,
        "scheduled",
        catalog::request_cancel_activity_execution_request,
        text,
        cancel_companions
    );
    root!(contexts;
        "cancel.request_id",
        RequestCancelActivityExecution,
        "scheduled",
        api::cancel_request_id_intent,
        default,
        none
    );
    Ok(())
}

fn queue_contexts(contexts: &mut Vec<Context>) -> Result<()> {
    root!(contexts;
        "poll_queue",
        PollActivityTaskQueue,
        "poll",
        catalog::poll_activity_task_queue_request,
        text,
        none
    );
    root!(contexts;
        "poll_queue.normal_name",
        PollActivityTaskQueue,
        "poll",
        api::poll_normal_name,
        default,
        none
    );
    optional!(contexts;
        "poll_queue.queue",
        PollActivityTaskQueue,
        "poll",
        w::PollActivityTaskQueueRequest,
        catalog::task_queue,
        task_queue,
        default
    );
    Ok(())
}

fn worker_contexts(contexts: &mut Vec<Context>) -> Result<()> {
    root!(contexts;
        "completed",
        RespondActivityTaskCompleted,
        "token",
        catalog::respond_activity_task_completed_request,
        text,
        none
    );
    root!(contexts;
        "failed",
        RespondActivityTaskFailed,
        "token",
        catalog::respond_activity_task_failed_request,
        text,
        none
    );
    variant!(contexts;
        "failed.application",
        catalog::application_failure_info,
        ApplicationFailureInfo,
        time,
        node,
        []
    );
    optional!(contexts;
        "failed.presence",
        RespondActivityTaskFailed,
        "token",
        w::RespondActivityTaskFailedRequest,
        api::worker_failure_presence,
        failure,
        default
    );
    variant!(contexts;
        "failed.cause.activity",
        catalog::activity_failure_info,
        ActivityFailureInfo,
        default,
        node,
        [cause]
    );
    variant!(contexts;
        "failed.cause.child",
        catalog::child_workflow_execution_failure_info,
        ChildWorkflowExecutionFailureInfo,
        default,
        node,
        [cause]
    );
    variant!(contexts;
        "failed.cause.timeout",
        catalog::timeout_failure_info,
        TimeoutFailureInfo,
        default,
        node,
        [cause]
    );
    variant!(contexts;
        "failed.cause.nexus",
        catalog::nexus_handler_failure_info,
        NexusHandlerFailureInfo,
        default,
        node,
        [cause]
    );
    root!(contexts;
        "canceled",
        RespondActivityTaskCanceled,
        "canceled_token",
        catalog::respond_activity_task_canceled_request,
        text,
        none
    );
    Ok(())
}

pub(crate) fn registry() -> Result<Registry> {
    let mut contexts = Vec::new();
    start_contexts(&mut contexts)?;
    client_contexts(&mut contexts)?;
    queue_contexts(&mut contexts)?;
    worker_contexts(&mut contexts)?;
    let mut declared = catalog::expected_probe_cases(&Limits::default())?;
    declared.extend(api::worker_failure_presence(&Limits::default())?.probe_cases());
    let baselines = [
        ("", "StartActivityExecution"),
        ("StartActivityExecution", "StartActivityExecution"),
        ("RespondActivityTaskCompleted", "RespondActivityTaskCompleted"),
        ("RespondActivityTaskCanceled", "RespondActivityTaskCanceled"),
        ("RequestCancelActivityExecution", "RequestCancelActivityExecution"),
    ]
    .into_iter()
    .map(|(selector, name)| BaselineSelector {
        selector,
        method: match name {
            "StartActivityExecution" => {
                "/temporal.api.workflowservice.v1.WorkflowService/StartActivityExecution"
            }
            "RespondActivityTaskCompleted" => {
                "/temporal.api.workflowservice.v1.WorkflowService/RespondActivityTaskCompleted"
            }
            "RespondActivityTaskCanceled" => {
                "/temporal.api.workflowservice.v1.WorkflowService/RespondActivityTaskCanceled"
            }
            _ => "/temporal.api.workflowservice.v1.WorkflowService/RequestCancelActivityExecution",
        },
        context: "",
    })
    .collect();
    Registry::new(contexts, baselines, &declared)
}
