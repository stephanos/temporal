//! Stateless typed witnesses and value-free classification for activity RPCs.

use umpire_runtime::annotations::ProbeCase;
use umpire_runtime::annotations::ProbeResponse;
use umpire_runtime::probe_recipe as recipe;

use crate::foundation::dc::Limits;

use crate::activity_api::{
    cancel_request_id_intent, catalog, start_request_id_intent, worker_failure_presence,
};
use prost::Message;
use std::{collections::BTreeSet, sync::OnceLock};
use umpire_catalog::{
    Catalog, FieldDescriptor, FieldValue, Kind, ProbeCase as Target, Rng, Violation, validate,
};
use umpire_runtime::prost_types::Duration;
use umpire_temporal_api::{
    temporal::api::{
        common::v1::{ActivityType, Payload, Payloads, RetryPolicy},
        failure::v1::{
            ActivityFailureInfo, ApplicationFailureInfo, ChildWorkflowExecutionFailureInfo,
            Failure, NexusHandlerFailureInfo, TimeoutFailureInfo, failure::FailureInfo,
        },
        taskqueue::v1::TaskQueue,
        workflowservice::v1::{
            DescribeActivityExecutionRequest, PollActivityExecutionRequest,
            PollActivityTaskQueueRequest, RequestCancelActivityExecutionRequest,
            RespondActivityTaskCanceledRequest, RespondActivityTaskCompletedRequest,
            RespondActivityTaskFailedRequest, StartActivityExecutionRequest,
        },
    },
    umpire::temporal::v1::{ProbeRequest, probe_request::Operation},
};

mod dispositions;
mod generic;
mod report;

type Result<T> = std::result::Result<T, Box<dyn std::error::Error>>;
const UUID: &str = "12345678-1234-4234-8234-123456789abc";

fn cached<T>(cache: &OnceLock<T>, initialize: impl FnOnce() -> Result<T>) -> Result<&T> {
    if let Some(value) = cache.get() {
        return Ok(value);
    }
    let value = initialize()?;
    Ok(cache.get_or_init(|| value))
}

struct Entry {
    context: &'static str,
    method: &'static str,
    prerequisite: &'static str,
    target: Target,
    field: Option<FieldDescriptor>,
}

fn inventory() -> Result<&'static [Entry]> {
    static ENTRIES: OnceLock<Vec<Entry>> = OnceLock::new();
    Ok(cached(&ENTRIES, build_inventory)?.as_slice())
}

fn add<M: 'static>(
    entries: &mut Vec<Entry>,
    context: &'static str,
    method: &'static str,
    prerequisite: &'static str,
    catalog: &Catalog<M>,
) {
    for target in catalog.probe_cases() {
        let field = catalog.fields().find(|field| field.path == target.violation.path).cloned();
        entries.push(Entry { context, method, prerequisite, target, field });
    }
}

