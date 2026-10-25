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
    Catalog, Expectation, FieldDescriptor, FieldValue, Kind, ProbeCase as Target, Rng, Violation,
    validate,
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
        match &expected.expectation {
            Expectation::Accepts => case.accepts = true,
            Expectation::RejectsCode(code) => case.code = *code as i32,
            Expectation::RejectsAnyStatus => {}
            Expectation::Rejects { code, message_pattern, details_type } => {
                case.code = *code;
                case.message_pattern.clone_from(message_pattern);
                case.details_type = details_type.clone().unwrap_or_default();
            }
            Expectation::Unobservable => {
                return Err("observable request case declared unobservable".into());
            }
        }
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

#[derive(Clone)]
enum Wire {
    Start(StartActivityExecutionRequest),
    Describe(DescribeActivityExecutionRequest),
    Poll(PollActivityExecutionRequest),
    Cancel(RequestCancelActivityExecutionRequest),
    Queue(PollActivityTaskQueueRequest),
    Completed(RespondActivityTaskCompletedRequest),
    Failed(RespondActivityTaskFailedRequest),
    Canceled(RespondActivityTaskCanceledRequest),
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

fn baseline(name: &str, slots: &ProbeRequest, context: &str) -> Result<Wire> {
    check_slots(name, slots)?;
    let queue = TaskQueue { name: slots.task_queue.clone(), ..Default::default() };
    Ok(match name {
        "StartActivityExecution" => Wire::Start(StartActivityExecutionRequest {
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
        }),
        "DescribeActivityExecution" => Wire::Describe(DescribeActivityExecutionRequest {
            namespace: slots.namespace.clone(),
            activity_id: slots.activity_id.clone(),
            run_id: slots.run_id.clone(),
            ..Default::default()
        }),
        "PollActivityExecution" => Wire::Poll(PollActivityExecutionRequest {
            namespace: slots.namespace.clone(),
            activity_id: slots.activity_id.clone(),
            run_id: slots.run_id.clone(),
        }),
        "RequestCancelActivityExecution" => Wire::Cancel(RequestCancelActivityExecutionRequest {
            namespace: slots.namespace.clone(),
            activity_id: slots.activity_id.clone(),
            run_id: slots.run_id.clone(),
            request_id: UUID.into(),
            identity: "umpire-probe".into(),
            ..Default::default()
        }),
        "PollActivityTaskQueue" => Wire::Queue(PollActivityTaskQueueRequest {
            namespace: slots.namespace.clone(),
            task_queue: Some(queue),
            identity: "umpire-probe".into(),
            ..Default::default()
        }),
        "RespondActivityTaskCompleted" => Wire::Completed(RespondActivityTaskCompletedRequest {
            namespace: slots.namespace.clone(),
            task_token: slots.task_token.clone(),
            identity: "umpire-probe".into(),
            ..Default::default()
        }),
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
            Wire::Failed(RespondActivityTaskFailedRequest {
                namespace: slots.namespace.clone(),
                task_token: slots.task_token.clone(),
                identity: "umpire-probe".into(),
                failure: Some(failure),
                ..Default::default()
            })
        }
        "RespondActivityTaskCanceled" => Wire::Canceled(RespondActivityTaskCanceledRequest {
            namespace: slots.namespace.clone(),
            task_token: slots.task_token.clone(),
            identity: "umpire-probe".into(),
            ..Default::default()
        }),
        _ => return Err("unknown prerequisite method".into()),
    })
}

