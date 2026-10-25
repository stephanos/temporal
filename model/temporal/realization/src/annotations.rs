use crate::activity_api;
use umpire_runtime::AnyMessage as _;

#[path = "live.rs"]
mod live;
use umpire_runtime::{
    annotations::{AnnotationRequest, AnnotationResponse},
    prost_types::Any,
    protocol,
};
use umpire_temporal_api::umpire::temporal::v1::ProbeRequest;

/// A Model runtime with independent, stateless transport annotation dispatch.
pub type AnnotationRuntime = umpire_runtime::AnnotationRuntime<
    live::LiveRuntime,
    fn(&Any) -> Result<Any, protocol::Refusal>,
>;

/// Creates one independently owned run with typed transport annotation support.
pub fn annotation_runtime() -> AnnotationRuntime {
    umpire_runtime::AnnotationRuntime::new(live::LiveRuntime::new(), annotate)
}

fn annotate(input: &Any) -> Result<Any, protocol::Refusal> {
    if input.type_url == Any::pack(&ProbeRequest::default()).type_url {
        let input: ProbeRequest = input.unpack()?;
        let response = crate::probes::probe(&input)
            .map_err(|error| protocol::Refusal::malformed(error.to_string()))?;
        return Ok(Any::pack(&response));
    }
    let input: AnnotationRequest = input.unpack()?;
    let result = activity_api::annotate(&input.method, input.response, &input.message, input.seed)?;
    Ok(Any::pack(&AnnotationResponse {
        message: result.bytes,
        violations: result.violations,
        out_of_scope: result.out_of_scope.unwrap_or_default(),
    }))
}