#[allow(
    clippy::too_many_lines,
    reason = "The contextual probe registry is one auditable inventory"
)]
fn build_inventory() -> Result<Vec<Entry>> {
    let mut entries = Vec::new();
    macro_rules! register {
        ($context:literal, $method:literal, $prerequisite:literal, $catalog:expr) => {
            add(&mut entries, $context, $method, $prerequisite, &$catalog?);
        };
    }
    register!(
        "start",
        "StartActivityExecution",
        "start",
        catalog::start_activity_execution_request(&Limits::default())
    );
    register!(
        "start.type",
        "StartActivityExecution",
        "start",
        catalog::activity_type(&Limits::default())
    );
    register!(
        "start.queue",
        "StartActivityExecution",
        "start",
        catalog::task_queue(&Limits::default())
    );
    register!(
        "start.retry_order",
        "StartActivityExecution",
        "start",
        crate::activity_api::start_retry_order(&Limits::default())
    );
    register!(
        "start.retry",
        "StartActivityExecution",
        "start",
        catalog::retry_policy(&Limits::default())
    );
    register!(
        "start.request_id",
        "StartActivityExecution",
        "start",
        start_request_id_intent(&Limits::default())
    );
    register!(
        "describe",
        "DescribeActivityExecution",
        "scheduled",
        catalog::describe_activity_execution_request(&Limits::default())
    );
    register!(
        "poll_execution",
        "PollActivityExecution",
        "completed",
        catalog::poll_activity_execution_request(&Limits::default())
    );
    register!(
        "cancel",
        "RequestCancelActivityExecution",
        "scheduled",
        catalog::request_cancel_activity_execution_request(&Limits::default())
    );
    register!(
        "cancel.request_id",
        "RequestCancelActivityExecution",
        "scheduled",
        cancel_request_id_intent(&Limits::default())
    );
    register!(
        "poll_queue",
        "PollActivityTaskQueue",
        "poll",
        catalog::poll_activity_task_queue_request(&Limits::default())
    );
    register!(
        "poll_queue.normal_name",
        "PollActivityTaskQueue",
        "poll",
        crate::activity_api::poll_normal_name(&Limits::default())
    );
    register!(
        "poll_queue.queue",
        "PollActivityTaskQueue",
        "poll",
        catalog::task_queue(&Limits::default())
    );
    register!(
        "completed",
        "RespondActivityTaskCompleted",
        "token",
        catalog::respond_activity_task_completed_request(&Limits::default())
    );
    register!(
        "failed",
        "RespondActivityTaskFailed",
        "token",
        catalog::respond_activity_task_failed_request(&Limits::default())
    );
    register!(
        "failed.application",
        "RespondActivityTaskFailed",
        "token",
        catalog::application_failure_info(&Limits::default())
    );
    register!(
        "failed.presence",
        "RespondActivityTaskFailed",
        "token",
        worker_failure_presence(&Limits::default())
    );
    register!(
        "failed.cause.activity",
        "RespondActivityTaskFailed",
        "token",
        catalog::activity_failure_info(&Limits::default())
    );
    register!(
        "failed.cause.child",
        "RespondActivityTaskFailed",
        "token",
        catalog::child_workflow_execution_failure_info(&Limits::default())
    );
    register!(
        "failed.cause.timeout",
        "RespondActivityTaskFailed",
        "token",
        catalog::timeout_failure_info(&Limits::default())
    );
    register!(
        "failed.cause.nexus",
        "RespondActivityTaskFailed",
        "token",
        catalog::nexus_handler_failure_info(&Limits::default())
    );
    register!(
        "canceled",
        "RespondActivityTaskCanceled",
        "canceled_token",
        catalog::respond_activity_task_canceled_request(&Limits::default())
    );
    entries.sort_by_key(Entry::id);
    verify_inventory(&entries)?;
    Ok(entries)
}

fn verify_inventory(entries: &[Entry]) -> Result<()> {
    let ids: BTreeSet<_> = entries.iter().map(Entry::id).collect();
    if ids.len() != entries.len() {
        return Err("duplicate contextual probe identity".into());
    }
    let mut declared = catalog::expected_probe_cases(&Limits::default())?;
    declared.extend(worker_failure_presence(&Limits::default())?.probe_cases());
    let registered: BTreeSet<_> = entries.iter().map(|entry| entry.target.id.clone()).collect();
    umpire_catalog::verify_probe_coverage(&declared, &registered.into_iter().collect::<Vec<_>>())?;
    Ok(())
}

impl Entry {
    fn id(&self) -> String {
        format!("{}::{}", self.context, self.target.id)
    }
    fn companions(&self) -> Result<Vec<Violation>> {
        if self.target.violation.constraint != "string.max_bytes"
            || !self.target.violation.path.ends_with(".request_id")
        {
            return Ok(Vec::new());
        }
        let targets = match self.context {
            "start" => start_request_id_intent(&Limits::default())?.probe_cases(),
            "cancel" => cancel_request_id_intent(&Limits::default())?.probe_cases(),
            _ => return Ok(Vec::new()),
        };
        Ok(targets.into_iter().map(|case| case.violation).collect())
    }
    fn metadata(&self) -> Result<ProbeCase> {
        let expected = self.target.violation.expected.as_ref().ok_or("missing expectation")?;
        let mut case = ProbeCase {
            id: self.id(),
            method: method(self.method),
            prerequisite: self.prerequisite.into(),
            ..Default::default()
        };
        let prediction =
            umpire_catalog::generic_probe::expectation(case.id.clone(), &expected.expectation)?;
        case.accepts = prediction.accepts;
        case.code = prediction.code;
        case.message_pattern = prediction.message_pattern;
        case.details_type = prediction.details_type;
        Ok(case)
    }
    fn negative<T: FieldValue>(&self, value: &T) -> Result<T> {
        let field = self.field.as_ref().ok_or("missing field descriptor")?;
        Ok(value.negative(
            &field.kind,
            field.repeated.as_ref(),
            &self.target.violation.constraint,
            &mut Rng::new(217),
        )?)
    }
    fn negative_text(&self, value: &String) -> Result<String> {
        if self.target.violation.constraint == "string.max_bytes" {
            let field = self.field.as_ref().ok_or("missing field descriptor")?;
            if let Kind::Text(rules) | Kind::Identifier(rules) = &field.kind {
                let size = rules
                    .max_bytes
                    .ok_or("missing byte maximum")?
                    .checked_add(1)
                    .ok_or("byte maximum overflow")?;
                if size > 16 * 1024 * 1024 {
                    return Err("string witness exceeds probe budget".into());
                }
                return Ok("x".repeat(size));
            }
        }
        self.negative(value)
    }
}