impl Wire {
    fn bytes(&self) -> Vec<u8> {
        match self {
            Self::Start(m) => m.encode_to_vec(),
            Self::Describe(m) => m.encode_to_vec(),
            Self::Poll(m) => m.encode_to_vec(),
            Self::Cancel(m) => m.encode_to_vec(),
            Self::Queue(m) => m.encode_to_vec(),
            Self::Completed(m) => m.encode_to_vec(),
            Self::Failed(m) => m.encode_to_vec(),
            Self::Canceled(m) => m.encode_to_vec(),
        }
    }
    fn decoded(&self) -> Result<Self> {
        let bytes = self.bytes();
        self.decoded_from(&bytes)
    }
    fn decoded_from(&self, bytes: &[u8]) -> Result<Self> {
        Ok(match self {
            Self::Start(_) => Self::Start(StartActivityExecutionRequest::decode(bytes)?),
            Self::Describe(_) => Self::Describe(DescribeActivityExecutionRequest::decode(bytes)?),
            Self::Poll(_) => Self::Poll(PollActivityExecutionRequest::decode(bytes)?),
            Self::Cancel(_) => Self::Cancel(RequestCancelActivityExecutionRequest::decode(bytes)?),
            Self::Queue(_) => Self::Queue(PollActivityTaskQueueRequest::decode(bytes)?),
            Self::Completed(_) => {
                Self::Completed(RespondActivityTaskCompletedRequest::decode(bytes)?)
            }
            Self::Failed(_) => Self::Failed(RespondActivityTaskFailedRequest::decode(bytes)?),
            Self::Canceled(_) => Self::Canceled(RespondActivityTaskCanceledRequest::decode(bytes)?),
        })
    }
    fn findings(&self) -> Result<Vec<Violation>> {
        let mut findings = Vec::new();
        macro_rules! check {
            ($value:expr, $catalog:expr) => {{
                findings.extend(validate($value, &$catalog?));
            }};
        }
        match self {
            Self::Start(m) => {
                check!(m, catalog::start_activity_execution_request(&Limits::default()));
                check!(m, start_request_id_intent(&Limits::default()));
                if let Some(v) = &m.activity_type {
                    check!(v, catalog::activity_type(&Limits::default()));
                }
                if let Some(v) = &m.task_queue {
                    check!(v, catalog::task_queue(&Limits::default()));
                }
                if let Some(v) = &m.retry_policy {
                    check!(v, catalog::retry_policy(&Limits::default()));
                }
            }
            Self::Describe(m) => {
                check!(m, catalog::describe_activity_execution_request(&Limits::default()));
            }
            Self::Poll(m) => {
                check!(m, catalog::poll_activity_execution_request(&Limits::default()));
            }
            Self::Cancel(m) => {
                check!(m, catalog::request_cancel_activity_execution_request(&Limits::default()));
                check!(m, cancel_request_id_intent(&Limits::default()));
            }
            Self::Queue(m) => {
                check!(m, catalog::poll_activity_task_queue_request(&Limits::default()));
                if let Some(v) = &m.task_queue {
                    check!(v, catalog::task_queue(&Limits::default()));
                }
            }
            Self::Completed(m) => {
                check!(m, catalog::respond_activity_task_completed_request(&Limits::default()));
            }
            Self::Failed(m) => {
                check!(m, catalog::respond_activity_task_failed_request(&Limits::default()));
                if let Some(v) = &m.failure {
                    check!(v, worker_failure_presence(&Limits::default()));
                    failure_findings(v, &mut findings)?;
                }
            }
            Self::Canceled(m) => {
                check!(m, catalog::respond_activity_task_canceled_request(&Limits::default()));
            }
        }
        for finding in &mut findings {
            finding.fields.sort();
        }
        Ok(findings)
    }
}

fn failure_findings(value: &Failure, findings: &mut Vec<Violation>) -> Result<()> {
    findings.extend(validate(value, &catalog::failure(&Limits::default())?));
    macro_rules! check {
        ($value:expr, $catalog:expr) => {{
            findings.extend(validate($value, &$catalog?));
        }};
    }
    match &value.failure_info {
        Some(FailureInfo::ApplicationFailureInfo(v)) => {
            check!(v, catalog::application_failure_info(&Limits::default()));
        }
        Some(FailureInfo::ActivityFailureInfo(v)) => {
            check!(v, catalog::activity_failure_info(&Limits::default()));
        }
        Some(FailureInfo::ChildWorkflowExecutionFailureInfo(v)) => {
            check!(v, catalog::child_workflow_execution_failure_info(&Limits::default()));
        }
        Some(FailureInfo::TimeoutFailureInfo(v)) => {
            check!(v, catalog::timeout_failure_info(&Limits::default()));
        }
        Some(FailureInfo::NexusHandlerFailureInfo(v)) => {
            check!(v, catalog::nexus_handler_failure_info(&Limits::default()));
        }
        _ => (),
    }
    if let Some(cause) = &value.cause {
        failure_findings(cause, findings)?;
    }
    Ok(())
}

fn bad_duration(entry: &Entry) -> Duration {
    if entry.target.violation.constraint == "duration.valid" {
        Duration { seconds: 1, nanos: -1 }
    } else {
        Duration { seconds: -1, nanos: 0 }
    }
}

fn mutate_admission_fields(wire: &mut Wire, entry: &Entry, field: &str) -> Result<bool> {
    let omitted = entry.target.omitted_field.as_deref().and_then(|path| path.rsplit('.').next());
    if field == "namespace" || omitted == Some("namespace") {
        let namespace = match wire {
            Wire::Start(m) => &mut m.namespace,
            Wire::Describe(m) => &mut m.namespace,
            Wire::Poll(m) => &mut m.namespace,
            Wire::Cancel(m) => &mut m.namespace,
            Wire::Queue(m) => &mut m.namespace,
            Wire::Completed(m) => &mut m.namespace,
            Wire::Canceled(m) => &mut m.namespace,
            Wire::Failed(m) => &mut m.namespace,
        };
        if omitted.is_some() {
            namespace.clear();
        } else {
            *namespace = entry.negative_text(namespace)?;
        }
        return Ok(true);
    }
    if omitted == Some("task_token") {
        match wire {
            Wire::Completed(m) => m.task_token.clear(),
            Wire::Canceled(m) => m.task_token.clear(),
            Wire::Failed(m) => m.task_token.clear(),
            _ => return Err("unsupported task token omission".into()),
        }
        return Ok(true);
    }
    Ok(false)
}

