//! Shared Temporal envelope, payload and RPC helpers.

use umpire_dsl::Rejection;
use umpire_runtime::AnyMessage as _;
use umpire_runtime::protocol::Refusal;
pub use umpire_runtime::{callback, reply};
use umpire_temporal_api::temporal::api::{
    common::v1::Payload,
    failure::v1::{ApplicationFailureInfo, Failure, failure::FailureInfo},
};

/// A Temporal observation, optionally retaining the acknowledgment its current read reconciles.
pub struct Observed {
    current: umpire_runtime::evidence::RpcView,
    acknowledged: Option<umpire_runtime::evidence::RpcView>,
}

impl Observed {
    /// Admit ordinary evidence or a pair of unchanged actual RPC observations.
    pub fn new(value: &umpire_runtime::prost_types::Any) -> Result<Self, Refusal> {
        use umpire_runtime::{evidence::RpcView, prost_types::Any};
        use umpire_temporal_api::umpire::temporal::v1::ReconciledObservation;
        let (current, acknowledged) =
            if value.type_url == <ReconciledObservation as prost::Name>::type_url() {
                let pair: ReconciledObservation = value.unpack()?;
                let acknowledged = RpcView::new(&Any::required(pair.acknowledged)?)?;
                let current = RpcView::new(&Any::required(pair.current)?)?;
                (current, Some(acknowledged))
            } else {
                (RpcView::new(value)?, None)
            };
        Ok(Self { current, acknowledged })
    }

    /// The current actual RPC observation used for dispatch and projection.
    pub fn current(&self) -> &umpire_runtime::realization::Observation {
        self.current.raw()
    }

    /// The earlier actual acknowledgment retained by compound evidence.
    pub fn acknowledged(&self) -> Option<&umpire_runtime::realization::Observation> {
        self.acknowledged.as_ref().map(umpire_runtime::evidence::RpcView::raw)
    }

    /// The typed actual invocation used for dispatch and projection.
    pub fn rpc(&self) -> &umpire_runtime::evidence::RpcView {
        &self.current
    }

    /// The typed earlier acknowledgment, when supplied by the transport.
    pub fn acknowledged_rpc(&self) -> Option<&umpire_runtime::evidence::RpcView> {
        self.acknowledged.as_ref()
    }
}

impl std::ops::Deref for Observed {
    type Target = umpire_runtime::Observed;
    fn deref(&self) -> &Self::Target {
        &self.current
    }
}

/// Translate the supported gRPC statuses into Model rejections.
pub fn rejection(code: i32) -> Option<Rejection> {
    match code {
        5 => Some(Rejection::NotFound),
        6 => Some(Rejection::AlreadyExists),
        9 => Some(Rejection::FailedPrecondition),
        3 => Some(Rejection::InvalidArgument),
        7 => Some(Rejection::PermissionDenied),
        14 => Some(Rejection::Unavailable),
        _ => None,
    }
}

/// A protobuf service's canonical identity, independent of its message names.
pub trait Service {
    /// Fully qualified protobuf service name.
    const NAME: &'static str;
}

/// Temporal's public workflow service.
pub struct WorkflowService;

impl Service for WorkflowService {
    const NAME: &'static str = "temporal.api.workflowservice.v1.WorkflowService";
}

/// Name an RPC on the explicitly selected service.
pub fn method<S: Service>(name: &str) -> String {
    format!("/{}/{name}", S::NAME)
}

/// Read the bytes of a payload carrying only the binary/plain encoding metadata.
pub fn plain_text(payload: &Payload) -> Result<&[u8], Refusal> {
    let encoding = payload
        .metadata
        .get("encoding")
        .ok_or_else(|| Refusal::malformed("payload encoding is missing"))?;
    if payload.metadata.len() != 1 {
        return Err(Refusal::malformed("payload metadata must contain only encoding"));
    }
    if encoding != b"binary/plain" {
        return Err(Refusal::unsupported("unsupported payload encoding"));
    }
    Ok(&payload.data)
}

/// Construct application failures on the upstream generated type.
pub trait ApplicationFailure {
    /// Create an application failure with an explicit retry policy.
    fn application(kind: &str, message: &str, non_retryable: bool) -> Self;
}

impl ApplicationFailure for Failure {
    fn application(kind: &str, message: &str, non_retryable: bool) -> Self {
        Self {
            message: message.into(),
            failure_info: Some(FailureInfo::ApplicationFailureInfo(ApplicationFailureInfo {
                r#type: kind.into(),
                non_retryable,
                ..Default::default()
            })),
            ..Default::default()
        }
    }
}

/// Assemble a declared read schedule without choosing feature finality or polling policy.
pub fn scheduled_read(
    request: umpire_runtime::prost_types::Any,
    poll_interval_ms: u64,
    read_timeout_ms: u64,
    backoff_ms: u64,
) -> umpire_temporal_api::umpire::temporal::v1::ObserveActivity {
    umpire_temporal_api::umpire::temporal::v1::ObserveActivity {
        request: Some(request),
        poll_interval_ms,
        read_timeout_ms,
        backoff_ms,
    }
}

/// Assemble the transport instruction after the feature has selected hold and reconciliation.
pub fn instruction(
    request: umpire_runtime::prost_types::Any,
    hold_worker_polls: bool,
    not_found_read: umpire_runtime::prost_types::Any,
) -> umpire_temporal_api::umpire::temporal::v1::TemporalInstruction {
    umpire_temporal_api::umpire::temporal::v1::TemporalInstruction {
        request: Some(request),
        hold_worker_polls,
        not_found_read: Some(not_found_read),
    }
}

/// Assemble the declared worker rows without choosing keys, responses or continuation domains.
pub fn worker_program(
    namespace: String,
    task_queue: String,
    rules: Vec<umpire_temporal_api::umpire::temporal::v1::Rule>,
    continuation_domains: Vec<umpire_runtime::prost_types::Any>,
) -> umpire_temporal_api::umpire::temporal::v1::WorkerProgram {
    umpire_temporal_api::umpire::temporal::v1::WorkerProgram {
        namespace,
        task_queue,
        rules,
        continuation_domains,
    }
}
