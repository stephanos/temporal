//! Activity RPC transport and nested catalog validation.

use super::{MetadataEntry, catalog, worker_failure_presence};
use prost::Message;
use umpire_annotations::{Catalog, Expectation, FieldValue, Kind, Rng, validate};
use umpire_temporal_api::temporal::api::{
    activity::v1::{ActivityExecutionInfo, ActivityExecutionOutcome, activity_execution_outcome},
    common::v1::{ActivityType, Payload, Payloads, RetryPolicy, WorkflowExecution, WorkflowType},
    failure::v1::{
        ActivityFailureInfo, ApplicationFailureInfo, CanceledFailureInfo,
        ChildWorkflowExecutionFailureInfo, Failure, NexusHandlerFailureInfo,
        NexusOperationFailureInfo, ResetWorkflowFailureInfo, ServerFailureInfo,
        TerminatedFailureInfo, TimeoutFailureInfo, failure,
    },
    taskqueue::v1::TaskQueue,
    workflowservice::v1::{
        DescribeActivityExecutionRequest, DescribeActivityExecutionResponse,
        PollActivityExecutionRequest, PollActivityExecutionResponse, PollActivityTaskQueueRequest,
        PollActivityTaskQueueResponse, RequestCancelActivityExecutionRequest,
        RequestCancelActivityExecutionResponse, RespondActivityTaskCanceledRequest,
        RespondActivityTaskCanceledResponse, RespondActivityTaskCompletedRequest,
        RespondActivityTaskCompletedResponse, RespondActivityTaskFailedRequest,
        RespondActivityTaskFailedResponse, StartActivityExecutionRequest,
        StartActivityExecutionResponse,
    },
};
use umpire_wasm::{
    google::protobuf::{Duration as ProtoDuration, Timestamp as ProtoTimestamp},
    protocol::Refusal,
};

/// Filled request bytes or untouched response bytes, with value-free findings.
pub struct AnnotationResult {
    /// Protobuf bytes to transport; response bytes are always unchanged.
    pub bytes: Vec<u8>,
    /// Field paths and violated constraint names, never field contents.
    pub violations: Vec<String>,
    /// Explicit limitation for a classified RPC outside this activity Model.
    pub out_of_scope: Option<String>,
}

/// Fills only absent request IDs and validates a classified activity RPC.
///
/// Request constraints declared `Accepts` retain probe metadata but do not prevent
/// transport of values accepted by the server. Responses retain every finding
/// and their original bytes. This operation never reads or mutates Judge state.
///
/// # Errors
/// Refuses unknown methods, malformed protobuf bytes and catalog admission failures.
pub fn annotate(
    method: &str,
    response: bool,
    bytes: &[u8],
    seed: u64,
) -> Result<AnnotationResult, Refusal> {
    let method = method
        .strip_prefix("/temporal.api.workflowservice.v1.WorkflowService/")
        .or_else(|| (!method.contains('/') && !method.contains('.')).then_some(method))
        .ok_or_else(|| Refusal::unsupported("annotation service is not classified"))?;
    let full = format!("temporal.api.workflowservice.v1.WorkflowService.{method}");
    let (_, disposition, reason) = catalog::RPC_BINDINGS
        .iter()
        .find(|(name, _, _)| *name == full)
        .ok_or_else(|| Refusal::unsupported("annotation RPC is not classified"))?;
    if *disposition == "OutOfScope" {
        return Ok(AnnotationResult {
            bytes: bytes.to_vec(),
            violations: Vec::new(),
            out_of_scope: Some((*reason).into()),
        });
    }
    match (method, response) {
        ("StartActivityExecution", false) => process::<StartActivityExecutionRequest>(
            bytes,
            false,
            seed,
            Some(|message, seed| fill_request_id(&mut message.request_id, seed)),
        ),
        ("StartActivityExecution", true) => {
            process::<StartActivityExecutionResponse>(bytes, true, seed, None)
        }
        ("DescribeActivityExecution", false) => {
            process::<DescribeActivityExecutionRequest>(bytes, false, seed, None)
        }
        ("DescribeActivityExecution", true) => {
            process::<DescribeActivityExecutionResponse>(bytes, true, seed, None)
        }
        ("RequestCancelActivityExecution", false) => {
            process::<RequestCancelActivityExecutionRequest>(
                bytes,
                false,
                seed,
                Some(|message, seed| fill_request_id(&mut message.request_id, seed)),
            )
        }
        ("RequestCancelActivityExecution", true) => {
            process::<RequestCancelActivityExecutionResponse>(bytes, true, seed, None)
        }
        ("PollActivityTaskQueue", false) => {
            process::<PollActivityTaskQueueRequest>(bytes, false, seed, None)
        }
        ("PollActivityTaskQueue", true) => {
            process::<PollActivityTaskQueueResponse>(bytes, true, seed, None)
        }
        ("RespondActivityTaskCompleted", false) => {
            process::<RespondActivityTaskCompletedRequest>(bytes, false, seed, None)
        }
        ("RespondActivityTaskCompleted", true) => {
            process::<RespondActivityTaskCompletedResponse>(bytes, true, seed, None)
        }
        ("RespondActivityTaskFailed", false) => {
            process::<RespondActivityTaskFailedRequest>(bytes, false, seed, None)
        }
        ("RespondActivityTaskFailed", true) => {
            process::<RespondActivityTaskFailedResponse>(bytes, true, seed, None)
        }
        ("RespondActivityTaskCanceled", false) => {
            process::<RespondActivityTaskCanceledRequest>(bytes, false, seed, None)
        }
        ("RespondActivityTaskCanceled", true) => {
            process::<RespondActivityTaskCanceledResponse>(bytes, true, seed, None)
        }
        ("PollActivityExecution", false) => {
            process::<PollActivityExecutionRequest>(bytes, false, seed, None)
        }
        ("PollActivityExecution", true) => {
            process::<PollActivityExecutionResponse>(bytes, true, seed, None)
        }
        _ => Err(Refusal::unsupported("annotation RPC has no typed transport catalog")),
    }
}