fn mutate(wire: &mut Wire, entry: &Entry) -> Result<()> {
    let field = entry.target.violation.path.rsplit('.').next().ok_or("missing field path")?;
    if mutate_admission_fields(wire, entry, field)? {
        return Ok(());
    }
    macro_rules! text {
        ($value:expr) => {
            $value = entry.negative_text(&$value)?
        };
    }
    match wire {
        Wire::Start(m) => mutate_start(m, field, entry)?,
        Wire::Describe(m) => match field {
            "activity_id" => text!(m.activity_id),
            "run_id" => text!(m.run_id),
            "DescribeActivityExecutionRequest" => {
                m.long_poll_token = vec![1];
                m.run_id.clear();
            }
            _ => return Err("unsupported describe field".into()),
        },
        Wire::Poll(m) => match field {
            "activity_id" => text!(m.activity_id),
            "run_id" => text!(m.run_id),
            _ => return Err("unsupported poll field".into()),
        },
        Wire::Cancel(m) => match field {
            "activity_id" => text!(m.activity_id),
            "run_id" => text!(m.run_id),
            "request_id" => text!(m.request_id),
            "identity" => text!(m.identity),
            "reason" => text!(m.reason),
            _ => return Err("unsupported cancel field".into()),
        },
        Wire::Queue(m) => {
            if entry.context == "poll_queue.queue" {
                mutate_queue(m.task_queue.as_mut().ok_or("missing queue")?, field, entry)?;
            } else {
                match field {
                    "identity" => text!(m.identity),
                    "PollActivityTaskQueueRequest" => m.task_queue = None,
                    _ => return Err("unsupported queue field".into()),
                }
            }
        }
        Wire::Completed(m) => text!(m.identity),
        Wire::Canceled(m) => text!(m.identity),
        Wire::Failed(m) => match entry.context {
            "failed" => match field {
                "identity" => text!(m.identity),
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
        },
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
                "required" => {
                    match entry.target.omitted_field.as_deref().and_then(|v| v.rsplit('.').next()) {
                        Some("activity_type") => m.activity_type = None,
                        Some("task_queue") => m.task_queue = None,
                        _ => return Err("unknown omission".into()),
                    }
                }
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
        || request.observations.len() > generic::MAX_CASES
        || request.case_id.len() > 4096
        || request.observations.iter().any(|observation| {
            observation.case_id.len() > 4096
                || observation.predictions.len() > generic::MAX_PREDICTIONS
                || observation.details_type.len() > 512
                || observation.refusal.len() > 256
        })
    {
        return Err("probe request exceeds metadata budget".into());
    }
    let entries = inventory()?;
    let generic_entries = generic::inventory()?;
    match Operation::try_from(request.operation)? {
        Operation::Inventory => Ok(ProbeResponse {
            cases: entries
                .iter()
                .map(Entry::metadata)
                .chain(generic_entries.iter().map(|entry| entry.declaration(entries)))
                .collect::<Result<_>>()?,
            ..Default::default()
        }),
        Operation::Prepare => {
            if let Some(entry) = generic_entries.iter().find(|entry| entry.id() == request.case_id)
            {
                return entry.prepare(request, entries);
            }
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
            let mut wire = baseline(name, request, context)?.decoded()?;
            if !wire.findings()?.is_empty() {
                return Err("invalid baseline".into());
            }
            if !request.baseline {
                let entry = entry.ok_or("missing target")?;
                mutate(&mut wire, entry)?;
                let mut findings = wire.decoded()?.findings()?;
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
            let bytes = wire.bytes();
            let mut response = ProbeResponse { method: method(name), ..Default::default() };
            if bytes.len() > 512 * 1024 {
                response.recipe = Some(recipe(&bytes)?);
            } else {
                response.message = bytes;
            }
            Ok(response)
        }
        Operation::Classify => report::classify(request, entries, generic_entries),
    }
}

#[cfg(test)]
mod tests {
    use super::{Result, cached, generic, inventory};
    use std::sync::{Barrier, OnceLock};

    #[test]
    fn validated_inventories_are_reused() -> Result<()> {
        let contextual = inventory()?;
        let contextual_again = inventory()?;
        let generic = generic::inventory()?;
        let generic_again = generic::inventory()?;
        assert!(!contextual.is_empty());
        assert!(!generic.is_empty());
        assert_eq!(
            (contextual.as_ptr(), generic.as_ptr()),
            (contextual_again.as_ptr(), generic_again.as_ptr()),
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