fn method(name: &str) -> String {
    format!("/temporal.api.workflowservice.v1.WorkflowService/{name}")
}
fn duration(seconds: i64) -> Duration {
    Duration { seconds, nanos: 0 }
}
fn application() -> Failure {
    Failure {
        failure_info: Some(FailureInfo::ApplicationFailureInfo(ApplicationFailureInfo {
            r#type: "ProbeFailure".into(),
            ..Default::default()
        })),
        ..Default::default()
    }
}

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

fn request_findings(name: &str, bytes: &[u8]) -> Result<Vec<Violation>> {
    macro_rules! supplemental {
        (RespondActivityTaskFailed, $message:ident, $limits:expr, $findings:ident) => {
            if let Some(failure) = &$message.failure {
                $findings.extend(validate(failure, &worker_failure_presence($limits)?));
            }
        };
        ($method:ident, $message:ident, $limits:expr, $findings:ident) => {};
    }
    let method =
        umpire_temporal_api::WorkflowServiceMethod::parse(name).ok_or("unknown request method")?;
    umpire_temporal_api::workflow_request_findings!(method, bytes, &Limits::default(), supplemental)
}

fn bad_duration(entry: &Entry) -> Duration {
    if entry.target.violation.constraint == "duration.valid" {
        Duration { seconds: 1, nanos: -1 }
    } else {
        Duration { seconds: -1, nanos: 0 }
    }
}