fn fill_request_id(value: &mut String, seed: u64) -> Result<bool, Refusal> {
    if !value.is_empty() {
        return Ok(false);
    }
    *value = String::generate(&Kind::Uuid, &mut Rng::new(seed))
        .map_err(|_| Refusal::unsupported("request ID generation failed"))?;
    Ok(true)
}

type Fill<M> = fn(&mut M, u64) -> Result<bool, Refusal>;

fn process<M: Message + Default + Inspect>(
    bytes: &[u8],
    response: bool,
    seed: u64,
    fill: Option<Fill<M>>,
) -> Result<AnnotationResult, Refusal> {
    let mut message = M::decode(bytes)
        .map_err(|_| Refusal::malformed("annotation protobuf could not be decoded"))?;
    let changed = if let Some(fill) = fill { fill(&mut message, seed)? } else { false };
    let mut violations = Vec::new();
    message.inspect_wire(response, &mut violations)?;
    violations.sort();
    violations.dedup();
    Ok(AnnotationResult {
        bytes: if changed { message.encode_to_vec() } else { bytes.to_vec() },
        violations,
        out_of_scope: None,
    })
}

fn findings<M: 'static>(
    message: &M,
    catalog: Result<Catalog<M>, umpire_annotations::AdmissionErrors>,
    response: bool,
    output: &mut Vec<String>,
) -> Result<(), Refusal> {
    let catalog = catalog
        .map_err(|_| Refusal::unsupported("activity annotation catalog admission failed"))?;
    for violation in validate(message, &catalog) {
        if !response
            && violation
                .expected
                .as_ref()
                .is_some_and(|expected| matches!(expected.expectation, Expectation::Accepts))
        {
            continue;
        }
        output.push(violation.to_string());
    }
    Ok(())
}

trait Inspect {
    fn inspect_wire(&self, response: bool, output: &mut Vec<String>) -> Result<(), Refusal>;
}
impl<T: Inspect> Inspect for Option<T> {
    fn inspect_wire(&self, response: bool, output: &mut Vec<String>) -> Result<(), Refusal> {
        if let Some(value) = self {
            value.inspect_wire(response, output)?;
        }
        Ok(())
    }
}
impl<T: Inspect> Inspect for Vec<T> {
    fn inspect_wire(&self, response: bool, output: &mut Vec<String>) -> Result<(), Refusal> {
        for value in self {
            value.inspect_wire(response, output)?;
        }
        Ok(())
    }
}
impl<T: Inspect> Inspect for Box<T> {
    fn inspect_wire(&self, response: bool, output: &mut Vec<String>) -> Result<(), Refusal> {
        self.as_ref().inspect_wire(response, output)
    }
}

