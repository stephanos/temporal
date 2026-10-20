//! Shared Temporal envelope, payload and RPC helpers.

use prost::Name;
use umpire_dsl::Rejection;
use umpire_temporal_api::{
    temporal::api::{
        common::v1::Payload,
        failure::v1::{ApplicationFailureInfo, Failure, failure::FailureInfo},
    },
    umpire::temporal::v1::{
        Callback, Observation, Reply, observation::Result as ObservationResult,
    },
};
use umpire_wasm::{google::protobuf::Any, protocol::Refusal};

/// An admitted observation whose result is decoded after selecting the RPC.
pub struct Observed {
    /// Fully qualified RPC method path.
    pub method: String,
    result: Option<ObservationResult>,
}

impl Observed {
    /// Admit the outer envelope and require its method, retaining result precedence.
    pub fn new(value: &Any) -> Result<Self, Refusal> {
        let observation: Observation = value.unpack()?;
        if observation.method.is_empty() {
            return Err(Refusal::malformed("observation method is missing"));
        }
        Ok(Self { method: observation.method, result: observation.result })
    }

    /// Decode a successful typed response; errors and absent results are malformed.
    pub fn response<T: Name + Default>(&self) -> Result<T, Refusal> {
        let Some(ObservationResult::Response(response)) = &self.result else {
            return Err(Refusal::malformed("observation response is missing"));
        };
        response.unpack()
    }

    /// Return the gRPC error code, or `None` for a response; require a result.
    pub fn code(&self) -> Result<Option<i32>, Refusal> {
        match &self.result {
            Some(ObservationResult::Error(error)) => Ok(Some(error.code)),
            Some(ObservationResult::Response(_)) => Ok(None),
            None => Err(Refusal::malformed("observation result is missing")),
        }
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

/// Pack a typed request into the Temporal reply envelope.
pub fn reply(request: &impl Name) -> Any {
    Any::pack(&Reply { request: Some(Any::pack(request)) })
}

/// Admit a callback envelope and decode its required typed task.
pub fn callback<T: Name + Default>(value: &Any) -> Result<T, Refusal> {
    let callback: Callback = value.unpack()?;
    Any::required(callback.task)?.unpack()
}