fn mutate(bytes: &[u8], entry: &Entry) -> Result<Vec<u8>> {
    use umpire_catalog::WireMutation;
    use umpire_temporal_api::{WorkflowServiceMethod, mutate_workflow_request};
    let method = WorkflowServiceMethod::parse(entry.method).ok_or("unknown request method")?;
    let field = entry.target.violation.path.rsplit('.').next().ok_or("missing field path")?;
    if let Some(path) = &entry.target.omitted_field
        && entry.target.violation.constraint == "required"
        && path.starts_with("temporal.api.workflowservice.v1.")
    {
        return Ok(mutate_workflow_request(method, bytes, path, &WireMutation::Absent)?);
    }
    if field == "namespace" {
        let Some(FieldDescriptor { kind: Kind::Text(rules), .. }) = &entry.field else {
            return Err("namespace kind missing".into());
        };
        let limit = rules.max_bytes.ok_or("namespace byte maximum missing")?;
        let path = format!("temporal.api.workflowservice.v1.{}Request.namespace", method.name());
        return Ok(mutate_workflow_request(
            method,
            bytes,
            &path,
            &WireMutation::LimitPlusOne { limit: limit.try_into()? },
        )?);
    }
    macro_rules! text {
        ($value:expr) => {
            $value = entry.negative_text(&$value)?
        };
    }
    macro_rules! mutate_typed {
        (StartActivityExecution, $message:ident, $entry:ident, $field:ident) => {{
            mutate_start(&mut $message, $field, $entry)?;
            Ok($message.encode_to_vec())
        }};
        (DescribeActivityExecution, $message:ident, $entry:ident, $field:ident) => {{
            match $field {
                "activity_id" => text!($message.activity_id),
                "run_id" => text!($message.run_id),
                "DescribeActivityExecutionRequest" => {
                    $message.long_poll_token = vec![1];
                    $message.run_id.clear();
                }
                _ => return Err("unsupported describe field".into()),
            };
            Ok($message.encode_to_vec())
        }};
        (PollActivityExecution, $message:ident, $entry:ident, $field:ident) => {{
            match $field {
                "activity_id" => text!($message.activity_id),
                "run_id" => text!($message.run_id),
                _ => return Err("unsupported poll field".into()),
            };
            Ok($message.encode_to_vec())
        }};
        (RequestCancelActivityExecution, $message:ident, $entry:ident, $field:ident) => {{
            match $field {
                "activity_id" => text!($message.activity_id),
                "run_id" => text!($message.run_id),
                "request_id" => text!($message.request_id),
                "identity" => text!($message.identity),
                "reason" => text!($message.reason),
                _ => return Err("unsupported cancel field".into()),
            };
            Ok($message.encode_to_vec())
        }};
        (PollActivityTaskQueue, $message:ident, $entry:ident, $field:ident) => {{
            if $entry.context == "poll_queue.normal_name" {
                let queue = $message.task_queue.as_mut().ok_or("missing queue")?;
                queue.normal_name = $entry.negative_text(&queue.normal_name)?;
            } else if $entry.context == "poll_queue.queue" {
                mutate_queue($message.task_queue.as_mut().ok_or("missing queue")?, $field, $entry)?;
            } else {
                match $field {
                    "identity" => text!($message.identity),
                    _ => return Err("unsupported queue field".into()),
                }
            }
            Ok($message.encode_to_vec())
        }};
        (RespondActivityTaskCompleted, $message:ident, $entry:ident, $field:ident) => {{
            text!($message.identity);
            Ok($message.encode_to_vec())
        }};
        (RespondActivityTaskCanceled, $message:ident, $entry:ident, $field:ident) => {{
            text!($message.identity);
            Ok($message.encode_to_vec())
        }};
        (RespondActivityTaskFailed, $message:ident, $entry:ident, $field:ident) => {{
            mutate_failed(&mut $message, $field, $entry)?;
            Ok($message.encode_to_vec())
        }};
        ($method:ident, $message:ident, $entry:ident, $field:ident) => {{ Err("unsupported contextual mutation".into()) }};
    }
    umpire_temporal_api::dispatch_workflow_request!(method, bytes, mutate_typed, entry, field)
}

fn mutate_failed(
    m: &mut RespondActivityTaskFailedRequest,
    field: &str,
    entry: &Entry,
) -> Result<()> {
    match entry.context {
        "failed" => match field {
            "identity" => m.identity = entry.negative_text(&m.identity)?,
            "cause" => m.cause = entry.negative(&m.cause)?,
            _ => return Err("unsupported failed field".into()),
        },
        "failed.presence" => m.failure.as_mut().ok_or("missing failure")?.failure_info = None,
        "failed.application" => {
            let Some(FailureInfo::ApplicationFailureInfo(v)) =
                &mut m.failure.as_mut().ok_or("missing failure")?.failure_info
            else {
                return Err("missing application info".into());
            };
            match field {
                "next_retry_delay" => v.next_retry_delay = Some(bad_duration(entry)),
                "category" => v.category = entry.negative(&v.category)?,
                _ => return Err("unsupported application field".into()),
            }
        }
        _ => {
            let info = &mut m
                .failure
                .as_mut()
                .ok_or("missing failure")?
                .cause
                .as_mut()
                .ok_or("missing cause")?
                .failure_info;
            match info {
                Some(FailureInfo::ActivityFailureInfo(v)) => {
                    v.retry_state = entry.negative(&v.retry_state)?;
                }
                Some(FailureInfo::ChildWorkflowExecutionFailureInfo(v)) => {
                    v.retry_state = entry.negative(&v.retry_state)?;
                }
                Some(FailureInfo::TimeoutFailureInfo(v)) => {
                    v.timeout_type = entry.negative(&v.timeout_type)?;
                }
                Some(FailureInfo::NexusHandlerFailureInfo(v)) => {
                    v.retry_behavior = entry.negative(&v.retry_behavior)?;
                }
                _ => return Err("missing nested failure variant".into()),
            }
        }
    }
    Ok(())
}

