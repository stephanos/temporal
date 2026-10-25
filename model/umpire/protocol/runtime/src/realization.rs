use crate::AnyMessage as _;
use crate::{
    prost_types::Any,
    protocol::Refusal,
    realization::{Callback, Observation, Reply, observation::Result as ObservationResult},
};
use prost::Name;

/// An admitted observation whose result is decoded after selecting the RPC.
pub struct Observed {
    /// Fully qualified RPC method path.
    pub method: String,
    result: Option<ObservationResult>,
    request: Option<Any>,
}

impl Observed {
    /// Admit the outer envelope and require its method, retaining result precedence.
    pub fn new(value: &Any) -> Result<Self, Refusal> {
        let observation: Observation = value.unpack()?;
        if observation.method.is_empty() {
            return Err(Refusal::malformed("observation method is missing"));
        }
        Ok(Self {
            method: observation.method,
            result: observation.result,
            request: observation.request,
        })
    }

    /// Decode the same invocation's actual request, or `None` when no owner captured it.
    pub fn request<T: Name + Default>(&self) -> Result<Option<T>, Refusal> {
        self.request.as_ref().map(Any::unpack).transpose()
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

/// Pack a typed request into the reply envelope.
pub fn reply(request: &impl Name) -> Any {
    Any::pack(&Reply { request: Some(Any::pack(request)) })
}

/// Admit a callback envelope and decode its required typed task.
pub fn callback<T: Name + Default>(value: &Any) -> Result<T, Refusal> {
    let callback: Callback = value.unpack()?;
    Any::required(callback.task)?.unpack()
}