macro_rules! inspect {
    ($ty:ty, $factory:ident, |$message:ident, $response:ident, $output:ident| $body:block) => {
        impl Inspect for $ty {
            fn inspect_wire(&$message, $response: bool, $output: &mut Vec<String>) -> Result<(), Refusal> {
                findings($message, catalog::$factory(), $response, $output)?;
                $body
                Ok(())
            }
        }
    };
}
inspect!(ActivityExecutionOutcome, activity_execution_outcome, |self, response, output| {
    match &self.value {
        Some(activity_execution_outcome::Value::Result(value)) => {
            value.inspect_wire(response, output)?;
        }
        Some(activity_execution_outcome::Value::Failure(value)) => {
            value.inspect_wire(response, output)?;
        }
        None => {}
    }
});
inspect!(ActivityExecutionInfo, activity_execution_info, |self, response, output| {
    self.activity_type.inspect_wire(response, output)?;
    self.heartbeat_details.inspect_wire(response, output)?;
    self.last_failure.inspect_wire(response, output)?;
    self.retry_policy.inspect_wire(response, output)?;
});
inspect!(Payloads, payloads, |self, response, output| {
    self.payloads.inspect_wire(response, output)?;
});
inspect!(Payload, payload, |self, response, output| {
    for (key, value) in &self.metadata {
        MetadataEntry { key: key.clone(), value: value.clone() }.inspect_wire(response, output)?;
    }
});
inspect!(ActivityType, activity_type, |self, response, output| {});
inspect!(RetryPolicy, retry_policy, |self, response, output| {
    if response {
        for (field, value) in [
            ("initial_interval", &self.initial_interval),
            ("maximum_interval", &self.maximum_interval),
        ] {
            if let Some(value) = raw_duration_view(value.as_ref()) {
                for constraint in
                    value.violations(&Kind::Duration(umpire_annotations::NumberRules::default()))
                {
                    output
                        .push(format!("temporal.api.common.v1.RetryPolicy.{field}: {constraint}"));
                }
            }
        }
    }
});
inspect!(ApplicationFailureInfo, application_failure_info, |self, response, output| {
    self.details.inspect_wire(response, output)?;
});
inspect!(Failure, failure, |self, response, output| {
    self.encoded_attributes.inspect_wire(response, output)?;
    self.cause.inspect_wire(response, output)?;
    match &self.failure_info {
        Some(failure::FailureInfo::ApplicationFailureInfo(value)) => {
            value.inspect_wire(response, output)?;
        }
        Some(failure::FailureInfo::TimeoutFailureInfo(value)) => {
            value.inspect_wire(response, output)?;
        }
        Some(failure::FailureInfo::CanceledFailureInfo(value)) => {
            value.inspect_wire(response, output)?;
        }
        Some(failure::FailureInfo::TerminatedFailureInfo(value)) => {
            value.inspect_wire(response, output)?;
        }
        Some(failure::FailureInfo::ServerFailureInfo(value)) => {
            value.inspect_wire(response, output)?;
        }
        Some(failure::FailureInfo::ResetWorkflowFailureInfo(value)) => {
            value.inspect_wire(response, output)?;
        }
        Some(failure::FailureInfo::ActivityFailureInfo(value)) => {
            value.inspect_wire(response, output)?;
        }
        Some(failure::FailureInfo::ChildWorkflowExecutionFailureInfo(value)) => {
            value.inspect_wire(response, output)?;
        }
        Some(failure::FailureInfo::NexusOperationExecutionFailureInfo(value)) => {
            value.inspect_wire(response, output)?;
        }
        Some(failure::FailureInfo::NexusHandlerFailureInfo(value)) => {
            value.inspect_wire(response, output)?;
        }
        None => {}
    }
});
inspect!(TaskQueue, task_queue, |self, response, output| {});
inspect!(
    PollActivityTaskQueueRequest,
    poll_activity_task_queue_request,
    |self, response, output| {
        self.task_queue.inspect_wire(response, output)?;
    }
);
inspect!(
    PollActivityTaskQueueResponse,
    poll_activity_task_queue_response,
    |self, response, output| {
        self.activity_type.inspect_wire(response, output)?;
        self.input.inspect_wire(response, output)?;
        self.heartbeat_details.inspect_wire(response, output)?;
        self.retry_policy.inspect_wire(response, output)?;
    }
);
inspect!(
    RespondActivityTaskCompletedRequest,
    respond_activity_task_completed_request,
    |self, response, output| {
        self.result.inspect_wire(response, output)?;
    }
);
inspect!(
    RespondActivityTaskCompletedResponse,
    respond_activity_task_completed_response,
    |self, response, output| {}
);
inspect!(
    RespondActivityTaskFailedRequest,
    respond_activity_task_failed_request,
    |self, response, output| {
        if let Some(failure) = &self.failure {
            findings(failure, worker_failure_presence(), response, output)?;
        }
        self.failure.inspect_wire(response, output)?;
        self.last_heartbeat_details.inspect_wire(response, output)?;
    }
);
inspect!(
    RespondActivityTaskFailedResponse,
    respond_activity_task_failed_response,
    |self, response, output| {
        self.failures.inspect_wire(response, output)?;
    }
);
inspect!(
    RespondActivityTaskCanceledRequest,
    respond_activity_task_canceled_request,
    |self, response, output| {
        self.details.inspect_wire(response, output)?;
    }
);
inspect!(
    RespondActivityTaskCanceledResponse,
    respond_activity_task_canceled_response,
    |self, response, output| {}
);
inspect!(
    StartActivityExecutionRequest,
    start_activity_execution_request,
    |self, response, output| {
        self.activity_type.inspect_wire(response, output)?;
        self.task_queue.inspect_wire(response, output)?;
        self.retry_policy.inspect_wire(response, output)?;
        self.input.inspect_wire(response, output)?;
    }
);
inspect!(
    StartActivityExecutionResponse,
    start_activity_execution_response,
    |self, response, output| {}
);
inspect!(
    DescribeActivityExecutionRequest,
    describe_activity_execution_request,
    |self, response, output| {}
);
inspect!(
    DescribeActivityExecutionResponse,
    describe_activity_execution_response,
    |self, response, output| {
        self.info.inspect_wire(response, output)?;
        self.input.inspect_wire(response, output)?;
        self.outcome.inspect_wire(response, output)?;
    }
);
inspect!(
    PollActivityExecutionRequest,
    poll_activity_execution_request,
    |self, response, output| {}
);
inspect!(
    PollActivityExecutionResponse,
    poll_activity_execution_response,
    |self, response, output| {
        self.outcome.inspect_wire(response, output)?;
    }
);
inspect!(
    RequestCancelActivityExecutionRequest,
    request_cancel_activity_execution_request,
    |self, response, output| {}
);
inspect!(
    RequestCancelActivityExecutionResponse,
    request_cancel_activity_execution_response,
    |self, response, output| {}
);
inspect!(TimeoutFailureInfo, timeout_failure_info, |self, response, output| {
    self.last_heartbeat_details.inspect_wire(response, output)?;
});
inspect!(CanceledFailureInfo, canceled_failure_info, |self, response, output| {
    self.details.inspect_wire(response, output)?;
});
inspect!(TerminatedFailureInfo, terminated_failure_info, |self, response, output| {});
inspect!(ServerFailureInfo, server_failure_info, |self, response, output| {});
inspect!(ResetWorkflowFailureInfo, reset_workflow_failure_info, |self, response, output| {
    self.last_heartbeat_details.inspect_wire(response, output)?;
});
inspect!(ActivityFailureInfo, activity_failure_info, |self, response, output| {
    self.activity_type.inspect_wire(response, output)?;
});
inspect!(
    ChildWorkflowExecutionFailureInfo,
    child_workflow_execution_failure_info,
    |self, response, output| {
        self.workflow_execution.inspect_wire(response, output)?;
        self.workflow_type.inspect_wire(response, output)?;
    }
);
inspect!(WorkflowExecution, workflow_execution, |self, response, output| {});
inspect!(WorkflowType, workflow_type, |self, response, output| {});
inspect!(NexusOperationFailureInfo, nexus_operation_failure_info, |self, response, output| {});
inspect!(NexusHandlerFailureInfo, nexus_handler_failure_info, |self, response, output| {});
inspect!(ProtoDuration, proto_duration, |self, response, output| {});
inspect!(ProtoTimestamp, proto_timestamp, |self, response, output| {});
inspect!(MetadataEntry, metadata_entry, |self, response, output| {});

