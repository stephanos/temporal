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
    inner: umpire_runtime::Observed,
    current: umpire_runtime::realization::Observation,
    acknowledged: Option<umpire_runtime::realization::Observation>,
}

impl Observed {
    /// Admit ordinary evidence or a pair of unchanged actual RPC observations.
    pub fn new(value: &umpire_runtime::prost_types::Any) -> Result<Self, Refusal> {
        use umpire_runtime::prost_types::Any;
        use umpire_temporal_api::umpire::temporal::v1::ReconciledObservation;
        let (current, acknowledged) =
            if value.type_url == <ReconciledObservation as prost::Name>::type_url() {
                let pair: ReconciledObservation = value.unpack()?;
                let acknowledged = Any::required(pair.acknowledged)?;
                let _ = umpire_runtime::Observed::new(&acknowledged)?;
                (Any::required(pair.current)?, Some(acknowledged.unpack()?))
            } else {
                (value.clone(), None)
            };
        Ok(Self {
            inner: umpire_runtime::Observed::new(&current)?,
            current: current.unpack()?,
            acknowledged,
        })
    }

    /// The current actual RPC observation used for dispatch and projection.
    pub fn current(&self) -> &umpire_runtime::realization::Observation {
        &self.current
    }

    /// The earlier actual acknowledgment retained by compound evidence.
    pub fn acknowledged(&self) -> Option<&umpire_runtime::realization::Observation> {
        self.acknowledged.as_ref()
    }
}

impl std::ops::Deref for Observed {
    type Target = umpire_runtime::Observed;
    fn deref(&self) -> &Self::Target {
        &self.inner
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