fn mutate_start(m: &mut StartActivityExecutionRequest, field: &str, entry: &Entry) -> Result<()> {
    macro_rules! text {
        ($value:expr) => {
            $value = entry.negative_text(&$value)?
        };
    }
    match entry.context {
        "start.type" => {
            let v = m.activity_type.as_mut().ok_or("missing type")?;
            text!(v.name);
        }
        "start.queue" => {
            mutate_queue(m.task_queue.as_mut().ok_or("missing queue")?, field, entry)?;
        }
        "start.retry_order" => {
            crate::foundation::views::start_retry_order::set(m, 0)?;
        }
        "start.retry" => {
            let v = m.retry_policy.as_mut().ok_or("missing retry")?;
            match field {
                "initial_interval" => v.initial_interval = Some(bad_duration(entry)),
                "maximum_interval" => v.maximum_interval = Some(bad_duration(entry)),
                "backoff_coefficient" => v.backoff_coefficient = 0.5,
                "maximum_attempts" => {
                    v.maximum_attempts = entry.negative(&v.maximum_attempts)?;
                }
                _ => return Err("unsupported retry field".into()),
            }
        }
        "start.request_id" => text!(m.request_id),
        _ => match field {
            "identity" => text!(m.identity),
            "request_id" => text!(m.request_id),
            "activity_id" => text!(m.activity_id),
            "schedule_to_close_timeout" => m.schedule_to_close_timeout = Some(bad_duration(entry)),
            "schedule_to_start_timeout" => m.schedule_to_start_timeout = Some(bad_duration(entry)),
            "start_to_close_timeout" => m.start_to_close_timeout = Some(bad_duration(entry)),
            "heartbeat_timeout" => m.heartbeat_timeout = Some(bad_duration(entry)),
            "start_delay" => m.start_delay = Some(bad_duration(entry)),
            "id_reuse_policy" => m.id_reuse_policy = entry.negative(&m.id_reuse_policy)?,
            "id_conflict_policy" => {
                m.id_conflict_policy = entry.negative(&m.id_conflict_policy)?;
            }
            "input" => {
                let Some(FieldDescriptor { kind: Kind::Bytes(rules), .. }) = &entry.field else {
                    return Err("input kind missing".into());
                };
                let size = rules
                    .max_len
                    .ok_or("input maximum missing")?
                    .checked_add(1)
                    .filter(|size| *size <= 16 * 1024 * 1024)
                    .ok_or("input witness exceeds probe budget")?;
                let data_size =
                    size.checked_sub(10).ok_or("input bound too small for payload witness")?;
                let mut payloads = Payloads {
                    payloads: vec![Payload { data: vec![b'x'; data_size], ..Default::default() }],
                };
                let padding = size.checked_sub(payloads.encoded_len()).ok_or("payload overhead")?;
                payloads.payloads[0].data.extend(std::iter::repeat_n(b'x', padding));
                if payloads.encoded_len() != size {
                    return Err("payload exact size mismatch".into());
                }
                m.input = Some(payloads);
            }
            "StartActivityExecutionRequest" => match entry.target.violation.constraint.as_str() {
                "any_of" => {
                    m.schedule_to_close_timeout = None;
                    m.start_to_close_timeout = None;
                }
                _ => return Err("unsupported start rule".into()),
            },
            _ => return Err("unsupported start field".into()),
        },
    }
    Ok(())
}

fn mutate_queue(queue: &mut TaskQueue, field: &str, entry: &Entry) -> Result<()> {
    match field {
        "name" => queue.name = entry.negative(&queue.name)?,
        "kind" => queue.kind = entry.negative(&queue.kind)?,
        _ => return Err("unsupported task queue field".into()),
    }
    Ok(())
}

/// Enumerates, prepares and classifies activity probes independently of Model state.
///
/// # Errors
/// Refuses missing prerequisites, unsupported mutations, invalid baselines/witnesses and incomplete observations.
pub fn probe(request: &ProbeRequest) -> Result<ProbeResponse> {
    if request.encoded_len() > 1024 * 1024 - 1024
        || request.observations.len() > umpire_catalog::generic_probe::MAX_CASES
        || request.case_id.len() > 4096
        || request.observations.iter().any(|observation| {
            observation.case_id.len() > 4096
                || observation.predictions.len() > umpire_catalog::generic_probe::MAX_PREDICTIONS
                || observation.details_type.len() > 512
                || observation.refusal.len() > 256
        })
    {
        return Err("probe request exceeds metadata budget".into());
    }
    let operation = match Operation::try_from(request.operation)? {
        Operation::Inventory => umpire_catalog::generic_probe::Request::Inventory,
        Operation::Prepare => umpire_catalog::generic_probe::Request::Prepare {
            id: &request.case_id,
            baseline: request.baseline,
            context: request,
        },
        Operation::Classify => {
            umpire_catalog::generic_probe::Request::Classify(&request.observations)
        }
    };
    generic::engine()?.run(&operation)
}