pub(super) fn uuid_view(value: &str) -> Option<String> {
    if value.is_empty() {
        return None;
    }
    let value = if value.len() == 45
        && value.get(..9).is_some_and(|prefix| prefix.eq_ignore_ascii_case("urn:uuid:"))
    {
        &value[9..]
    } else {
        value
    };
    let value = if value.len() == 38 { value.get(1..37).unwrap_or(value) } else { value };
    if value.len() == 32 && value.bytes().all(|byte| byte.is_ascii_hexdigit()) {
        Some(format!(
            "{}-{}-{}-{}-{}",
            &value[..8],
            &value[8..12],
            &value[12..16],
            &value[16..20],
            &value[20..]
        ))
    } else {
        Some(value.to_owned())
    }
}

pub(super) fn raw_duration_view(
    value: Option<&ProtoDuration>,
) -> Option<umpire_annotations::Duration> {
    value.map(|value| umpire_annotations::Duration { seconds: value.seconds, nanos: value.nanos })
}

pub(super) fn duration_view(value: Option<&ProtoDuration>) -> Option<umpire_annotations::Duration> {
    let value = value?;
    if value.seconds == 0 && value.nanos == 0 {
        return None;
    }
    if (value.seconds > 0 && value.nanos < 0) || (value.seconds < 0 && value.nanos > 0) {
        return Some(umpire_annotations::Duration { seconds: value.seconds, nanos: value.nanos });
    }
    let total = (i128::from(value.seconds) * 1_000_000_000 + i128::from(value.nanos))
        .clamp(i128::from(i64::MIN), 3_153_600_000_000_000_000);
    Some(umpire_annotations::Duration {
        seconds: i64::try_from(total / 1_000_000_000).unwrap_or(i64::MIN),
        nanos: i32::try_from(total % 1_000_000_000).unwrap_or(0),
    })
}