fn prepare_strict(request: &ProbeRequest, entries: &[Entry]) -> Result<ProbeResponse> {
    let entry = entries.iter().find(|entry| entry.id() == request.case_id);
    let selector = if request.case_id.is_empty() {
        "StartActivityExecution"
    } else {
        request.case_id.as_str()
    };
    let (name, context) = if let Some(entry) = entry {
        (entry.method, entry.context)
    } else if request.baseline
        && matches!(
            selector,
            "StartActivityExecution"
                | "RespondActivityTaskCompleted"
                | "RespondActivityTaskCanceled"
                | "RequestCancelActivityExecution"
        )
    {
        (selector, "")
    } else {
        return Err("unknown probe case".into());
    };
    let mut bytes = baseline(name, request, context)?;
    if !request_findings(name, &bytes)?.is_empty() {
        return Err("invalid baseline".into());
    }
    if !request.baseline {
        let entry = entry.ok_or("missing target")?;
        bytes = mutate(&bytes, entry)?;
        let mut findings = request_findings(name, &bytes)?;
        let mut expected = entry.companions()?;
        expected.push(entry.target.violation.clone());
        findings.sort_by_key(|finding| (finding.path.clone(), finding.constraint.clone()));
        expected.sort_by_key(|finding| (finding.path.clone(), finding.constraint.clone()));
        if findings != expected {
            return Err(format!(
                "{}: witness does not isolate declared target: {findings:?}",
                entry.id()
            )
            .into());
        }
    }
    let mut response = ProbeResponse { method: method(name), ..Default::default() };
    if bytes.len() > 512 * 1024 {
        response.recipe = Some(recipe(&bytes)?);
    } else {
        response.message = bytes;
    }
    Ok(response)
}

#[cfg(test)]
mod tests {
    use super::{Result, cached, generic, inventory};
    use std::sync::{Barrier, OnceLock};

    #[test]
    fn generated_findings_preserve_contextual_intents_and_recursive_checks() -> Result<()> {
        use super::{
            Failure, FailureInfo, ProbeRequest, RequestCancelActivityExecutionRequest,
            RespondActivityTaskFailedRequest, StartActivityExecutionRequest, TimeoutFailureInfo,
            UUID, application, baseline, duration, request_findings,
        };
        use prost::Message;
        let slots = ProbeRequest {
            namespace: "namespace".into(),
            activity_id: "activity".into(),
            task_queue: "queue".into(),
            run_id: UUID.into(),
            task_token: vec![1],
            ..Default::default()
        };
        for method in ["StartActivityExecution", "RequestCancelActivityExecution"] {
            let bytes = baseline(method, &slots, "")?;
            let bytes = if method == "StartActivityExecution" {
                let mut request = StartActivityExecutionRequest::decode(bytes.as_slice())?;
                request.request_id = "invalid-uuid".into();
                request.encode_to_vec()
            } else {
                let mut request = RequestCancelActivityExecutionRequest::decode(bytes.as_slice())?;
                request.request_id = "invalid-uuid".into();
                request.encode_to_vec()
            };
            let findings = request_findings(method, &bytes)?;
            assert_eq!(findings.len(), 1, "{method}: {findings:?}");
            assert_eq!(findings[0].constraint, "string.uuid");
        }
        let mut request = RespondActivityTaskFailedRequest::decode(
            baseline("RespondActivityTaskFailed", &slots, "")?.as_slice(),
        )?;
        let mut failure = application();
        failure.cause = Some(Box::new(Failure {
            failure_info: Some(FailureInfo::TimeoutFailureInfo(TimeoutFailureInfo {
                timeout_type: i32::MAX,
                ..Default::default()
            })),
            ..Default::default()
        }));
        request.failure.as_mut().ok_or("missing failure")?.cause = Some(Box::new(failure));
        let findings = request_findings("RespondActivityTaskFailed", &request.encode_to_vec())?;
        assert_eq!(findings.len(), 1, "{findings:?}");
        assert!(findings[0].path.ends_with("TimeoutFailureInfo.timeout_type"));
        request.failure.as_mut().ok_or("missing failure")?.failure_info = None;
        let findings = request_findings("RespondActivityTaskFailed", &request.encode_to_vec())?;
        assert_eq!(findings.len(), 2, "{findings:?}");
        assert!(findings.iter().any(|finding| finding.constraint == "required"));
        let mut request = StartActivityExecutionRequest::decode(
            baseline("StartActivityExecution", &slots, "")?.as_slice(),
        )?;
        let retry = request.retry_policy.as_mut().ok_or("missing retry")?;
        retry.maximum_attempts = 1;
        retry.initial_interval = Some(duration(-1));
        retry.maximum_interval = Some(duration(-1));
        retry.backoff_coefficient = 0.5;
        assert_eq!(request_findings("StartActivityExecution", &request.encode_to_vec())?, []);
        request.retry_policy.as_mut().ok_or("missing retry")?.maximum_attempts = 2;
        assert_eq!(request_findings("StartActivityExecution", &request.encode_to_vec())?.len(), 3);
        for (method, bytes) in [
            ("unknown", &[][..]),
            ("GetSystemInfo", &[][..]),
            ("StartActivityExecution", &[0xff][..]),
        ] {
            assert!(request_findings(method, bytes).is_err());
        }
        Ok(())
    }

    #[test]
    fn validated_inventories_are_reused() -> Result<()> {
        let contextual = inventory()?;
        let contextual_again = inventory()?;
        let generic = generic::engine()?;
        let generic_again = generic::engine()?;
        assert!(!contextual.is_empty());
        assert!(
            generic
                .run(&umpire_catalog::generic_probe::Request::Inventory)?
                .cases
                .iter()
                .any(|case| case.generic)
        );
        assert_eq!(
            (contextual.as_ptr(), std::ptr::from_ref(generic)),
            (contextual_again.as_ptr(), std::ptr::from_ref(generic_again)),
            "successfully validated metadata must not be rebuilt for each probe"
        );
        Ok(())
    }

    #[test]
    fn failed_or_panicked_initialization_publishes_nothing() -> Result<()> {
        let cache = OnceLock::new();
        let failed = cached(&cache, || -> Result<Vec<u8>> {
            let mut partial = vec![1];
            partial.push(2);
            Err("metadata validation refused".into())
        });
        assert_eq!(
            failed.err().ok_or("missing initialization error")?.to_string(),
            "metadata validation refused"
        );
        assert!(cache.get().is_none());
        let panicked = std::panic::catch_unwind(|| {
            cached(&cache, || -> Result<Vec<u8>> {
                std::panic::resume_unwind(Box::new("initializer panic"))
            })
        });
        assert!(panicked.is_err());
        assert!(cache.get().is_none());
        assert_eq!(cached(&cache, || Ok(vec![3]))?, &[3]);
        assert_eq!(cached(&cache, || Err("must not rerun".into()))?, &[3]);
        Ok(())
    }

    #[test]
    fn concurrent_initialization_publishes_one_complete_inventory() -> Result<()> {
        let cache = OnceLock::new();
        let barrier = Barrier::new(4);
        std::thread::scope(|scope| -> Result<()> {
            let handles: Vec<_> = (0..4)
                .map(|_| {
                    scope.spawn(|| {
                        cached(&cache, || {
                            let entries = super::build_inventory()?;
                            barrier.wait();
                            Ok(entries)
                        })
                        .map(|entries| entries.as_ptr() as usize)
                        .map_err(|error| error.to_string())
                    })
                })
                .collect();
            let addresses = handles
                .into_iter()
                .map(|handle| {
                    handle
                        .join()
                        .map_err(|_| "inventory initialization thread panicked")?
                        .map_err(Into::into)
                })
                .collect::<Result<Vec<_>>>()?;
            assert!(addresses.iter().all(|address| *address == addresses[0]));
            assert!(!cache.get().ok_or("missing complete inventory")?.is_empty());
            Ok(())
        })
    }
}
